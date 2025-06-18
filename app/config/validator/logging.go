// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

type Logging struct {
	Level    string `yaml:"level" default:"info" env:"LOG_LEVEL" env-description:"logging level such as debug, info, error"`
	Location string `yaml:"location" default:"/prometheus/cloudzero-agent-validator.log" env:"LOG_LOCATION" env-description:"location where to write logs"`
}

func (s *Logging) Validate() error {
	if s.Level == "" {
		s.Level = "info"
	}

	if s.Location == "" {
		s.Location = "/prometheus/cloudzero-agent-validator.log"
	}

	location, err := absFilePath(s.Location)
	if err != nil {
		return err
	}
	s.Location = location
	dir := filepath.Dir(s.Location)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return err
	}

	return nil
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
