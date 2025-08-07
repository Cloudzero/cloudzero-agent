// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package core provides error translation utilities that map GORM-specific errors
// to application-level error types for consistent error handling across the CloudZero agent.
//
// Error translation ensures that:
//   - Repository implementations return consistent error types
//   - Business logic doesn't depend on ORM-specific error types
//   - Error handling is predictable across different storage backends
//   - Application errors can be properly categorized and handled
//
// The translation layer maps common database errors like:
//   - Record not found errors for missing data
//   - Constraint violations for data integrity issues
//   - Transaction errors for concurrency problems
//   - Validation errors for invalid data scenarios
//
// Usage:
//   err := db.First(&user, id).Error
//   return core.TranslateError(err)  // Returns types.ErrNotFound instead of gorm.ErrRecordNotFound
package core

import (
	"errors"

	"gorm.io/gorm"

	"github.com/cloudzero/cloudzero-agent/app/types"
)

// TranslateError converts GORM-specific database errors to application-level error types.
//
// This function provides a centralized error translation layer that maps GORM ORM errors
// to consistent application error types defined in the types package. This abstraction
// ensures that business logic doesn't depend on ORM implementation details and enables
// consistent error handling across different storage backends.
//
// Error categories handled:
//   - Data access errors: Record not found, invalid queries
//   - Constraint violations: Duplicate keys, foreign key violations, check constraints
//   - Transaction errors: Invalid transactions, concurrent access issues
//   - Schema errors: Missing primary keys, invalid field access
//   - Validation errors: Invalid data, missing required values
//
// Parameters:
//   - err: GORM error to translate (may be nil)
//
// Returns:
//   - error: Application-level error type or original error if no mapping exists
//
// Behavior:
//   - Returns nil if input error is nil
//   - Maps known GORM errors to types package error constants
//   - Returns original error unchanged if no mapping is available
//
// Usage:
//   var user User
//   err := db.Where("email = ?", email).First(&user).Error
//   return &user, core.TranslateError(err)
//
// This enables consistent error handling:
//   user, err := repo.FindByEmail(ctx, email)
//   if errors.Is(err, types.ErrNotFound) {
//       // Handle missing user consistently across all repos
//   }
func TranslateError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return types.ErrNotFound
	}

	switch {
	case errors.Is(err, gorm.ErrDuplicatedKey):
		return types.ErrDuplicateKey
	case errors.Is(err, gorm.ErrForeignKeyViolated):
		return types.ErrForeignKeyViolation
	case errors.Is(err, gorm.ErrInvalidTransaction):
		return types.ErrInvalidTransaction
	case errors.Is(err, gorm.ErrNotImplemented):
		return types.ErrNotImplemented
	case errors.Is(err, gorm.ErrMissingWhereClause):
		return types.ErrMissingWhereClause
	case errors.Is(err, gorm.ErrUnsupportedRelation):
		return types.ErrUnsupportedRelation
	case errors.Is(err, gorm.ErrPrimaryKeyRequired):
		return types.ErrPrimaryKeyRequired
	case errors.Is(err, gorm.ErrModelValueRequired):
		return types.ErrModelValueRequired
	case errors.Is(err, gorm.ErrModelAccessibleFieldsRequired):
		return types.ErrModelAccessibleFieldsRequired
	case errors.Is(err, gorm.ErrSubQueryRequired):
		return types.ErrSubQueryRequired
	case errors.Is(err, gorm.ErrInvalidData):
		return types.ErrInvalidData
	case errors.Is(err, gorm.ErrUnsupportedDriver):
		return types.ErrUnsupportedDriver
	case errors.Is(err, gorm.ErrRegistered):
		return types.ErrAlreadyRegistered
	case errors.Is(err, gorm.ErrInvalidField):
		return types.ErrInvalidField
	case errors.Is(err, gorm.ErrEmptySlice):
		return types.ErrEmptySlice
	case errors.Is(err, gorm.ErrDryRunModeUnsupported):
		return types.ErrDryRunModeUnsupported
	case errors.Is(err, gorm.ErrInvalidDB):
		return types.ErrInvalidDB
	case errors.Is(err, gorm.ErrInvalidValue):
		return types.ErrInvalidValue
	case errors.Is(err, gorm.ErrInvalidValueOfLength):
		return types.ErrInvalidValueLength
	case errors.Is(err, gorm.ErrPreloadNotAllowed):
		return types.ErrPreloadNotAllowed
	case errors.Is(err, gorm.ErrCheckConstraintViolated):
		return types.ErrCheckConstraintViolated
	}
	return err
}
