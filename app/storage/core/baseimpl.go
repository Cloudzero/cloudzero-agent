// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package core provides foundational database repository infrastructure for CloudZero Agent storage operations.
// This package implements the Secondary Adapter layer in the hexagonal architecture, providing consistent
// database access patterns and transaction management across all CloudZero Agent data persistence operations.
//
// The core storage infrastructure enables:
//   - Consistent transaction management: Context-based transaction propagation and nesting
//   - Repository pattern implementation: Base implementations for CRUD operations and common patterns
//   - Database abstraction: GORM-based ORM abstraction with error translation and context handling
//   - Resource metadata storage: Foundation for storing Kubernetes resource cost allocation data
//   - Operational data persistence: Storage for agent configuration, status, and audit information
//
// Architecture:
//   - RawBaseRepoImpl: Low-level database access with transaction context management
//   - BaseRepoImpl: Higher-level repository base with model-specific operations
//   - Context management: Thread-safe transaction context propagation for nested operations
//   - Error translation: Consistent error handling and mapping from database errors to domain errors
//
// The storage core enables CloudZero Agent components to persist and retrieve data using consistent
// patterns while maintaining transactional integrity across complex operations involving multiple
// tables and entities. This is essential for accurate cost allocation data and operational reliability.
//
// Key design principles:
//   - Transaction safety: All operations support transactional contexts with proper rollback
//   - Context propagation: Database connections and transactions flow through Go contexts
//   - Type safety: Generic repository patterns with compile-time type checking
//   - Performance optimization: Connection pooling, prepared statements, and efficient queries
//
// Integration points:
//   - Resource repositories: Store Kubernetes resource metadata for cost allocation
//   - Configuration storage: Persist agent settings and dynamic configuration updates
//   - Audit logging: Store operational events and cost allocation decisions
//   - Status tracking: Maintain agent health and operational state information
package core

import (
	"context"

	"gorm.io/gorm"
)

// RawBaseRepoImpl provides low-level database access infrastructure for CloudZero Agent repositories.
// This struct implements the foundational database operations required by all repository implementations,
// including transaction management, context propagation, and connection pooling without assuming
// specific table models or entity structures.
//
// The "Raw" designation indicates that this implementation provides direct database access without
// model-specific assumptions, making it suitable for repositories that need custom query logic
// or work with multiple tables/models within a single repository interface.
//
// Key capabilities:
//   - Transaction context management: Automatic detection and propagation of transaction contexts
//   - Connection pooling: Efficient reuse of database connections across operations
//   - Context cancellation: Proper handling of request cancellation and timeouts
//   - Error translation: Consistent error mapping from GORM to domain-specific errors
//
// Usage pattern:
//
//	Raw base repositories are used when repositories need custom query logic that doesn't fit
//	the standard single-model pattern, such as repositories that join multiple tables or
//	perform complex analytical queries for cost allocation reporting.
type RawBaseRepoImpl struct {
	// db provides the root GORM database connection for repository operations.
	// This connection is used as the default when no transaction context is present,
	// and serves as the parent for transaction creation and connection pooling.
	//
	// The database connection includes:
	//   - Connection pool configuration for performance optimization
	//   - Prepared statement caching for query efficiency
	//   - Schema migration support for operational flexibility
	//   - Logging and metrics collection for operational monitoring
	db *gorm.DB
}

// NewRawBaseRepoImpl creates a new RawBaseRepoImpl for use in a concrete instance of a repository.
func NewRawBaseRepoImpl(db *gorm.DB) RawBaseRepoImpl {
	return RawBaseRepoImpl{
		db: db,
	}
}

// DB provides context-aware database access with automatic transaction detection and propagation.
// This method implements the core pattern used throughout CloudZero Agent storage operations
// to ensure consistent transaction behavior and proper context handling across all database operations.
//
// Transaction detection logic:
//  1. Check if the context contains an active transaction (via FromContext)
//  2. If transaction found, return the transaction-scoped database connection
//  3. If no transaction, return the default connection with context propagation
//  4. Always augment the connection with the current context for cancellation support
//
// Context propagation ensures:
//   - Request cancellation is properly handled during long-running queries
//   - Database timeouts are respected according to context deadlines
//   - Tracing and logging metadata flows through database operations
//   - Consistent behavior across transactional and non-transactional operations
//
// Usage patterns:
//   - All repository methods should use r.DB(ctx) instead of direct database access
//   - Enables seamless transaction support without changing repository method signatures
//   - Provides consistent error handling and context cancellation behavior
//   - Supports nested transactions through context propagation
//
// This method is the foundation that enables CloudZero Agent repositories to participate
// in complex transactional operations while maintaining clean, context-aware interfaces.
func (b *RawBaseRepoImpl) DB(ctx context.Context) *gorm.DB {
	if tx, found := FromContext(ctx); found {
		return tx.WithContext(ctx)
	}

	return b.db.WithContext(ctx)
}

// Tx executes a function within a database transaction, providing atomic operations for CloudZero Agent data consistency.
// This method enables complex multi-table operations to be performed atomically, ensuring data integrity
// for critical cost allocation operations and resource metadata management.
//
// Transaction lifecycle:
//  1. Creates a new database transaction using GORM's transaction support
//  2. Embeds the transaction connection into a new context (ctxTx)
//  3. Executes the provided block function with the transaction context
//  4. Automatically commits on successful block completion (no error returned)
//  5. Automatically rolls back if the block returns any error
//  6. Supports nested transactions through context layering
//
// Atomic operation examples:
//   - Resource metadata updates: Ensure webhook processing and storage are consistent
//   - Configuration changes: Update multiple configuration tables atomically
//   - Audit logging: Ensure operation and audit records are written together
//   - Batch operations: Process multiple resource updates with rollback capability
//
// Context propagation:
//
//	The ctxTx context passed to the block contains the transaction connection,
//	enabling all repository operations within the block to automatically participate
//	in the transaction without explicit transaction parameter passing.
//
// Nested transaction support:
//
//	If the current context already contains a transaction, GORM handles nested
//	transactions appropriately using savepoints, enabling complex operation composition.
//
// Error handling:
//   - Any error returned by the block function triggers automatic rollback
//   - Database-level errors during commit/rollback are propagated to the caller
//   - Context cancellation during the transaction triggers rollback
//   - Panic recovery within GORM ensures transaction cleanup
//
// This method is essential for maintaining data consistency in CloudZero Agent operations
// where resource metadata, configuration, and audit information must be kept synchronized.
func (b *RawBaseRepoImpl) Tx(ctx context.Context, block func(ctxTx context.Context) error) error {
	db := b.DB(ctx)
	err := db.Transaction(func(tx *gorm.DB) error {
		ctxTx := NewContext(ctx, tx)
		return block(ctxTx)
	})
	return err
}

// BaseRepoImpl adds core behaviors applicable to any database repository implementation.
// Repositories should be defined as follows:
//
//	   type MyAwesomeRepoImpl struct {
//		     core.BaseRepoImpl
//	   }
//
// And constructed as follows:
//
//	   func NewMyAwesomeRepoImpl(db *gorm.DB) *MyAwesomeRepoImpl {
//		     return &MyAwesomeRepoImpl{BaseRepoImpl:
//		       core.NewBaseRepoImpl(db, &MyAwesomeModel)}
//	   }
//
// Any database operations should be invoked using the DB() function:
//
//	r.DB(ctx).Where(...)
//
// Which ensures the proper *gorm.DB instance is used (this enables transaction support).
type BaseRepoImpl struct {
	RawBaseRepoImpl
	model interface{}
}

// NewBaseRepoImpl creates a new BaseRepoImpl for use in a concrete instance of a repository,
// as shown above. The `model` parameter is used to specify the struct that this repository is
// associated with. It is used, for example, in the Count() function.
func NewBaseRepoImpl(db *gorm.DB, model interface{}) BaseRepoImpl {
	return BaseRepoImpl{
		RawBaseRepoImpl: NewRawBaseRepoImpl(db),
		model:           model,
	}
}

// Count returns the number of rows in the table.
func (b *BaseRepoImpl) Count(ctx context.Context) (int, error) {
	var count int64
	err := b.DB(ctx).Model(b.model).Count(&count).Error
	return int(count), TranslateError(err)
}

// DeleteAll deletes all rows in the table (useful in testing).
func (b *BaseRepoImpl) DeleteAll(ctx context.Context) error {
	return TranslateError(b.DB(ctx).Where("1 = 1").Delete(b.model).Error)
}

// key is an unexported type for keys defined in this package.
// This prevents collisions with keys defined in other packages.
type key int

// dbKey is the key for *gorm.DB values in Contexts. It is
// unexported; clients use core.NewContext and core.FromContext
// instead of using this key directly.
var dbKey key

// NewContext returns a new Context that carries value db.
func NewContext(ctx context.Context, db *gorm.DB) context.Context {
	return context.WithValue(ctx, dbKey, db)
}

// FromContext returns the gorm.DB value stored in ctx, if any.
func FromContext(ctx context.Context) (*gorm.DB, bool) {
	db, ok := ctx.Value(dbKey).(*gorm.DB)
	return db, ok
}
