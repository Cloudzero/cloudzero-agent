// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package config

import (
	"fmt"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/pkg/errors"
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
			return nil, errors.Wrap(err, fmt.Sprintf("no config %s", cfgFile))
		}

		err := cleanenv.ReadConfig(cfgFile, &cfg)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("config read %s", cfgFile))
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
