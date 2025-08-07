// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package core provides GORM database driver initialization and configuration
// utilities that establish standardized database behavior across the CloudZero agent.
//
// This file contains the core database driver initialization logic that ensures
// consistent behavior across different database backends (SQLite, PostgreSQL, etc.):
//
//   - Naming strategy: Singular table names for consistency
//   - Timestamp handling: UTC timestamps with millisecond precision
//   - Logging integration: Structured logging through zerolog adapter
//   - Error translation: Automatic translation of database errors to application types
//
// The driver configuration is designed to provide:
//   - Predictable timestamp behavior across different time zones
//   - Consistent table naming regardless of model struct names
//   - Unified error handling across different database backends
//   - Performance-optimized settings for agent workloads
//
// Database timing:
//   All timestamps use UTC with millisecond precision to ensure:
//   - Consistent ordering across distributed systems
//   - Platform-independent behavior
//   - Compatibility with CloudZero API timestamp formats
//   - Efficient storage and indexing
package core

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// NewDriver creates a standardized GORM database instance with CloudZero agent configurations.
//
// This function applies consistent configuration settings across all database backends
// to ensure predictable behavior regardless of the underlying database technology.
//
// Applied configurations:
//   - SingularTable: Uses singular table names ("user" not "users")
//   - NowFunc: UTC timestamps with millisecond precision for consistency
//   - Logger: Structured logging through zerolog adapter
//   - TranslateError: Automatic error translation to application types
//
// The configuration is designed to:
//   - Provide consistent behavior across SQLite, PostgreSQL, MySQL, etc.
//   - Ensure timezone-independent timestamp handling
//   - Enable structured logging for debugging and monitoring
//   - Simplify error handling through consistent error types
//
// Parameters:
//   - dialector: GORM dialector for specific database backend
//
// Returns:
//   - *gorm.DB: Configured database instance ready for use
//   - error: Database connection or configuration error
//
// Usage:
//   // For SQLite
//   db, err := NewDriver(sqlite.Open("app.db"))
//   
//   // For PostgreSQL
//   db, err := NewDriver(postgres.Open(dsn))
func NewDriver(dialector gorm.Dialector) (*gorm.DB, error) {
	return gorm.Open(dialector, &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
		NowFunc:        DatabaseNow, // For timestamps, use UTC, truncated to milliseconds
		Logger:         &ZeroLogAdapter{},
		TranslateError: true,
	})
}

// DatabaseNow returns a standardized timestamp for database operations.
//
// This function provides consistent timestamp generation across all database
// operations by:
//   - Using UTC timezone to avoid timezone-related issues
//   - Truncating to millisecond precision for consistent storage
//   - Ensuring compatibility with CloudZero API timestamp formats
//
// The millisecond truncation ensures that:
//   - Timestamps are consistent across different database backends
//   - Sorting and comparison operations work predictably
//   - Storage efficiency is optimized
//   - API compatibility is maintained
//
// Returns:
//   - time.Time: Current time in UTC, truncated to millisecond precision
//
// This function is used by GORM for all created_at and updated_at fields.
func DatabaseNow() time.Time {
	return time.Now().UTC().Truncate(time.Millisecond)
}

// DatabaseNowPointer returns a pointer to a standardized timestamp for database fields.
//
// This function is a convenience wrapper around DatabaseNow() that returns a pointer
// to the timestamp, which is required for optional timestamp fields in GORM models.
//
// Returns:
//   - *time.Time: Pointer to current UTC timestamp with millisecond precision
//
// Usage:
//   type Model struct {
//       ID        uint       `gorm:"primarykey"`
//       CreatedAt *time.Time // Can use DatabaseNowPointer() for default
//       UpdatedAt *time.Time
//   }
func DatabaseNowPointer() *time.Time {
	now := DatabaseNow()
	return &now
}
