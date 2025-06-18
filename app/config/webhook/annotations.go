// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config

type Annotations struct {
	Enabled   bool      `yaml:"enabled" default:"false" env:"ANNOTATIONS_ENABLED" env-description:"enable annotations"`
	Resources Resources `yaml:"resources"`
	Patterns  []string  `yaml:"patterns" env:"ANNOTATIONS_FILTERS" env-description:"list of annotations regular expressions to filter"`
}
