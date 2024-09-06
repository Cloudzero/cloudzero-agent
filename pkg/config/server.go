// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package config

import "time"

type Server struct {
	Port         string        `yaml:"port" default:"8080" env:"PORT" env-description:"port to listen on"`
	ReadTimeout  time.Duration `yaml:"read_timeout" default:"15s" env:"READ_TIMEOUT" env-description:"server read timeout in seconds"`
	WriteTimeout time.Duration `yaml:"write_timeout" default:"15s" env:"WRITE_TIMEOUT" env-description:"server write timeout in seconds"`
	IdleTimeout  time.Duration `yaml:"idle_timeout" default:"60s" env:"IDLE_TIMEOUT" env-description:"server idle timeout in seconds"`
}
