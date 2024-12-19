// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package config

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"github.com/rs/zerolog/log"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/pkg/errors"
)

// Settings represents the configuration settings for the application.
type Settings struct {
	CloudAccountID    string      `yaml:"cloud_account_id" env:"CLOUD_ACCOUNT_ID" env-description:"CSP account ID"`
	Region            string      `yaml:"region" env:"CSP_REGION" env-description:"cloud service provider region"`
	ClusterName       string      `yaml:"cluster_name" env:"CLUSTER_NAME" env-description:"name of the cluster to monitor"`
	Host              string      `yaml:"host" env:"HOST" default:"api.cloudzero.com" env-description:"host to send metrics to"`
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
	SendTimeout     time.Duration `yaml:"send_timeout" default:"10s" env:"SEND_TIMEOUT" env-description:"timeout in seconds to send data"`
	MaxRetries      int           `yaml:"max_retries" default:"3" env:"MAX_RETRIES" env-description:"maximum number of retries"`
}

func NewSettings(configFiles ...string) (*Settings, error) {
	var cfg Settings
	for _, cfgFile := range configFiles {
		if cfgFile == "" {
			continue
		}

		if _, err := os.Stat(cfgFile); os.IsNotExist(err) {
			return nil, errors.Wrap(err, fmt.Sprintf("no config %s", cfgFile))
		}

		err := cleanenv.ReadConfig(cfgFile, &cfg)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("config read %s", cfgFile))
		}
	}

	// clean unexpected characters from CloudAccountID
	// should only be A-Z, a-z, 0-9 at beginning and end
	cfg.CloudAccountID = cleanString(cfg.CloudAccountID)

	cfg.setCompiledFilters()

	if err := cfg.SetAPIKey(); err != nil {
		return nil, errors.Wrap(err, "failed to get API key")
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
		return errors.Wrap(err, "failed to get absolute path")
	}

	if _, err := os.Stat(apiKeyPathLocation); os.IsNotExist(err) {
		return errors.Wrap(err, fmt.Sprintf("API key file %s not found", apiKeyPathLocation))
	}
	apiKey, err := os.ReadFile(s.APIKeyPath)
	if err != nil {
		return errors.Wrap(err, "failed to read API key")
	}
	s.RemoteWrite.apiKey = strings.TrimSpace(string(apiKey))

	if len(s.RemoteWrite.apiKey) == 0 {
		return errors.New("API key is empty")
	}
	return nil
}

func (s *Settings) setRemoteWriteURL() {
	if s.Host == "" {
		log.Fatal().Msg("Host is required")
	}
	baseURL, err := url.Parse(fmt.Sprintf("https://%s", s.Host))
	if err != nil {
		fmt.Println("Malformed URL: ", err.Error())
		return
	}
	baseURL.Path += "/v1/container-metrics"
	params := url.Values{}
	params.Add("cluster_name", s.ClusterName)
	params.Add("cloud_account_id", s.CloudAccountID)
	params.Add("region", s.Region)
	baseURL.RawQuery = params.Encode()
	url := baseURL.String()

	if !isValidURL(url) {
		log.Fatal().Msgf("URL format invalid: %s", url)
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
			log.Info().Err(err).Msgf("invalid regex pattern: %v", err)
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
			return "", errors.Wrap(err, "working directory")
		}
		location = filepath.Clean(filepath.Join(wd, location))
	}
	return location, nil
}

// ConfigFiles is a custom flag type to handle multiple configuration files
type Files []string

func (c *Files) String() string {
	return strings.Join(*c, ",")
}

// appends a new configuration file to the ConfigFiles
func (c *Files) Set(value string) error {
	*c = append(*c, value)
	return nil
}

func cleanString(s string) string {
	// clean unexpected characters from CloudAccountID
	// should only be A-Z, a-z, 0-9 at beginning and end
	s = strings.TrimSpace(s)
	return strings.Trim(s, "\"'")
}
