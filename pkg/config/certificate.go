// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package config

type Certificate struct {
	Key  string `yaml:"key" env:"TLS_KEY" env-description:"path to the TLS key"`
	Cert string `yaml:"cert" env:"TLS_CERT" env-description:"path to the TLS certificate"`
}
