// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import "time"

type Server struct {
	Mode      string `yaml:"mode" default:"http" env:"SERVER_MODE" env-description:"server mode such as http, https"`
	Namespace string `yaml:"namespace" env:"NAMESPACE" env-description:"namespace of the webhook"`
	Domain    string `yaml:"domain" default:"" env:"SERVER_DOMAIN" env-description:"server domain"`
	Port      uint   `yaml:"port" default:"8080" env:"SERVER_PORT" env-description:"server port"`
	Profiling bool   `yaml:"profiling" default:"false" env:"SERVER_PROFILING" env-description:"enable profiling"`

	ReadTimeout        time.Duration `yaml:"read_timeout" default:"15s" env:"READ_TIMEOUT" env-description:"server read timeout in seconds"`
	WriteTimeout       time.Duration `yaml:"write_timeout" default:"15s" env:"WRITE_TIMEOUT" env-description:"server write timeout in seconds"`
	IdleTimeout        time.Duration `yaml:"idle_timeout" default:"60s" env:"IDLE_TIMEOUT" env-description:"server idle timeout in seconds"`
	ReconnectFrequency int           `yaml:"reconnect_frequency" default:"16" env:"SERVER_RECONNECT_FREQUENCY" env-description:"how frequently to close HTTP connections from clients, to distribute the load. 0=never, otherwise 1/N probability."`
}
