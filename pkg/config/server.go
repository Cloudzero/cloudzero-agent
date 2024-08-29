// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package config

type Server struct {
	Port         string `yaml:"port" default:"8080" env:"PORT" env-description:"port to listen on"`
	ReadTimeout  int    `yaml:"read_timeout" default:"15" env:"READ_TIMEOUT" env-description:"server read timeout in seconds"`
	WriteTimeout int    `yaml:"write_timeout" default:"15" env:"WRITE_TIMEOUT" env-description:"server write timeout in seconds"`
	IdleTimeout  int    `yaml:"idle_timeout" default:"60" env:"IDLE_TIMEOUT" env-description:"server idle timeout in seconds"`
}
