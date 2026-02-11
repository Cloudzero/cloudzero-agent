// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package sqlite provides SQLite database driver implementation for CloudZero Agent data persistence.
// This package enables lightweight, embedded database storage for CloudZero Agent deployments where
// external database dependencies are not desired or available.
//
// SQLite integration serves as a Secondary Adapter in the hexagonal architecture, providing
// reliable local data persistence for resource metadata, configuration, and operational state
// without requiring external database infrastructure.
//
// Key capabilities:
//   - Embedded database: No external database server required
//   - ACID compliance: Full transaction support for data consistency
//   - Cross-platform: Consistent behavior across different operating systems
//   - Zero configuration: Self-contained database file with automatic initialization
//   - Performance optimization: Local disk access with minimal network overhead
//
// Use cases in CloudZero Agent:
//   - Resource metadata storage: Kubernetes resource cost allocation information
//   - Configuration persistence: Agent settings and policy data
//   - Operational state: Health checks, status tracking, and audit logs
//   - Development and testing: Simplified deployment without database dependencies
//   - Edge deployments: Lightweight installations with minimal infrastructure requirements
//
// Database configurations:
//   - File-based storage: Persistent database files for production deployments
//   - In-memory database: Temporary storage for testing and development
//   - Shared cache: Multiple connections sharing in-memory state
//
// The SQLite implementation provides the same repository interfaces as other storage backends,
// enabling seamless switching between SQLite and more robust database systems based on
// deployment requirements and operational scale.
//
// Performance considerations:
//   - Single-writer limitation: SQLite uses database-level locking for writes
//   - Read scalability: Multiple concurrent readers supported efficiently
//   - File-based durability: Automatic persistence with configurable sync modes
//   - Memory efficiency: Minimal memory footprint suitable for constrained environments
package sqlite

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/cloudzero/cloudzero-agent/app/storage/core"
)

const (
	// InMemoryDSN specifies a temporary in-memory SQLite database for testing and development.
	// This configuration creates a completely temporary database that exists only in memory
	// and is destroyed when the last connection is closed.
	//
	// Characteristics:
	//   - Zero persistence: Data is lost when connections close
	//   - Maximum performance: No disk I/O overhead
	//   - Isolation: Each connection gets a separate database instance
	//   - Testing suitability: Clean state for each test run
	//
	// Usage scenarios:
	//   - Unit testing: Clean database state for each test
	//   - Development: Rapid prototyping without file management
	//   - Temporary storage: Short-lived data processing operations
	InMemoryDSN = ":memory:"

	// MemorySharedCached creates an in-memory SQLite database with shared cache support.
	// This configuration enables multiple database connections to share the same in-memory
	// database instance, providing persistence within the application lifetime.
	//
	// Characteristics:
	//   - Shared state: Multiple connections access the same database instance
	//   - Application-scoped persistence: Data survives individual connection closures
	//   - Memory-only: No disk persistence, data lost on application restart
	//   - Concurrency support: Multiple goroutines can access shared database
	//
	// Usage scenarios:
	//   - Integration testing: Shared database state across multiple test components
	//   - Multi-connection applications: Repository instances sharing data
	//   - Caching layer: Temporary shared storage for computed results
	//   - Development: Simulating multi-instance behavior without file systems
	MemorySharedCached = "file:memory?mode=memory&cache=shared"
)

// NewSQLiteDriver creates a configured GORM database connection using SQLite as the underlying storage engine.
// This function initializes a SQLite database connection with CloudZero Agent-specific configuration
// optimized for resource metadata storage and operational data persistence.
//
// Configuration applied:
//   - Connection pooling: Optimized for CloudZero Agent usage patterns
//   - Transaction settings: ACID compliance with performance optimization
//   - Logging integration: Structured logging for database operations
//   - Error handling: Consistent error translation to domain errors
//   - Schema management: Automatic table creation and migration support
//
// DSN (Data Source Name) formats:
//   - File-based: "/path/to/database.db" for persistent storage
//   - In-memory: ":memory:" for temporary testing databases
//   - Shared cache: "file:memory?mode=memory&cache=shared" for multi-connection testing
//   - Custom options: "file:path?_pragma=value" for advanced SQLite configuration
//
// The returned database connection is ready for use with CloudZero Agent repositories
// and includes all necessary configuration for production deployment or testing scenarios.
//
// Error conditions:
//   - Invalid DSN format or file path permissions
//   - SQLite library initialization failures
//   - Database file corruption or access restrictions
//   - Insufficient disk space for database creation
//
// Returns a configured *gorm.DB instance ready for repository operations.
func NewSQLiteDriver(dsn string) (*gorm.DB, error) {
	db, err := core.NewDriver(sqlite.Open(dsn))
	if err != nil {
		return nil, err
	}
	return db, nil
}
