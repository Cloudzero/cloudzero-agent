// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package config

type Database struct {
	Enabled     bool   `yaml:"enabled" default:"false" env:"DATABASE_ENABLED" env-description:"when enabled will write to persistent storage, otherwise only in memory sqlite"`
	StoragePath string `yaml:"storage_path" default:"/opt/insights" env:"DATABASE_STORAGE_PATH" env-description:"location where to write database"`
}
