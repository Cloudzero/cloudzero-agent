// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	config "github.com/cloudzero/cloudzero-agent/app/config/gator"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/cloudzero/cloudzero-agent/app/types/mocks"
)

// Helper function to test the core shutdown logic without signal handling
func performShutdownSequence(_ context.Context, settings *config.Settings, stores ...types.WritableStore) {
	// Simulate the core logic from HandleShutdownEvents after signal is received
	for _, appendable := range stores {
		appendable.Flush()
	}

	// Signal shutdown completion to shipper via file marker
	shutdownFile := filepath.Join(settings.Database.StoragePath, config.ShutdownMarkerFilename)
	if err := os.WriteFile(shutdownFile, []byte("done"), 0644); err != nil {
		// In real code this would log an error, for tests we can ignore
		_ = err
	}
}

func TestPerformShutdownSequence_FileCreation(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	settings := &config.Settings{
		Database: config.Database{
			StoragePath: tempDir,
		},
	}
	ctx := context.Background()
	
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	
	mockStore1 := mocks.NewMockStore(ctrl)
	mockStore2 := mocks.NewMockStore(ctrl)
	
	// Set expectations
	mockStore1.EXPECT().Flush().Return(nil).Times(1)
	mockStore2.EXPECT().Flush().Return(nil).Times(1)

	// Test
	performShutdownSequence(ctx, settings, mockStore1, mockStore2)

	// Assertions
	expectedFile := filepath.Join(tempDir, config.ShutdownMarkerFilename)
	
	// Check that shutdown marker file was created
	assert.FileExists(t, expectedFile, "shutdown marker file should be created")
	
	// Check file contents
	content, err := os.ReadFile(expectedFile)
	require.NoError(t, err)
	assert.Equal(t, "done", string(content), "shutdown marker should contain 'done'")
}

func TestPerformShutdownSequence_FileCreationWithInvalidPath(t *testing.T) {
	// Setup with invalid storage path
	settings := &config.Settings{
		Database: config.Database{
			StoragePath: "/invalid/path/that/does/not/exist",
		},
	}
	ctx := context.Background()
	
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	
	mockStore := mocks.NewMockStore(ctrl)
	mockStore.EXPECT().Flush().Return(nil).Times(1)

	// Test - should not panic even with invalid path
	performShutdownSequence(ctx, settings, mockStore)

	// Assertions
	expectedFile := filepath.Join(settings.Database.StoragePath, config.ShutdownMarkerFilename)
	
	// File should not exist due to invalid path
	assert.NoFileExists(t, expectedFile, "shutdown marker file should not be created with invalid path")
}

func TestPerformShutdownSequence_StoreFlushError(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	settings := &config.Settings{
		Database: config.Database{
			StoragePath: tempDir,
		},
	}
	ctx := context.Background()
	
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	
	mockStore := mocks.NewMockStore(ctrl)
	mockStore.EXPECT().Flush().Return(assert.AnError).Times(1)

	// Test
	performShutdownSequence(ctx, settings, mockStore)

	// Assertions
	expectedFile := filepath.Join(tempDir, config.ShutdownMarkerFilename)
	
	// Should still create marker file even if flush fails
	assert.FileExists(t, expectedFile, "shutdown marker file should be created even if flush fails")
	
	content, err := os.ReadFile(expectedFile)
	require.NoError(t, err)
	assert.Equal(t, "done", string(content), "shutdown marker should contain 'done'")
}

func TestPerformShutdownSequence_NoStores(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	settings := &config.Settings{
		Database: config.Database{
			StoragePath: tempDir,
		},
	}
	ctx := context.Background()

	// Test with no stores
	performShutdownSequence(ctx, settings)

	// Assertions
	expectedFile := filepath.Join(tempDir, config.ShutdownMarkerFilename)
	assert.FileExists(t, expectedFile, "shutdown marker file should be created even with no stores")
}

func TestPerformShutdownSequence_MultipleStores(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	settings := &config.Settings{
		Database: config.Database{
			StoragePath: tempDir,
		},
	}
	ctx := context.Background()
	
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	
	stores := make([]*mocks.MockStore, 5)
	for i := range stores {
		stores[i] = mocks.NewMockStore(ctrl)
		stores[i].EXPECT().Flush().Return(nil).Times(1)
	}

	// Convert to interface slice
	var storeInterfaces []types.WritableStore
	for _, store := range stores {
		storeInterfaces = append(storeInterfaces, store)
	}

	// Test
	performShutdownSequence(ctx, settings, storeInterfaces...)

	// Assertions
	expectedFile := filepath.Join(tempDir, config.ShutdownMarkerFilename)
	assert.FileExists(t, expectedFile, "shutdown marker file should be created with multiple stores")
}

func TestShutdownMarkerFilename_Constant(t *testing.T) {
	// Test that the constant is properly defined and accessible
	assert.Equal(t, "collector-shutdown-complete", config.ShutdownMarkerFilename)
	assert.NotEmpty(t, config.ShutdownMarkerFilename, "shutdown marker filename should not be empty")
}

func TestShutdownMarkerFile_PathConstruction(t *testing.T) {
	// Test various path scenarios
	testCases := []struct {
		name        string
		storagePath string
		expected    string
	}{
		{
			name:        "simple path",
			storagePath: "/data",
			expected:    "/data/collector-shutdown-complete",
		},
		{
			name:        "path with trailing slash",
			storagePath: "/data/",
			expected:    "/data/collector-shutdown-complete",
		},
		{
			name:        "relative path",
			storagePath: "./test-data",
			expected:    "test-data/collector-shutdown-complete",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := filepath.Join(tc.storagePath, config.ShutdownMarkerFilename)
			assert.Equal(t, tc.expected, result)
		})
	}
}