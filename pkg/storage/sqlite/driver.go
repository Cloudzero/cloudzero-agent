// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
// Package sqlite provides implementations for resource repository interfaces
// using SQLite as the underlying database. This package includes implementations
// for repositories that can be extended to fit specific use cases. It supports
// transaction management and context-based database operations.
package sqlite

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/storage/core"
)

const (
	InMemoryDSN        = ":memory:"
	MemorySharedCached = "file::memory:?cache=shared"
)

// NewSQLiteDriver creates a gorm SQLite driver configured with our settings.
func NewSQLiteDriver(dsn string) (*gorm.DB, error) {
	db, err := core.NewDriver(sqlite.Open(dsn))
	if err != nil {
		return nil, err
	}
	return db, nil
}
