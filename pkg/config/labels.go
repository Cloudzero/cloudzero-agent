// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package config

type Labels struct {
	Enabled bool     `yaml:"enabled" default:"false" env:"LABELS_ENABLED" env-description:"enable labels"`
	Filters []string `yaml:"filters" env:"LABELS_FILTERS" env-description:"list of labels to filter"`
}
