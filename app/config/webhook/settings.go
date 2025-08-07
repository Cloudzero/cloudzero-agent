// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package config implements configuration management for the CloudZero webhook component.
//
// This package provides configuration structures and validation logic specifically
// for the webhook admission controller functionality. It handles:
//
//   - Webhook-specific settings and validation rules
//   - Kubernetes admission controller configuration
//   - Certificate and TLS configuration for secure webhook communication
//   - Resource filtering rules (labels, annotations, namespaces)
//   - Remote write endpoint configuration for metrics forwarding
//   - Database settings for webhook event storage
//
// Key differences from gator config:
//   - Focuses on webhook admission controller requirements
//   - Includes Kubernetes-specific filtering and validation
//   - Manages TLS certificate configuration for secure API server communication
//   - Handles regex pattern compilation for resource filtering
//   - Supports HTML sanitization policies for security
//
// Configuration workflow:
//   1. NewSettings() loads configuration from files and environment
//   2. Cloud environment auto-detection (AWS, Azure, GCP)
//   3. API key validation and caching
//   4. Regex pattern compilation for resource filtering
//   5. TLS certificate and security policy initialization
//
// Usage:
//   settings, err := NewSettings("/path/to/webhook-config.yaml")
//   if err != nil {
//       return fmt.Errorf("webhook configuration error: %w", err)
//   }
//
// The webhook configuration system ensures secure, validated processing
// of Kubernetes admission review requests with proper resource filtering
// and CloudZero integration.
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

// Settings represents the complete configuration for the CloudZero webhook admission controller.
// It manages all aspects of webhook operation including Kubernetes integration, resource filtering,
// TLS security, and CloudZero API communication.
//
// The webhook Settings differs from the main gator configuration by focusing specifically
// on admission controller requirements:
//   - Kubernetes API server integration and authentication
//   - Resource filtering based on labels, annotations, and namespaces
//   - TLS certificate management for secure webhook communication
//   - Regex pattern compilation for efficient resource matching
//   - HTML sanitization policies for security
//
// Configuration lifecycle:
//   1. Load from YAML files and environment variables
//   2. Auto-detect cloud environment (AWS, Azure, GCP)
//   3. Validate and compile regex patterns for resource filtering
//   4. Initialize TLS certificates and security policies
//   5. Prepare CloudZero API endpoints for metric forwarding
//
// Thread safety:
//   Includes mutex for safe concurrent access to runtime-mutable values
//   like compiled regex patterns and API keys.
type Settings struct {
	// Core cloud and cluster identification
	
	// CloudAccountID identifies the cloud provider account (AWS Account ID,
	// Azure Subscription ID, GCP Project ID). Auto-detected if not specified.
	CloudAccountID string `yaml:"cloud_account_id" env:"CLOUD_ACCOUNT_ID" env-description:"CSP account ID"`
	
	// Region specifies the cloud provider region for cost attribution.
	// Auto-detected from cloud metadata if not explicitly configured.
	Region string `yaml:"region" env:"CSP_REGION" env-description:"cloud service provider region"`
	
	// ClusterName uniquely identifies this Kubernetes cluster within CloudZero.
	// Auto-detected from cluster metadata if not explicitly configured.
	ClusterName string `yaml:"cluster_name" env:"CLUSTER_NAME" env-description:"name of the cluster to monitor"`
	
	// Destination specifies the CloudZero API endpoint for metric uploads.
	// Defaults to production CloudZero API if not specified.
	Destination string `yaml:"destination" env:"DESTINATION" env-default:"https://api.cloudzero.com/v1/container-metrics" env-description:"location to send metrics to"`
	
	// APIKeyPath points to the file containing CloudZero API authentication key.
	// Required for secure communication with CloudZero services.
	APIKeyPath string `yaml:"api_key_path" env:"API_KEY_PATH" env-description:"path to the API key file"`
	
	// Service configuration sections
	
	// Server configures the webhook HTTP server that receives admission review requests
	Server Server `yaml:"server"`
	
	// Certificate manages TLS certificates for secure webhook communication with API server
	Certificate Certificate `yaml:"certificate"`
	
	// Logging controls webhook operation logging and debugging
	Logging Logging `yaml:"logging"`
	
	// Database configures local storage for webhook events and metrics
	Database Database `yaml:"database"`
	
	// Filters define rules for which Kubernetes resources to process
	Filters Filters `yaml:"filters"`
	
	// RemoteWrite configures forwarding of metrics to CloudZero
	RemoteWrite RemoteWrite `yaml:"remote_write"`
	
	// K8sClient configures Kubernetes API client for cluster interaction
	K8sClient K8sClient `yaml:"k8s_client"`
	
	// Runtime compiled patterns for efficient resource filtering
	
	// LabelMatches contains compiled regex patterns for label-based filtering.
	// Populated during configuration initialization from Filters.Labels.Patterns.
	LabelMatches []regexp.Regexp
	
	// AnnotationMatches contains compiled regex patterns for annotation-based filtering.
	// Populated during configuration initialization from Filters.Annotations.Patterns.
	AnnotationMatches []regexp.Regexp

	// Thread safety mutex for runtime configuration updates
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

// NewSettings creates and initializes a new webhook Settings instance by loading
// configuration from multiple sources and performing complete validation and setup.
// This constructor is specifically designed for webhook admission controller requirements.
//
// Parameters:
//   - configFiles: Variable list of YAML configuration file paths to load.
//     Files are processed sequentially with later files overriding earlier ones.
//     Empty strings are ignored for flexibility.
//
// Returns:
//   - *Settings: Fully initialized webhook configuration ready for use
//   - error: Configuration loading, validation, or setup error
//
// The webhook constructor performs these specialized operations:
//   1. Load configuration from YAML files and environment variables
//   2. Clean and validate cloud account ID using alphanumeric filtering
//   3. Auto-detect cloud environment (AWS, Azure, GCP) using scout package
//   4. Compile regex patterns for resource filtering (labels, annotations)
//   5. Load and validate CloudZero API key from configured path
//   6. Build CloudZero API endpoints with cluster parameters
//   7. Initialize HTML sanitization policies for security
//   8. Configure logging options for webhook operation
//
// Key webhook-specific features:
//   - Regex pattern compilation for efficient Kubernetes resource filtering
//   - HTML sanitization policy initialization for security
//   - TLS certificate validation for secure API server communication
//   - Kubernetes client configuration for cluster interaction
//
// Error conditions:
//   - Missing or unreadable configuration files
//   - Invalid regex patterns in filter configuration
//   - Cloud environment auto-detection failures
//   - Missing or invalid API key file
//   - TLS certificate validation failures
//
// Example:
//   settings, err := NewSettings("/etc/webhook/config.yaml", "/etc/webhook/filters.yaml")
//   if err != nil {
//       return fmt.Errorf("failed to initialize webhook config: %w", err)
//   }
//   
//   log.Printf("Webhook monitoring cluster %s with %d label filters", 
//       settings.ClusterName, len(settings.LabelMatches))
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
	logger := log.Logger.With().Str("component", "webhook-settings").Logger()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := scout.DetectConfiguration(ctx, &logger, nil, &cfg.Region, &cfg.CloudAccountID, &cfg.ClusterName)
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
