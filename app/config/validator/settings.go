// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package config contains configuration settings.
package config

import (
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type Settings struct {
	ExecutionContext Context
	Logging          Logging     `yaml:"logging"`
	Deployment       Deployment  `yaml:"deployment"`
	Versions         Versions    `yaml:"versions"`
	Cloudzero        Cloudzero   `yaml:"cloudzero"`
	Prometheus       Prometheus  `yaml:"prometheus"`
	Diagnostics      Diagnostics `yaml:"diagnostics"`
	Services         Services    `yaml:"services"`
}

type Services struct {
	Namespace        string `yaml:"namespace" validate:"required"`
	InsightsService  string `yaml:"insights_service" validate:"required"`
	CollectorService string `yaml:"collector_service" validate:"required"`
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
			return nil, fmt.Errorf("no config file %s: %w", cfgFile, err)
		}

		err := cleanenv.ReadConfig(cfgFile, &cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to read config from %s: %w", cfgFile, err)
		}
	}
	return &cfg, nil
}

func (s *Settings) Validate() error {
	if err := s.Logging.Validate(); err != nil {
		return err
	}

	if err := s.Deployment.Validate(); err != nil {
		return err
	}

	if err := s.Versions.Validate(); err != nil {
		return err
	}

	if err := s.Cloudzero.Validate(); err != nil {
		return err
	}

	if err := s.Prometheus.Validate(); err != nil {
		return err
	}

	if err := s.Diagnostics.Validate(); err != nil {
		return err
	}

	return nil
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
