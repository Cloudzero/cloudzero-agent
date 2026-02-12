// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package core_test

import (
	"testing"

	"github.com/cloudzero/cloudzero-agent/app/storage/core"
	"github.com/cloudzero/cloudzero-agent/app/storage/sqlite"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/stretchr/testify/assert"
)

func TestTranslateError_SQLiteTableMissing(t *testing.T) {
	// Create an in-memory SQLite database
	db, err := sqlite.NewSQLiteDriver(sqlite.MemorySharedCached)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	// Try to query a non-existent table to get the real SQLite error
	var results []struct {
		ID   uint `gorm:"primaryKey"`
		Name string
	}
	err = db.Table("resource_tags").Find(&results).Error

	// Verify we got the expected SQLite error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no such table: resource_tags")

	// Test our TranslateError function
	translatedErr := core.TranslateError(err)
	assert.Error(t, translatedErr)
	assert.Equal(t, types.ErrTableMissing, translatedErr)
}

func TestTranslateError_UnknownError(t *testing.T) {
	// Test that unknown errors are passed through unchanged
	unknownErr := assert.AnError
	result := core.TranslateError(unknownErr)
	assert.Equal(t, unknownErr, result)
}
