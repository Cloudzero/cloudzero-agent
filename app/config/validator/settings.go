// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package config contains configuration settings.
package config

import (
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type Settings struct {
	ExecutionContext Context
	Logging          Logging     `yaml:"logging"`
	Deployment       Deployment  `yaml:"deployment"`
	Versions         Versions    `yaml:"versions"`
	Cloudzero        Cloudzero   `yaml:"cloudzero"`
	Prometheus       Prometheus  `yaml:"prometheus"`
	Diagnostics      Diagnostics `yaml:"diagnostics"`
}

func NewSettings(configFiles ...string) (*Settings, error) {
	var cfg Settings
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
