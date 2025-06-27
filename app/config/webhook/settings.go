// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package config contains the configuration for the application.
package config

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/cloudzero/cloudzero-agent/app/utils/scout"
	"github.com/microcosm-cc/bluemonday"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"

	"github.com/ilyakaznacheev/cleanenv"
)

// Settings represents the configuration settings for the application.
type Settings struct {
	CloudAccountID    string      `yaml:"cloud_account_id" env:"CLOUD_ACCOUNT_ID" env-description:"CSP account ID"`
	Region            string      `yaml:"region" env:"CSP_REGION" env-description:"cloud service provider region"`
	ClusterName       string      `yaml:"cluster_name" env:"CLUSTER_NAME" env-description:"name of the cluster to monitor"`
	Destination       string      `yaml:"destination" env:"DESTINATION" env-default:"https://api.cloudzero.com/v1/container-metrics" env-description:"location to send metrics to"`
	APIKeyPath        string      `yaml:"api_key_path" env:"API_KEY_PATH" env-description:"path to the API key file"`
	Server            Server      `yaml:"server"`
	Certificate       Certificate `yaml:"certificate"`
	Logging           Logging     `yaml:"logging"`
	Database          Database    `yaml:"database"`
	Filters           Filters     `yaml:"filters"`
	RemoteWrite       RemoteWrite `yaml:"remote_write"`
	K8sClient         K8sClient   `yaml:"k8s_client"`
	LabelMatches      []regexp.Regexp
	AnnotationMatches []regexp.Regexp

	// control for dynamic reloading
	mu sync.Mutex
}

type RemoteWrite struct {
	apiKey          string
	Host            string
	MaxBytesPerSend int           `yaml:"max_bytes_per_send" default:"10000000" env:"MAX_BYTES_PER_SEND" env-description:"maximum bytes to send in a single request"`
	SendInterval    time.Duration `yaml:"send_interval" default:"60s" env:"SEND_INTERVAL" env-description:"interval in seconds to send data"`
	SendTimeout     time.Duration `yaml:"send_timeout" default:"30s" env:"SEND_TIMEOUT" env-description:"timeout in seconds to send data"`
	MaxRetries      int           `yaml:"max_retries" default:"3" env:"MAX_RETRIES" env-description:"maximum number of retries"`
}

func NewSettings(configFiles ...string) (*Settings, error) {
	var cfg Settings

	// do not allow empty arrays
	if configFiles == nil {
		return nil, errors.New("the config files slice cannot be nil")
	}

	for _, cfgFile := range configFiles {
		if cfgFile == "" {
			continue
		}

		if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
			return nil, fmt.Errorf("no config %s: %w", cfgFile, err)
		}

		err := cleanenv.ReadConfig(cfgFile, &cfg)
		if err != nil {
			return nil, fmt.Errorf("config read %s: %w", cfgFile, err)
		}
	}

	// clean unexpected characters from CloudAccountID
	// should only be A-Z, a-z, 0-9 at beginning and end
	cfg.CloudAccountID = cleanString(cfg.CloudAccountID)
	cfg.Region = strings.TrimSpace(cfg.Region)

	// Auto-detect cloud account ID and region if needed
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := scout.DetectConfiguration(ctx, nil, &cfg.Region, &cfg.CloudAccountID)
	if err != nil {
		return nil, fmt.Errorf("failed to auto-detect cloud environment: %w", err)
	}

	cfg.setCompiledFilters()

	if err := cfg.SetAPIKey(); err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	cfg.setRemoteWriteURL()
	cfg.setPolicy()

	setLoggingOptions(&cfg.Logging)

	return &cfg, nil
}

func (s *Settings) GetAPIKey() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.RemoteWrite.apiKey
}

func (s *Settings) SetAPIKey() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	apiKeyPathLocation, err := absFilePath(s.APIKeyPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	if _, err = os.Stat(apiKeyPathLocation); os.IsNotExist(err) {
		return fmt.Errorf("API key file %s not found: %w", apiKeyPathLocation, err)
	}
	apiKey, err := os.ReadFile(s.APIKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read API key: %w", err)
	}
	s.RemoteWrite.apiKey = strings.TrimSpace(string(apiKey))

	if len(s.RemoteWrite.apiKey) == 0 {
		return errors.New("API key is empty")
	}
	return nil
}

func (s *Settings) setRemoteWriteURL() {
	if s.Destination == "" {
		s.Destination = "https://api.cloudzero.com/v1/container-metrics"
	}
	baseURL, err := url.Parse(s.Destination)
	if err != nil {
		fmt.Println("Malformed URL: ", err.Error())
		return
	}
	params := url.Values{}
	params.Add("cluster_name", s.ClusterName)
	params.Add("cloud_account_id", s.CloudAccountID)
	params.Add("region", s.Region)
	baseURL.RawQuery = params.Encode()
	url := baseURL.String()

	if !isValidURL(url) {
		log.Fatal().Str("url", url).Msg("URL format invalid")
	}
	s.RemoteWrite.Host = url
}

func isValidURL(uri string) bool {
	if _, err := url.ParseRequestURI(uri); err != nil {
		return false
	}
	return true
}

func (s *Settings) setPolicy() {
	s.Filters.Policy = *bluemonday.StrictPolicy()
}

func (s *Settings) setCompiledFilters() {
	s.LabelMatches = s.compilePatterns(s.Filters.Labels.Patterns)
	s.AnnotationMatches = s.compilePatterns(s.Filters.Annotations.Patterns)
}

func (s *Settings) compilePatterns(patterns []string) []regexp.Regexp {
	errHistory := []error{}
	compiledPatterns := []regexp.Regexp{}

	for _, pattern := range patterns {
		compiled, err := regexp.Compile(pattern)
		if err != nil {
			errHistory = append(errHistory, err)
		} else {
			compiledPatterns = append(compiledPatterns, *compiled)
		}
	}
	if len(errHistory) > 0 {
		for _, err := range errHistory {
			log.Info().Err(err).Msg("invalid regex pattern")
		}
		log.Fatal().Msg("Config file contains invalid regex patterns")
	}
	return compiledPatterns
}

func absFilePath(location string) (string, error) {
	dir := filepath.Dir(filepath.Clean(location))
	// validate path if not local directory
	if dir == "" || strings.HasPrefix(dir, ".") {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get working directory: %w", err)
		}
		location = filepath.Clean(filepath.Join(wd, location))
	}
	return location, nil
}

// Files is a custom flag type to handle multiple configuration files
type Files []string

func (c *Files) String() string {
	return strings.Join(*c, ",")
}

// Set appends a new configuration file to the Files
func (c *Files) Set(value string) error {
	*c = append(*c, value)
	return nil
}

// cleanString trims non-alphanumeric characters from the beginning and end of a
// string.
//
// The resulting string should have an alphanumeric character at the beginning
// and end. If not alphanumeric characters are found, return an empty string.
func cleanString(s string) string {
	// Find first alphanumeric character from start
	start := -1
	for i, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			start = i
			break
		}
	}

	// if no alphanumeric characters found, return empty string
	if start < 0 {
		return ""
	}

	// Find last alphanumeric character from end
	end := len(s)
	for i := len(s) - 1; i >= 0; i-- {
		c := s[i]
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			end = i + 1
			break
		}
	}

	return s[start:end]
}

func (s *Settings) ToYAML() ([]byte, error) {
	raw, err := yaml.Marshal(s)
	if err != nil {
		return nil, fmt.Errorf("failed to encode into yaml: %w", err)
	}
	return raw, nil
}

// ToBytes returns a serialized representation of the data in the class
func (s *Settings) ToBytes() ([]byte, error) {
	return s.ToYAML()
}
