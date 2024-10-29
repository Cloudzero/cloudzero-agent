// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package config

import "time"

type Database struct {
	Enabled         bool          `yaml:"enabled" default:"false" env:"DATABASE_ENABLED" env-description:"when enabled will write to persistent storage, otherwise only in memory sqlite"`
	StoragePath     string        `yaml:"storage_path" default:"/opt/insights" env:"DATABASE_STORAGE_PATH" env-description:"location where to write database"`
	RetentionTime   time.Duration `yaml:"retention_time" default:"24h" env:"DATABASE_RETENTION" env-description:"how long local data should be retain before being deleted"`
	CleanupInterval time.Duration `yaml:"cleanup_interval" default:"3h" env:"DATABASE_CLEANUP_INTERVAL" env-description:"how often to check for expired data"`
}
