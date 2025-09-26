// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"context"
)

// StorageCommon defines transaction and bulk operation methods shared across all repository implementations.
// These operations are provided by BaseRepoImpl and enable consistent database operations
// with proper transaction management for data integrity in the CloudZero Agent.
type StorageCommon interface {
	// Tx executes a block of operations within a database transaction for atomicity.
	// The provided context includes transaction state that must be passed to all operations within the block.
	// Used for complex operations involving multiple tables or ensuring data consistency across operations.
	Tx(ctx context.Context, block func(ctxTx context.Context) error) error

	// Count returns the total number of records in the associated table.
	// Used for capacity monitoring, pagination calculations, and storage usage reporting.
	Count(ctx context.Context) (int, error)

	// DeleteAll removes all records from the associated table.
	// Used for data cleanup operations, testing teardown, and maintenance procedures.
	// Should be used with extreme caution in production environments.
	DeleteAll(ctx context.Context) error
}

// Storage defines the complete CRUD interface for CloudZero Agent data repositories.
// This generic interface combines all basic database operations into a unified contract
// that supports type-safe operations across different model types and ID formats.
// Used as the foundation for resource metadata, configuration, and operational data storage.
type Storage[Model any, ID comparable] interface {
	// Creator provides record creation capabilities for new data entities.
	Creator[Model]

	// Reader provides record retrieval capabilities for existing data lookup.
	Reader[Model, ID]

	// Updater provides record modification capabilities for data maintenance.
	Updater[Model]

	// Deleter provides record removal capabilities for data lifecycle management.
	Deleter[ID]
}

// Creator defines the interface for repository implementations that support record creation operations.
// This interface is part of the CRUD pattern used throughout the CloudZero Agent storage layer,
// enabling consistent data persistence across different model types and storage backends.
// Used for creating new metric records, configuration entries, and operational state data.
type Creator[Model any] interface {
	// Create persists a new record in the underlying storage system.
	// The method may modify the input Model during the creation process, typically to:
	// - Set auto-generated primary key IDs for database records
	// - Populate creation timestamps and audit fields
	// - Apply default values defined by the storage schema
	//
	// Used throughout the agent for persisting new metric data, configuration changes,
	// and operational state records. The generic type parameter enables type-safe
	// operations while maintaining a consistent interface across different model types.
	//
	// Returns an error if the creation operation fails due to:
	// - Database connectivity issues or transaction conflicts
	// - Constraint violations (unique keys, foreign keys, check constraints)
	// - Storage capacity limits or disk space exhaustion
	// - Invalid model data that fails validation during persistence
	Create(ctx context.Context, it *Model) error
}

// Reader defines the interface for repository implementations that support record retrieval operations.
// This interface provides the "R" in CRUD operations, enabling type-safe data access patterns
// across different model types and storage backends throughout the CloudZero Agent.
// Used for retrieving metric records, configuration data, and operational state information.
type Reader[Model any, ID comparable] interface {
	// Get retrieves a single record from the storage system by its unique identifier.
	// The generic ID parameter supports various identifier types (string, int64, UUID)
	// depending on the specific model and storage backend requirements.
	//
	// Used throughout the agent for:
	// - Loading specific metric records for analysis or reprocessing
	// - Retrieving configuration entries by key for runtime behavior
	// - Fetching operational state data for system monitoring
	//
	// Returns a pointer to the requested Model instance if found, or an error if:
	// - The record with the specified ID does not exist (types.ErrNotFound)
	// - Database connectivity issues prevent the query from executing
	// - Storage corruption or schema migration issues affect data access
	// - Context cancellation interrupts the retrieval operation
	//
	// The returned Model pointer should not be modified without proper synchronization
	// if the storage backend uses shared data structures or caching mechanisms.
	Get(ctx context.Context, id ID) (*Model, error)
}

// Updater defines the interface for repository implementations that support record modification operations.
// This interface provides the "U" in CRUD operations, enabling consistent data update patterns
// across different model types and storage backends throughout the CloudZero Agent.
// Used for modifying metric records, configuration updates, and operational state changes.
type Updater[Model any] interface {
	// Update modifies an existing record in the underlying storage system.
	// The method may modify the input Model during the update process, typically to:
	// - Set modification timestamps and audit trail information
	// - Apply version increments for optimistic locking mechanisms
	// - Normalize or validate field values according to business rules
	//
	// Used throughout the agent for:
	// - Updating metric processing status and classification results
	// - Modifying configuration settings during runtime reconfiguration
	// - Updating operational state data for system health monitoring
	//
	// The generic type parameter enables type-safe update operations while
	// maintaining interface consistency across different model types.
	//
	// Returns an error if the update operation fails due to:
	// - Record not found (the target record no longer exists)
	// - Concurrent modification conflicts (optimistic locking failures)
	// - Database connectivity issues or transaction rollback conditions
	// - Constraint violations resulting from the updated field values
	// - Storage capacity limits preventing the update from completing
	//
	// Implementations should ensure atomicity of updates and proper error
	// reporting to enable appropriate retry and recovery strategies.
	Update(ctx context.Context, it *Model) error
}

// Deleter defines the interface for repository implementations that support record removal operations.
// This interface provides the "D" in CRUD operations, enabling consistent data deletion patterns
// across different model types and storage backends throughout the CloudZero Agent.
// Used for removing outdated metrics, cleaning up temporary data, and lifecycle management.
type Deleter[ID comparable] interface {
	// Delete removes a record from the underlying storage system by its unique identifier.
	// The generic ID parameter supports various identifier types (string, int64, UUID)
	// matching the corresponding Reader interface for consistency.
	//
	// Used throughout the agent for:
	// - Removing processed metric files after successful upload to CloudZero
	// - Cleaning up expired configuration entries during system maintenance
	// - Deleting temporary operational state data after processing completion
	//
	// The operation should be idempotent - deleting a non-existent record
	// should not return an error to simplify cleanup logic and retry scenarios.
	//
	// Returns an error if the deletion operation fails due to:
	// - Database connectivity issues preventing the delete operation
	// - Foreign key constraints that prevent cascade deletion
	// - Storage system errors (disk full, permissions, corruption)
	// - Context cancellation interrupting the deletion process
	//
	// Implementations should handle referential integrity appropriately,
	// either through cascade deletion or by returning descriptive errors
	// when deletion would violate data consistency requirements.
	//
	// For systems requiring audit trails, implementations may perform
	// "soft deletes" (marking records as deleted) rather than physical removal.
	Delete(ctx context.Context, id ID) error
}
