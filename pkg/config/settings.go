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
	LabelMatches      []regexp.Regexp
	AnnotationMatches []regexp.Regexp
	CloudZero         CloudZero
}

type CloudZero struct {
	APIKey string
	Host   string
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
	cfg.setCompiledFilters()
	cfg.getAPIKey()
	cfg.setRemoteWriteURL()
	cfg.setPolicy()
	return &cfg, nil
}

func (s *Settings) getAPIKey() {

	apiKeyPathLocation, err := absFilePath(s.APIKeyPath)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get API key file path")
	}

	if _, err := os.Stat(apiKeyPathLocation); os.IsNotExist(err) {
		log.Fatal().Err(err).Msg("API key file does not exist")
	}
	apiKey, err := os.ReadFile(s.APIKeyPath)
	if err != nil {
		log.Err(err).Msg("Failed to read API key")
	}
	s.CloudZero.APIKey = string(apiKey)
}

func (s *Settings) setRemoteWriteURL() {
	s.CloudZero.Host = fmt.Sprintf("https://%s/v1/container-metrics?cluster_name=%s&cloud_account_id=%s&region=%s", s.Host, s.ClusterName, s.CloudAccountID, s.Region)
	if s.Host == "" {
		log.Fatal().Msg("Host is required")
	}
	if !isValidURL(s.CloudZero.Host) {
		log.Fatal().Msgf("URL format invalid: %s", s.Host)
	}
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

func isValidURL(uri string) bool {
	if _, err := url.ParseRequestURI(uri); err != nil {
		return false
	}
	return true
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
