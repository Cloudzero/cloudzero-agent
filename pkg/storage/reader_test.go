// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/types"
)

func TestReader_ReadData(t *testing.T) {
	db := SetupDatabase()

	now := time.Now().UTC()
	testNamespace := "test-namespace"
	records := []types.ResourceTags{
		{Type: 1, Name: "test-deploy", Namespace: &testNamespace, RecordUpdated: now.Add(-2 * time.Hour), RecordCreated: now.Add(-3 * time.Hour)},
		{Type: 2, Name: "test-sts", Namespace: &testNamespace, RecordUpdated: now.Add(-1 * time.Hour), RecordCreated: now.Add(-2 * time.Hour)},
	}
	for _, record := range records {
		db.Create(&record)
	}

	settings := &config.Settings{
		RemoteWrite: config.RemoteWrite{
			MaxBytesPerSend: 50,
		},
	}
	reader := NewReader(db, settings)

	// Test ReadData
	result, err := reader.ReadData(now)
	assert.NoError(t, err)
	assert.Len(t, result, 2)

	// Test ReadData with no matching records
	ct := now.Add(-4 * time.Hour)
	result, err = reader.ReadData(ct)
	assert.NoError(t, err)
	assert.Len(t, result, 0)

	// Test ReadData with error
	badDb, _ := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	reader = NewReader(badDb, settings)
	result, err = reader.ReadData(ct)
	assert.Error(t, err)
	assert.Nil(t, result)
	t.Cleanup(func() {
		dbCleanup(db)
		dbCleanup(badDb)
	})
}
