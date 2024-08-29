// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package config

type Annotations struct {
	Enabled bool     `yaml:"enabled" default:"false" env:"ANNOTATIONS_ENABLED" env-description:"enable annotations"`
	Filters []string `yaml:"filters" env:"ANNOTATIONS_FILTERS" env-description:"list of annotations to filter"`
}
