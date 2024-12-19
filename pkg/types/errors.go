package types

import (
	"errors"
)

// General Errors
var (
	// ErrNotFound is returned when a specific item or record is not found.
	ErrNotFound = errors.New("not found")

	// ErrDuplicateKey is returned when a duplicate key is detected during an indexing operation.
	ErrDuplicateKey = errors.New("duplicate key")

	// ErrForeignKeyViolation is returned when a foreign key constraint is violated.
	ErrForeignKeyViolation = errors.New("foreign key violation")

	// ErrMalformedKey is returned when a key is malformed during an indexing operation.
	ErrMalformedKey = errors.New("malformed key")

	// ErrMultipleItemsFound is returned when multiple items are found when only one was expected.
	ErrMultipleItemsFound = errors.New("multiple items found")
)

// Validation Errors
var (
	// ErrMissingIndices is returned when no indices are provided for an operation.
	ErrMissingIndices = errors.New("no indices provided")

	// ErrMissingItem is returned when no item is provided for an operation.
	ErrMissingItem = errors.New("no item provided")

	// ErrMissingIndex is returned when no index is provided for an operation.
	ErrMissingIndex = errors.New("no index provided")

	// ErrMissingKey is returned when no key is provided for an operation.
	ErrMissingKey = errors.New("no key provided")

	// ErrMissingQuery is returned when no query is provided for an operation.
	ErrMissingQuery = errors.New("no query provided")

	// ErrMissingValue is returned when no value is provided for an operation.
	ErrMissingValue = errors.New("no value provided")

	// ErrPrimaryKeyRequired is returned when a primary key is required but not provided.
	ErrPrimaryKeyRequired = errors.New("primary key required")

	// ErrModelValueRequired is returned when a model value is required but not provided.
	ErrModelValueRequired = errors.New("model value required")

	// ErrModelAccessibleFieldsRequired is returned when accessible fields are required but not provided.
	ErrModelAccessibleFieldsRequired = errors.New("model accessible fields required")

	// ErrSubQueryRequired is returned when a subquery is required but not provided.
	ErrSubQueryRequired = errors.New("subquery required")

	// ErrInvalidData is returned when the provided data is invalid.
	ErrInvalidData = errors.New("invalid data")

	// ErrInvalidField is returned when a provided field is invalid.
	ErrInvalidField = errors.New("invalid field")

	// ErrInvalidValue is returned when a provided value is invalid.
	ErrInvalidValue = errors.New("invalid value")

	// ErrInvalidValueLength is returned when a provided value does not meet length requirements.
	ErrInvalidValueLength = errors.New("invalid value length")

	// ErrEmptySlice is returned when an operation is performed on an empty slice.
	ErrEmptySlice = errors.New("empty slice")
)

// Operational Errors
var (
	// ErrNotReady is returned when an operation is attempted on a component that is not ready.
	ErrNotReady = errors.New("not ready")

	// ErrInvalidTransaction is returned when a transaction is invalid.
	ErrInvalidTransaction = errors.New("invalid transaction")

	// ErrNotImplemented is returned when a feature or function is not implemented.
	ErrNotImplemented = errors.New("not implemented")

	// ErrMissingWhereClause is returned when a where clause is missing in a query.
	ErrMissingWhereClause = errors.New("missing where clause")

	// ErrUnsupportedRelation is returned when an unsupported relation is encountered.
	ErrUnsupportedRelation = errors.New("unsupported relation")

	// ErrUnsupportedDriver is returned when an unsupported database driver is used.
	ErrUnsupportedDriver = errors.New("unsupported driver")

	// ErrAlreadyRegistered is returned when attempting to register something that's already registered.
	ErrAlreadyRegistered = errors.New("already registered")

	// ErrDryRunModeUnsupported is returned when dry run mode is unsupported.
	ErrDryRunModeUnsupported = errors.New("dry run mode unsupported")

	// ErrInvalidDB is returned when an invalid database instance is provided.
	ErrInvalidDB = errors.New("invalid database")

	// ErrPreloadNotAllowed is returned when preloading is not permitted in a query.
	ErrPreloadNotAllowed = errors.New("preload not allowed")

	// ErrCheckConstraintViolated is returned when a check constraint is violated.
	ErrCheckConstraintViolated = errors.New("check constraint violated")
)
