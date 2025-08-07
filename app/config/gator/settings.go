// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package config implements the configuration management system for the CloudZero agent.
//
// This package provides the main configuration structure (Settings) and validation
// logic for the agent's core functionality. It handles:
//
//   - Configuration file parsing and environment variable binding
//   - Cloud environment auto-detection (AWS, Azure, GCP)
//   - API key management and validation
//   - Service endpoint configuration and URL building
//   - Resource limits and performance tuning settings
//   - Database storage and purging configuration
//
// Configuration Sources:
//   The Settings structure supports multiple configuration sources with precedence:
//   1. Environment variables (highest priority)
//   2. YAML configuration files
//   3. Default values (lowest priority)
//
// Key Configuration Areas:
//   - Core Settings: Cluster identification, cloud account details, region
//   - Server Settings: HTTP server configuration, ports, profiling
//   - Database Settings: Storage paths, compression, retention policies
//   - CloudZero Settings: API endpoints, authentication, upload intervals
//   - Metrics Settings: Cost/observability metric filtering rules
//   - Logging Settings: Log levels, capture settings
//
// Usage:
//   settings, err := NewSettings("/path/to/config.yaml")
//   if err != nil {
//       log.Fatal(err)
//   }
//   defer settings.Validate()
//
// The configuration system automatically detects cloud environment details
// using the scout package, validates all settings, and prepares API endpoints
// for CloudZero integration.
package config

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ccoveille/go-safecast"
	"github.com/cloudzero/cloudzero-agent/app/domain/filter"
	"github.com/cloudzero/cloudzero-agent/app/utils/scout"
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/api/resource"
)

// Configuration defaults and constants for the CloudZero agent.
// These values provide sensible defaults that balance performance, reliability,
// and resource usage across different deployment environments.
const (
	// CloudZero API configuration defaults
	
	// DefaultCZHost is the production CloudZero API hostname for metric uploads
	DefaultCZHost = "api.cloudzero.com"
	
	// DefaultCZSendInterval controls how frequently metrics are uploaded to CloudZero.
	// 10 minutes provides a good balance between data freshness and API load.
	DefaultCZSendInterval = 10 * time.Minute
	
	// DefaultCZSendTimeout is the maximum time allowed for individual upload requests.
	// Short enough to prevent hanging connections, long enough for large payloads.
	DefaultCZSendTimeout = 10 * time.Second
	
	// DefaultCZRotateInterval controls API key refresh frequency.
	// Matches send interval to ensure fresh authentication for uploads.
	DefaultCZRotateInterval = 10 * time.Minute
	
	// Database and storage configuration defaults
	
	// DefaultDatabaseMaxRecords limits the number of metrics per file to optimize
	// memory usage and upload performance. 1.5M records ≈ 100-200MB files.
	DefaultDatabaseMaxRecords = 1_500_000
	
	// DefaultDatabaseCompressionLevel balances compression ratio vs CPU usage.
	// Level 8 provides good compression without excessive CPU overhead.
	DefaultDatabaseCompressionLevel = 8
	
	// DefaultDatabaseCostMaxInterval controls how frequently cost metrics are flushed
	// to storage. Shorter interval ensures cost data is captured quickly.
	DefaultDatabaseCostMaxInterval = 10 * time.Minute
	
	// DefaultDatabaseObservabilityMaxInterval controls observability metric flushing.
	// Longer interval acceptable as these metrics are less time-critical.
	DefaultDatabaseObservabilityMaxInterval = 30 * time.Minute
	
	// HTTP server configuration defaults
	
	// DefaultServerPort is the standard HTTP port for agent API endpoints
	DefaultServerPort = 8080
	
	// DefaultServerMode specifies HTTP (not HTTPS) for internal cluster communication
	DefaultServerMode = "http"

	// Shutdown coordination constants
	
	// ShutdownMarkerFilename is created when the collector completes graceful shutdown.
	// Used to coordinate shutdown sequencing between agent components.
	ShutdownMarkerFilename = "collector-shutdown-complete"
	
	// ShutdownMarkerFileMode sets restrictive permissions on the shutdown marker file
	ShutdownMarkerFileMode = 0o600
)

// Settings represents the complete configuration structure for the CloudZero agent.
// It aggregates all configuration sections and provides centralized management
// of agent behavior, API integration, storage, and metric processing.
//
// The Settings struct supports configuration from multiple sources:
//   - YAML configuration files (via yaml tags)
//   - Environment variables (via env tags)
//   - Programmatic defaults during validation
//
// Thread safety:
//   The Settings struct includes a mutex for thread-safe access to mutable
//   configuration values like API keys that may be refreshed at runtime.
//
// Configuration lifecycle:
//   1. NewSettings() creates and loads configuration from files/environment
//   2. Validate() performs validation and auto-detection
//   3. SetAPIKey() loads and caches authentication credentials
//   4. SetRemoteUploadAPI() prepares CloudZero endpoint URLs
//
// Usage:
//   settings, err := NewSettings("/path/to/config.yaml")
//   if err != nil {
//       return fmt.Errorf("configuration error: %w", err)
//   }
//   
//   // Settings is now ready for use by agent components
//   collector := NewMetricCollector(settings, ...)
type Settings struct {
	// Core identification settings for CloudZero integration
	
	// CloudAccountID identifies the cloud provider account (AWS Account ID, 
	// Azure Subscription ID, GCP Project ID). Auto-detected if not specified.
	CloudAccountID string `yaml:"cloud_account_id" env:"CLOUD_ACCOUNT_ID" env-description:"CSP account ID"`
	
	// Region specifies the cloud provider region where the cluster is deployed.
	// Used for cost attribution and resource organization. Auto-detected if not specified.
	Region string `yaml:"region" env:"CSP_REGION" env-description:"cloud service provider region"`
	
	// ClusterName uniquely identifies this Kubernetes cluster within the cloud account.
	// Must be unique across all clusters monitored by CloudZero. Auto-detected if not specified.
	ClusterName string `yaml:"cluster_name" env:"CLUSTER_NAME" env-description:"name of the cluster to monitor"`

	// Service configuration sections
	
	// Server configures the HTTP server that exposes agent APIs
	Server Server `yaml:"server"`
	
	// Logging controls log output, levels, and persistence settings
	Logging Logging `yaml:"logging"`
	
	// Database configures local storage for metrics before upload
	Database Database `yaml:"database"`
	
	// Cloudzero configures integration with CloudZero APIs and services
	Cloudzero Cloudzero `yaml:"cloudzero"`
	
	// Metrics defines filtering rules for cost vs observability metrics
	Metrics Metrics `yaml:"metrics"`

	// Thread safety mutex for runtime configuration updates
	mu sync.Mutex
}

type Metrics struct {
	Cost                []filter.FilterEntry `yaml:"cost"`
	Observability       []filter.FilterEntry `yaml:"observability"`
	CostLabels          []filter.FilterEntry `yaml:"cost_labels"`
	ObservabilityLabels []filter.FilterEntry `yaml:"observability_labels"`
}

type Logging struct {
	Level   string `yaml:"level" default:"info" env:"LOG_LEVEL" env-description:"logging level such as debug, info, error"`
	Capture bool   `yaml:"capture" default:"true" env:"LOG_CAPTURE" env-description:"whether to persist logs to disk or not"`
}

type Database struct {
	StoragePath              string        `yaml:"storage_path" default:"/cloudzero/data" env:"DATABASE_STORAGE_PATH" env-description:"location where to write database"`
	MaxRecords               int           `yaml:"max_records" default:"1000000" env:"MAX_RECORDS_PER_FILE" env-description:"maximum records per file"`
	CompressionLevel         int           `yaml:"compression_level" default:"8" env:"DATABASE_COMPRESS_LEVEL" env-description:"compression level for database files"`
	CostMaxInterval          time.Duration `yaml:"cost_max_interval" default:"10m" env:"COST_MAX_INTERVAL" env-description:"maximum interval to wait before flushing cost metrics"`
	ObservabilityMaxInterval time.Duration `yaml:"observability_max_interval" default:"10m" env:"OBSERVABILITY_MAX_INTERVAL" env-description:"maximum interval to wait before flushing observability metrics"`

	PurgeRules       PurgeRules `yaml:"purge_rules"`
	AvailableStorage string     `yaml:"available_storage" default:"" env:"DATABASE_AVAILABLE_STORAGE" env-description:"total size alloted to the gator to store metric files"`
}

type PurgeRules struct {
	MetricsOlderThan time.Duration `yaml:"metrics_older_than" env-default:"2160h" env:"PURGE_METRICS_OLDER_THAN" env-description:"The amount of time to keep metric information locally. Any file older than the duration specified here can be deleted to free up space on the disk"`
	Lazy             bool          `yaml:"lazy" default:"true" env:"PURGE_LAZY" env-description:"Whether to purge the files in lazy mode. In this mode, if the metrics are older than 'metrics_older_than' but there is no detected disk pressure, the older 'stale' metrics will be retained"`
	Percent          int           `yaml:"percent" default:"20" env:"PURGE_PERCENT" env-description:"The percentage of files to remove from disk when critical disk pressure is detected. This is critical for ensuring the disk health is preserved"`
}

type Server struct {
	Mode               string `yaml:"mode" default:"http" env:"SERVER_MODE" env-description:"server mode such as http, https"`
	Port               uint   `yaml:"port" default:"8080" env:"SERVER_PORT" env-description:"server port"`
	Profiling          bool   `yaml:"profiling" default:"false" env:"SERVER_PROFILING" env-description:"enable profiling"`
	ReconnectFrequency int    `yaml:"reconnect_frequency" default:"16" env:"SERVER_RECONNECT_FREQUENCY" env-description:"how frequently to close HTTP connections from clients, to distribute the load. 0=never, otherwise 1/N probability."`
}

type Cloudzero struct {
	APIKeyPath     string        `yaml:"api_key_path" env:"API_KEY_PATH" env-description:"path to the API key file"`
	RotateInterval time.Duration `yaml:"rotate_interval" default:"10m" env:"ROTATE_INTERVAL" env-description:"interval in hours to rotate API key"`
	SendInterval   time.Duration `yaml:"send_interval" default:"10m" env:"SEND_INTERVAL" env-description:"interval in seconds to send data"`
	SendTimeout    time.Duration `yaml:"send_timeout" default:"120s" env:"SEND_TIMEOUT" env-description:"timeout in seconds to send data"`
	HTTPMaxRetries int           `yaml:"http_max_retries" default:"10" env:"HTTP_MAX_RETRIES" env-description:"number of times the http client will retry on failures"`
	HTTPMaxWait    time.Duration `yaml:"http_max_wait" default:"30s" env:"HTTP_MAX_WAIT" env-description:"interval to wait between HTTP request retries"`
	Host           string        `yaml:"host" env:"HOST" default:"api.cloudzero.com" env-description:"host to send metrics to"`
	UseHTTP        bool          `yaml:"use_http" env:"USE_HTTP" default:"false" env-description:"use http for client requests instead of https"`
	apiKey         string        // Set after reading keypath

	_host string // cached value of `Host` since it is overridden in initialization
}

// NewSettings creates a new Settings instance by loading configuration from
// multiple sources in order of precedence: environment variables, configuration
// files, and defaults. It performs full initialization including validation,
// cloud environment detection, and API endpoint preparation.
//
// Parameters:
//   - configFiles: Variable list of YAML configuration file paths to load.
//     Files are processed in order, with later files potentially overriding
//     earlier ones. Empty strings are ignored.
//
// Returns:
//   - *Settings: Fully initialized and validated configuration ready for use
//   - error: Configuration, validation, or initialization error
//
// The constructor performs these operations in sequence:
//   1. Load configuration from each file using cleanenv
//   2. Validate all configuration values and apply defaults
//   3. Auto-detect cloud environment details (account ID, region, cluster name)
//   4. Load and validate API key from configured path
//   5. Build CloudZero API endpoints with proper parameters
//
// Error conditions:
//   - Missing or unreadable configuration files
//   - Invalid configuration values or missing required fields
//   - Cloud environment auto-detection failures
//   - Missing or invalid API key file
//   - Invalid CloudZero API endpoint configuration
//
// Example:
//   settings, err := NewSettings("/etc/agent/config.yaml", "/etc/agent/secrets.yaml")
//   if err != nil {
//       return fmt.Errorf("failed to load configuration: %w", err)
//   }
//   
//   log.Printf("Monitoring cluster %s in %s", settings.ClusterName, settings.Region)
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
			return nil, fmt.Errorf("no config %s", cfgFile)
		}

		err := cleanenv.ReadConfig(cfgFile, &cfg)
		if err != nil {
			return nil, fmt.Errorf("config read %s: %w", cfgFile, err)
		}
	}

	if err := cfg.Validate(); err != nil {
		return nil, errors.Wrap(err, "failed to validate settings")
	}

	if err := cfg.SetAPIKey(); err != nil {
		return nil, errors.Wrap(err, "failed to get API key")
	}

	if err := cfg.SetRemoteUploadAPI(); err != nil {
		return nil, errors.Wrap(err, "failed to set remote upload API")
	}

	return &cfg, nil
}

func (s *Settings) Validate() error {
	// Cleanup and validate settings
	s.CloudAccountID = strings.TrimSpace(s.CloudAccountID)
	s.Region = strings.TrimSpace(s.Region)

	// Auto-detect cloud account ID and region if needed
	logger := log.Logger.With().Str("component", "gator-settings").Logger()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := scout.DetectConfiguration(ctx, &logger, nil, &s.Region, &s.CloudAccountID, &s.ClusterName)
	if err != nil {
		return fmt.Errorf("failed to auto-detect cloud environment: %w", err)
	}

	s.ClusterName = strings.TrimSpace(s.ClusterName)
	if s.ClusterName == "" {
		return errors.New("cluster name is empty")
	}

	if err := s.Server.Validate(); err != nil {
		return errors.Wrap(err, "server validation")
	}

	if err := s.Database.Validate(); err != nil {
		return errors.Wrap(err, "database validation")
	}

	if err := s.Cloudzero.Validate(); err != nil {
		return errors.Wrap(err, "cloudzero validation")
	}

	return nil
}

func (d *Database) Validate() error {
	if d.MaxRecords <= 0 {
		d.MaxRecords = DefaultDatabaseMaxRecords
	}
	if d.CostMaxInterval <= 0 {
		d.CostMaxInterval = DefaultDatabaseCostMaxInterval
	}
	if d.ObservabilityMaxInterval <= 0 {
		d.ObservabilityMaxInterval = DefaultDatabaseObservabilityMaxInterval
	}
	if _, err := os.Stat(d.StoragePath); os.IsNotExist(err) {
		return errors.Wrap(err, "database storage path does not exist")
	}

	// validate the passed sizeLimit is valid if it is not empty
	if d.AvailableStorage != "" {
		if _, err := resource.ParseQuantity(d.AvailableStorage); err != nil {
			return fmt.Errorf("failed to parse the size_limit quantity: %w", err)
		}
	}

	return nil
}

func (s *Server) Validate() error {
	if s.Mode == "" {
		s.Mode = DefaultServerMode
	}
	if s.Port == 0 {
		s.Port = DefaultServerPort
	}
	return nil
}

func (c *Cloudzero) Validate() error {
	if c.Host == "" {
		c.Host = DefaultCZHost
	}
	if c.SendInterval <= 0 {
		c.SendInterval = DefaultCZSendInterval
	}
	if c.SendTimeout <= 0 {
		c.SendTimeout = DefaultCZSendTimeout
	}
	if c.RotateInterval <= 0 {
		c.RotateInterval = DefaultCZRotateInterval
	}
	if c.APIKeyPath == "" {
		return errors.New("API key path is empty")
	}
	if _, err := os.Stat(c.APIKeyPath); os.IsNotExist(err) {
		return errors.Wrap(err, "API key path does not exist")
	}
	return nil
}

func (s *Settings) GetAPIKey() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Cloudzero.apiKey
}

func (s *Settings) SetAPIKey() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	apiKeyPathLocation, err := absFilePath(s.Cloudzero.APIKeyPath)
	if err != nil {
		return errors.Wrap(err, "failed to get absolute path")
	}

	if _, err = os.Stat(apiKeyPathLocation); os.IsNotExist(err) {
		return fmt.Errorf("API key file %s not found", apiKeyPathLocation)
	}
	apiKey, err := os.ReadFile(s.Cloudzero.APIKeyPath)
	if err != nil {
		return errors.Wrap(err, "failed to read API key")
	}
	s.Cloudzero.apiKey = strings.TrimSpace(string(apiKey))

	if len(s.Cloudzero.apiKey) == 0 {
		return errors.New("API key is empty")
	}
	return nil
}

func (s *Settings) SetRemoteUploadAPI() error {
	if s.Cloudzero.Host == "" {
		return errors.New("host is empty")
	}
	s.Cloudzero._host = s.Cloudzero.Host // cache value to use later
	baseURL, err := url.Parse("https://" + s.Cloudzero.Host)
	if err != nil {
		return errors.Wrap(err, "failed to parse host")
	}
	baseURL.Path += "/v1/container-metrics/upload"
	params := url.Values{}
	params.Add("cluster_name", s.ClusterName)
	params.Add("cloud_account_id", s.CloudAccountID)
	params.Add("region", s.Region)
	baseURL.RawQuery = params.Encode()
	url := baseURL.String()

	if !isValidURL(url) {
		return errors.New("invalid URL")
	}
	s.Cloudzero.Host = url
	return nil
}

// GetRemoteAPIBase sanitizes the input host from the config, and returns a
// standard `url.URL` type to build the query from
func (s *Settings) GetRemoteAPIBase() (*url.URL, error) {
	if s.Cloudzero._host == "" {
		s.Cloudzero._host = s.Cloudzero.Host
	}

	// format the host to a standardized format
	val := s.Cloudzero._host
	if !strings.Contains(s.Cloudzero._host, "://") {
		val = "http://" + val
	}

	// parse as url
	u, err := url.Parse(val)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the url: %w", err)
	}

	// set extra info on the url
	if s.Cloudzero.UseHTTP {
		u.Scheme = "http"
	} else {
		u.Scheme = "https"
	}
	u.Path += "/v1/container-metrics"
	return u, nil
}

// GetAvailableSizeBytes parses the config file in real time and attempts
// to get the available size in bytes of the storage volume.
// If the value fails to be parsed, it will return 0.
func (s *Settings) GetAvailableSizeBytes() (uint64, error) {
	if s.Database.AvailableStorage == "" {
		return 0, nil
	}

	quantity, err := resource.ParseQuantity(s.Database.AvailableStorage)
	if err != nil {
		log.Ctx(context.Background()).Warn().Err(err).Str("sizeLimit", s.Database.AvailableStorage).Msg("failed to parse the size_limit, using 0 as the default value (all available space)")
		return 0, nil
	}

	// value will give size in bytes
	return safecast.MustConvert[uint64](quantity.Value()), nil
}

func isValidURL(uri string) bool {
	if _, err := url.ParseRequestURI(uri); err != nil {
		return false
	}
	return true
}

func absFilePath(location string) (string, error) {
	dir := filepath.Dir(filepath.Clean(location))
	if dir == "" || strings.HasPrefix(dir, ".") {
		wd, err := os.Getwd()
		if err != nil {
			return "", errors.Wrap(err, "working directory")
		}
		location = filepath.Clean(filepath.Join(wd, location))
	}
	return location, nil
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
