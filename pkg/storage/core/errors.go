// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"errors"

	"gorm.io/gorm"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/types"
)

// TranslateError maps GORM errors to application-specific errors.
// If the error does not match any known GORM errors, it returns the original error.
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
