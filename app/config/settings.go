// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/pkg/errors"
)

const (
	DefaultCZHost                   = "api.cloudzero.com"
	DefaultCZSendInterval           = 10 * time.Minute
	DefaultCZSendTimeout            = 10 * time.Second
	DefaultCZRotateInterval         = 10 * time.Minute
	DefaultDatabaseMaxRecords       = 1_000_000
	DefaultDatabaseCompressionLevel = 8
	DefaultServerPort               = 8080
	DefaultServerMode               = "http"
)

type Settings struct {
	// Core Settings
	CloudAccountID string `yaml:"cloud_account_id" env:"CLOUD_ACCOUNT_ID" env-description:"CSP account ID"`
	Region         string `yaml:"region" env:"CSP_REGION" env-description:"cloud service provider region"`
	ClusterName    string `yaml:"cluster_name" env:"CLUSTER_NAME" env-description:"name of the cluster to monitor"`

	Server    Server    `yaml:"server"`
	Logging   Logging   `yaml:"logging"`
	Database  Database  `yaml:"database"`
	Cloudzero Cloudzero `yaml:"cloudzero"`

	mu sync.Mutex
}

type Logging struct {
	Level string `yaml:"level" default:"info" env:"LOG_LEVEL" env-description:"logging level such as debug, info, error"`
}

type Database struct {
	StoragePath          string `yaml:"storage_path" default:"/cloudzero/data" env:"DATABASE_STORAGE_PATH" env-description:"location where to write database"`
	StorageUploadSubpath string `yaml:"storage_upload_subpath" default:"uploaded" env:"DATABASE_STORAGE_UPLOAD_SUBPATH" env-description:"subpath inside of 'storage_path' on where to store uploaded files"`
	MaxRecords           int    `yaml:"max_records" default:"1000000" env:"MAX_RECORDS_PER_FILE" env-description:"maximum records per file"`
	CompressionLevel     int    `yaml:"compression_level" default:"8" env:"DATABASE_COMPRESS_LEVEL" env-description:"compression level for database files"`
}

type Server struct {
	Mode string `yaml:"mode" default:"http" env:"SERVER_MODE" env-description:"server mode such as http, https"`
	Port uint   `yaml:"port" default:"8080" env:"SERVER_PORT" env-description:"server port"`
}

type Cloudzero struct {
	APIKeyPath     string        `yaml:"api_key_path" env:"API_KEY_PATH" env-description:"path to the API key file"`
	RotateInterval time.Duration `yaml:"rotate_interval" default:"10m" env:"ROTATE_INTERVAL" env-description:"interval in hours to rotate API key"`
	SendInterval   time.Duration `yaml:"send_interval" default:"10m" env:"SEND_INTERVAL" env-description:"interval in seconds to send data"`
	SendTimeout    time.Duration `yaml:"send_timeout" default:"10s" env:"SEND_TIMEOUT" env-description:"timeout in seconds to send data"`
	Host           string        `yaml:"host" env:"HOST" default:"api.cloudzero.com" env-description:"host to send metrics to"`
	apiKey         string        // Set after reading keypath

	_host string // cached value of `Host` since it is overriden in initalization
}

func NewSettings(configFiles ...string) (*Settings, error) {
	var cfg Settings
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
	if s.CloudAccountID == "" {
		return errors.New("cloud account ID is empty")
	}
	s.Region = strings.TrimSpace(s.Region)
	if s.Region == "" {
		return errors.New("region is empty")
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
	if _, err := os.Stat(d.StoragePath); os.IsNotExist(err) {
		return errors.Wrap(err, "database storage path does not exist")
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

// Sanitizes the input host from the config, and returns a standard
// `url.URL` type to build the query from
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

	// remove a port off the host if any
	host := u.Host
	if strings.Contains(host, ":") {
		var err error
		host, _, err = net.SplitHostPort(host)
		if err != nil {
			return nil, fmt.Errorf("failed to remove the port from the host: %w", err)
		}
	}

	// define the parameters
	params := url.Values{}
	params.Add("cluster_name", s.ClusterName)
	params.Add("cloud_account_id", s.CloudAccountID)
	params.Add("region", s.Region)

	// set extra info on the url
	u.Scheme = "https"
	u.Host = host
	u.Path += "/v1/container-metrics"
	u.RawQuery = params.Encode()
	return u, nil
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
