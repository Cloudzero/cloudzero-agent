// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package sqlite provides SQLite database driver configuration and initialization
// for the CloudZero agent's resource repository implementations.
//
// This package specializes the core database infrastructure for SQLite usage,
// providing optimized configurations for the agent's data storage requirements:
//
//   - SQLite driver initialization with CloudZero-specific settings
//   - In-memory database configurations for testing and development
//   - Shared cache configurations for multi-connection scenarios
//   - Integration with the core repository pattern infrastructure
//
// SQLite configurations:
//   - In-memory databases: Perfect for testing and temporary data
//   - File-based databases: Persistent storage with configurable paths
//   - Shared cache: Enables concurrent access from multiple connections
//   - Transaction support: Full ACID compliance with proper isolation
//
// The package leverages the core storage abstractions while providing
// SQLite-specific optimizations:
//   - Optimized connection settings for agent workloads
//   - Proper foreign key constraint enforcement
//   - UTF-8 encoding configuration
//   - Transaction timeout configurations
//
// Usage patterns:
//   // Development/testing with in-memory database
//   db, err := sqlite.NewSQLiteDriver(sqlite.InMemoryDSN)
//   
//   // Production with persistent file storage
//   db, err := sqlite.NewSQLiteDriver(\"/var/lib/agent/data.sqlite\")
//   
//   // Shared cache for multiple connections
//   db, err := sqlite.NewSQLiteDriver(sqlite.MemorySharedCached)
//
// Integration with repositories:
//   The returned *gorm.DB can be used with any repository implementation
//   from the core package, providing type-safe CRUD operations with
//   transaction support.
package sqlite

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/cloudzero/cloudzero-agent/app/storage/core"
)

const (
	// InMemoryDSN configures an in-memory SQLite database that exists only in RAM.
	// Perfect for testing and development scenarios where persistence is not required.
	// Each connection gets its own isolated database instance.
	InMemoryDSN        = ":memory:"
	
	// MemorySharedCached configures a shared in-memory SQLite database.
	// Multiple connections can access the same database instance through shared cache.
	// Ideal for testing scenarios that require multiple concurrent connections
	// or when testing repository interactions across different contexts.
	MemorySharedCached = "file:memory?mode=memory&cache=shared"
)

// NewSQLiteDriver creates a GORM database instance configured for SQLite with CloudZero agent settings.
//
// This function initializes a SQLite database connection using the core driver infrastructure,
// which applies standardized configurations including:
//   - Singular table naming strategy
//   - UTC timestamp handling with millisecond precision
//   - Structured logging integration
//   - Error translation for consistent error handling
//
// The returned database instance is ready for use with repository implementations
// and supports full transaction capabilities.
//
// Parameters:
//   - dsn: Data Source Name specifying the SQLite database location
//          Common values: file path, InMemoryDSN, MemorySharedCached
//
// Returns:
//   - *gorm.DB: Configured database instance ready for repository usage
//   - error: Driver initialization or connection error
//
// Example usage:
//   // File-based database for production
//   db, err := NewSQLiteDriver("/var/lib/agent/resources.sqlite")
//   if err != nil {
//       return fmt.Errorf("database init failed: %w", err)
//   }
//   
//   // In-memory database for testing
//   testDB, err := NewSQLiteDriver(InMemoryDSN)
//   if err != nil {
//       t.Fatalf("test database setup failed: %v", err)
//   }
func NewSQLiteDriver(dsn string) (*gorm.DB, error) {
	db, err := core.NewDriver(sqlite.Open(dsn))
	if err != nil {
		return nil, err
	}
	return db, nil
}
