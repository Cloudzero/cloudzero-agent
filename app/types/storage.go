// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package types defines the core interfaces and data structures for the CloudZero agent.
//
// This package serves as the foundation for the entire agent architecture by providing:
//
//   - Storage interfaces: Generic CRUD operations for different persistence layers
//   - Data models: Core structures for metrics, resources, and configuration
//   - Interface contracts: Abstractions that decouple business logic from implementations
//   - Type definitions: Common types used throughout the agent codebase
//
// Key design patterns:
//   - Generic interfaces using Go 1.18+ type parameters for type safety
//   - Repository pattern for data access abstraction
//   - Clear separation between storage operations (CRUD) and business logic
//   - Composable interfaces that can be implemented by different storage backends
//
// Storage abstraction hierarchy:
//   - StorageCommon: Base operations like transactions and record counting
//   - Creator/Reader/Updater/Deleter: Specific CRUD operation interfaces
//   - Storage[Model, ID]: Complete CRUD interface combining all operations
//
// Usage patterns:
//   Implementations in app/storage/ provide concrete storage backends (disk, sqlite)
//   that implement these interfaces, allowing the domain layer to work with
//   any storage implementation through these abstractions.
//
// This design enables:
//   - Easy testing with mock implementations
//   - Flexible storage backend selection
//   - Clean domain logic separated from storage concerns
//   - Type-safe operations with compile-time validation
package types

import (
	"context"
)

// StorageCommon defines common methods that all repository implementations provide
// by virtue of using BaseRepoImpl. These operations are fundamental to repository
// management and are available regardless of the specific model type.
//
// This interface provides:
//   - Transaction management for atomic operations
//   - Record counting for monitoring and pagination
//   - Bulk deletion for cleanup and testing scenarios
type StorageCommon interface {
	// Tx runs the provided block within a database transaction.
	// The block receives a transaction context that should be used for all
	// database operations within the transaction. If the block returns an error,
	// the transaction is rolled back; otherwise, it is committed.
	//
	// Parameters:
	//   - ctx: Base context for the transaction
	//   - block: Function to execute within the transaction scope
	//
	// Returns:
	//   - error: Transaction error or error from the block function
	//
	// Usage:
	//   err := repo.Tx(ctx, func(txCtx context.Context) error {
	//       // All operations in this block are transactional
	//       return repo.Create(txCtx, &model)
	//   })
	Tx(ctx context.Context, block func(ctxTx context.Context) error) error
	
	// Count returns the total number of records in the repository.
	// This is useful for monitoring, pagination, and capacity planning.
	//
	// Returns:
	//   - int: Total record count
	//   - error: Database access error
	Count(ctx context.Context) (int, error)
	
	// DeleteAll removes all records from the repository.
	// This is primarily used for testing cleanup and administrative operations.
	// Use with caution in production environments.
	//
	// Returns:
	//   - error: Database operation error
	DeleteAll(ctx context.Context) error
}

// Storage is a complete CRUD interface that combines all basic data access operations
// for a specific model type. It provides type-safe storage operations using Go generics,
// ensuring compile-time validation of model types and ID types.
//
// Type parameters:
//   - Model: The data model type to be stored (e.g., Metric, Resource, etc.)
//   - ID: The identifier type used for model lookup (e.g., uuid.UUID, string, int)
//
// This interface composes four fundamental operation interfaces:
//   - Creator[Model]: For creating new records
//   - Reader[Model, ID]: For retrieving records by ID
//   - Updater[Model]: For modifying existing records
//   - Deleter[ID]: For removing records by ID
//
// Usage pattern:
//   var repo Storage[Metric, uuid.UUID]
//   repo = NewDiskStore[Metric, uuid.UUID](config)
//   
//   metric := &Metric{Name: "cpu_usage"}
//   err := repo.Create(ctx, metric)  // Type-safe creation
//   found, err := repo.Get(ctx, metric.ID)  // Type-safe retrieval
type Storage[Model any, ID comparable] interface {
	Creator[Model]
	Reader[Model, ID]
	Updater[Model]
	Deleter[ID]
}

// Creator is an interface that defines the method that must be implemented by a
// repository that provides access to records that can be created.
type Creator[Model any] interface {
	// Create creates a new record in the database. It may modify the input Model
	// along the way (e.g. to set the ID). It returns an error if there was a
	// problem creating the record.
	Create(ctx context.Context, it *Model) error
}

// Reader is an interface that defines the method that must be implemented by a
// repository that provides access to records that can be read.
type Reader[Model any, ID comparable] interface {
	// Get retrieves a record from the database by ID. It returns an error if there
	// was a problem retrieving the record.
	Get(ctx context.Context, id ID) (*Model, error)
}

// Updater is an interface that defines the method that must be implemented by a
// repository that provides access to records that can be updated.
type Updater[Model any] interface {
	// Update updates an existing record in the database. It may modify the input
	// Model along the way (e.g. to set the updated_at timestamp). It returns an
	// error if there was a problem updating the record.
	Update(ctx context.Context, it *Model) error
}

// Deleter is an interface that defines the method that must be implemented by a
// repository that provides access to records that can be deleted.
type Deleter[ID comparable] interface {
	// Delete deletes a record from the database by ID. It returns an error if
	// there was a problem deleting the record.
	Delete(ctx context.Context, id ID) error
}
