// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package core provides foundational database repository abstractions and implementations
// that enable consistent, transaction-aware data access patterns across the CloudZero agent.
//
// This package implements the repository pattern with GORM as the underlying ORM, providing:
//
//   - Transaction management: Seamless transaction support with context-based transaction passing
//   - Base implementations: Reusable repository functionality for CRUD operations
//   - Context-aware operations: All database operations respect transaction context
//   - Error translation: Consistent error handling across different database operations
//
// Architecture:
//   - RawBaseRepoImpl: Core database abstraction without model assumptions
//   - BaseRepoImpl: Standard repository implementation with model-specific operations
//   - Context-based transactions: Transaction state passed through context.Context
//   - GORM integration: Leverages GORM for ORM functionality and query building
//
// Transaction patterns:
//   The package provides transparent transaction support where operations automatically
//   participate in ongoing transactions when available:
//
//   1. Outside transaction: Uses default database connection
//   2. Inside transaction: Automatically uses transaction context
//   3. Nested transactions: Supported through context nesting
//
// Usage patterns:
//   type UserRepo struct { core.BaseRepoImpl }
//   
//   func NewUserRepo(db *gorm.DB) *UserRepo {
//       return &UserRepo{BaseRepoImpl: core.NewBaseRepoImpl(db, &User{})}
//   }
//   
//   func (r *UserRepo) FindByEmail(ctx context.Context, email string) (*User, error) {
//       var user User
//       err := r.DB(ctx).Where("email = ?", email).First(&user).Error
//       return &user, core.TranslateError(err)
//   }
//
// This design enables:
//   - Consistent transaction behavior across all repositories
//   - Easy testing with transaction rollbacks
//   - Clean separation of database concerns from business logic
//   - Reusable patterns for common database operations
package core

import (
	"context"

	"gorm.io/gorm"
)

// RawBaseRepoImpl provides the foundational database abstraction layer for repository
// implementations without making assumptions about specific data models.
//
// This implementation provides:
//   - Context-aware database connection management
//   - Transparent transaction support through context passing
//   - GORM integration with proper context handling
//   - Base functionality for more specific repository implementations
//
// RawBaseRepoImpl is designed to be embedded in concrete repository implementations
// that need database access but don't follow standard table-based patterns.
// For standard table-based repositories, use BaseRepoImpl instead.
//
// Usage:
//   type CustomRepo struct {
//       core.RawBaseRepoImpl
//   }
//   
//   func (r *CustomRepo) ComplexQuery(ctx context.Context) error {
//       return r.DB(ctx).Raw("SELECT ...").Error
//   }
type RawBaseRepoImpl struct {
	db *gorm.DB
}

// NewRawBaseRepoImpl creates a new RawBaseRepoImpl for use in concrete repository implementations.
//
// This constructor initializes the repository with a GORM database instance that will be used
// for all database operations. The database connection supports both regular operations and
// transaction-aware operations through context.
//
// Parameters:
//   - db: GORM database instance for database operations
//
// Returns:
//   - RawBaseRepoImpl: Configured repository base with database connection
//
// Example:
//   db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
//   if err != nil {
//       panic(err)
//   }
//   baseRepo := core.NewRawBaseRepoImpl(db)
func NewRawBaseRepoImpl(db *gorm.DB) RawBaseRepoImpl {
	return RawBaseRepoImpl{
		db: db,
	}
}

// DB returns a context-aware GORM database instance for use in database operations.
//
// This method provides transparent transaction support by automatically detecting
// whether the current context contains an ongoing transaction:
//
//   1. If context contains transaction: Returns transaction-aware database instance
//   2. If no transaction in context: Returns default database instance
//   3. Always applies context to the database instance for proper cancellation/timeout handling
//
// This design enables seamless transaction support without requiring repositories
// to explicitly handle transaction vs non-transaction scenarios.
//
// Parameters:
//   - ctx: Context that may contain transaction information
//
// Returns:
//   - *gorm.DB: Context-aware database instance (transaction or default)
//
// Usage:
//   // Works both inside and outside transactions
//   err := r.DB(ctx).Where("id = ?", id).First(&model).Error
func (b *RawBaseRepoImpl) DB(ctx context.Context) *gorm.DB {
	if tx, found := FromContext(ctx); found {
		return tx.WithContext(ctx)
	}

	return b.db.WithContext(ctx)
}

// Tx executes the provided block function within a database transaction.
//
// This method creates a new transaction context and passes it to the block function.
// All repository operations performed within the block that use the transaction context
// will participate in the same transaction.
//
// Transaction behavior:
//   - Success: If block returns nil, transaction is committed
//   - Error: If block returns error, transaction is rolled back
//   - Nesting: Supports nested transactions through context nesting
//   - Isolation: Each transaction operates with appropriate isolation level
//
// Parameters:
//   - ctx: Base context for the transaction
//   - block: Function to execute within transaction scope
//
// Returns:
//   - error: Transaction error or error from block function
//
// Example:
//   err := repo.Tx(ctx, func(txCtx context.Context) error {
//       // All operations with txCtx are transactional
//       if err := repo.Create(txCtx, &model1); err != nil {
//           return err // Triggers rollback
//       }
//       if err := repo.Update(txCtx, &model2); err != nil {
//           return err // Triggers rollback
//       }
//       return nil // Triggers commit
//   })
func (b *RawBaseRepoImpl) Tx(ctx context.Context, block func(ctxTx context.Context) error) error {
	db := b.DB(ctx)
	err := db.Transaction(func(tx *gorm.DB) error {
		ctxTx := NewContext(ctx, tx)
		return block(ctxTx)
	})
	return err
}

// BaseRepoImpl provides a complete repository base implementation for standard table-based
// data models, extending RawBaseRepoImpl with model-specific functionality.
//
// This implementation adds:
//   - Model-specific operations (Count, DeleteAll)
//   - Type-safe operations for specific data models
//   - Common repository patterns for CRUD operations
//   - Integration with the types.StorageCommon interface
//
// BaseRepoImpl is designed to be embedded in concrete repository implementations
// that follow standard table-based patterns with a single primary model type.
//
// Standard implementation pattern:
//   type UserRepoImpl struct {
//       core.BaseRepoImpl
//   }
//   
//   func NewUserRepoImpl(db *gorm.DB) *UserRepoImpl {
//       return &UserRepoImpl{
//           BaseRepoImpl: core.NewBaseRepoImpl(db, &User{}),
//       }
//   }
//   
//   func (r *UserRepoImpl) FindByEmail(ctx context.Context, email string) (*User, error) {
//       var user User
//       err := r.DB(ctx).Where("email = ?", email).First(&user).Error
//       return &user, core.TranslateError(err)
//   }
//
// The DB() method should be used for all database operations to ensure proper
// transaction support and context handling:
//   r.DB(ctx).Where(...).First(&model)
//
// This pattern ensures:
//   - Automatic transaction participation when context contains transaction
//   - Proper context cancellation and timeout handling
//   - Consistent error handling across all operations
type BaseRepoImpl struct {
	RawBaseRepoImpl
	model interface{}
}

// NewBaseRepoImpl creates a new BaseRepoImpl for standard table-based repository implementations.
//
// This constructor initializes a repository base with both database connection and model
// information, enabling model-specific operations like counting and bulk deletion.
//
// Parameters:
//   - db: GORM database instance for database operations
//   - model: Example instance of the model struct this repository manages
//
// Returns:
//   - BaseRepoImpl: Configured repository base with database and model information
//
// The model parameter is used to:
//   - Determine table name and schema information
//   - Enable model-specific operations (Count, DeleteAll)
//   - Provide type information to GORM for query building
//
// Example:
//   type User struct {
//       ID    uint   `gorm:"primaryKey"`
//       Name  string
//       Email string `gorm:"uniqueIndex"`
//   }
//   
//   baseRepo := core.NewBaseRepoImpl(db, &User{})
func NewBaseRepoImpl(db *gorm.DB, model interface{}) BaseRepoImpl {
	return BaseRepoImpl{
		RawBaseRepoImpl: NewRawBaseRepoImpl(db),
		model:           model,
	}
}

// Count returns the total number of records in the repository's table.
//
// This method provides a consistent way to get record counts across all repositories
// that extend BaseRepoImpl. It automatically uses the model information provided
// during repository construction.
//
// Parameters:
//   - ctx: Context for the database operation (may contain transaction)
//
// Returns:
//   - int: Total number of records in the table
//   - error: Database error (translated through core.TranslateError)
//
// Usage:
//   count, err := repo.Count(ctx)
//   if err != nil {
//       return fmt.Errorf("failed to count records: %w", err)
//   }
//   log.Printf("Found %d records", count)
func (b *BaseRepoImpl) Count(ctx context.Context) (int, error) {
	var count int64
	err := b.DB(ctx).Model(b.model).Count(&count).Error
	return int(count), TranslateError(err)
}

// DeleteAll removes all records from the repository's table.
//
// This method provides a way to completely clear a table, which is primarily useful
// for testing scenarios where you need a clean state. Use with extreme caution in
// production environments.
//
// Parameters:
//   - ctx: Context for the database operation (may contain transaction)
//
// Returns:
//   - error: Database error (translated through core.TranslateError)
//
// Warning:
//   This operation is irreversible and will remove ALL data from the table.
//   It should primarily be used in testing scenarios.
//
// Usage:
//   // Typically used in test cleanup
//   err := repo.DeleteAll(ctx)
//   if err != nil {
//       t.Fatalf("Failed to clean up test data: %v", err)
//   }
func (b *BaseRepoImpl) DeleteAll(ctx context.Context) error {
	return TranslateError(b.DB(ctx).Where("1 = 1").Delete(b.model).Error)
}

// key is an unexported type for context keys defined in this package.
// This prevents collisions with keys defined in other packages by ensuring
// that context keys are package-specific and not accidentally overwritten.
type key int

// dbKey is the context key for storing *gorm.DB transaction instances.
// This key is unexported to prevent direct access - clients should use
// core.NewContext and core.FromContext functions instead of accessing
// this key directly.
var dbKey key

// NewContext creates a new context that carries a GORM database transaction.
//
// This function is used internally to create transaction-aware contexts that
// carry database transaction information. Repository operations performed with
// this context will automatically participate in the transaction.
//
// Parameters:
//   - ctx: Base context to extend with transaction information
//   - db: GORM database transaction instance to store in context
//
// Returns:
//   - context.Context: New context containing the database transaction
//
// This function is primarily used internally by the Tx() method and should
// rarely be used directly by application code.
func NewContext(ctx context.Context, db *gorm.DB) context.Context {
	return context.WithValue(ctx, dbKey, db)
}

// FromContext retrieves the GORM database transaction from the context, if present.
//
// This function is used internally to detect whether the current context contains
// an ongoing database transaction, enabling transparent transaction support.
//
// Parameters:
//   - ctx: Context that may contain a database transaction
//
// Returns:
//   - *gorm.DB: Database transaction instance if present
//   - bool: True if transaction was found in context, false otherwise
//
// This function is primarily used internally by the DB() method and should
// rarely be used directly by application code.
func FromContext(ctx context.Context) (*gorm.DB, bool) {
	db, ok := ctx.Value(dbKey).(*gorm.DB)
	return db, ok
}
