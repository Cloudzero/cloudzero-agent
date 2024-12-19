// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package core

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

// NewDriver creates a standard *gorm.DB for the database dialect passed in.
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

// DatabaseNow returns time.Now() in UTC, truncated to Milliseconds.
func DatabaseNow() time.Time {
	return time.Now().UTC().Truncate(time.Millisecond)
}

// DatabaseNowPointer returns a pointer to a time.Time created by DatabaseNow
func DatabaseNowPointer() *time.Time {
	now := DatabaseNow()
	return &now
}
