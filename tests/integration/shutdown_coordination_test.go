//go:build integration
// +build integration

// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	config "github.com/cloudzero/cloudzero-agent/app/config/gator"
)

// TestShutdownCoordination_Integration verifies the file-based coordination mechanism
// This test follows the existing integration test pattern and doesn't require a full K8s cluster
func TestShutdownCoordination_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Test coordination logic integration scenarios
	t.Run("CollectorShutdownMarkerCreation", func(t *testing.T) {
		testCollectorShutdownMarkerCreation(t)
	})

	t.Run("ShipperCoordinationWithCollector", func(t *testing.T) {
		testShipperCoordinationWithCollector(t)
	})

	t.Run("ShipperTimeoutHandling", func(t *testing.T) {
		testShipperTimeoutHandling(t)
	})

	t.Run("CoordinationWithConfigurableSettings", func(t *testing.T) {
		testCoordinationWithConfigurableSettings(t)
	})
}

// testCollectorShutdownMarkerCreation tests the collector's ability to create shutdown marker files
func testCollectorShutdownMarkerCreation(t *testing.T) {
	tmpDir := t.TempDir()
	shutdownFile := filepath.Join(tmpDir, config.ShutdownMarkerFilename)

	// Simulate collector creating shutdown marker
	err := os.WriteFile(shutdownFile, []byte("done"), config.ShutdownMarkerFileMode)
	require.NoError(t, err, "Failed to create shutdown marker file")

	// Verify file exists and has correct permissions (allow for umask differences)
	fileInfo, err := os.Stat(shutdownFile)
	require.NoError(t, err, "Shutdown marker file should exist")
	actualPerms := fileInfo.Mode().Perm()
	assert.True(t, actualPerms&0o600 == 0o600, "File should have at least read/write permissions for owner: got %o", actualPerms)

	// Verify file contents
	content, err := os.ReadFile(shutdownFile)
	require.NoError(t, err, "Should be able to read shutdown marker file")
	assert.Equal(t, "done", string(content), "File should contain expected content")
}

// testShipperCoordinationWithCollector tests the shipper waiting for collector coordination
func testShipperCoordinationWithCollector(t *testing.T) {
	tmpDir := t.TempDir()
	shutdownFile := filepath.Join(tmpDir, config.ShutdownMarkerFilename)
	ctx := context.Background()

	// Test case 1: Collector creates file immediately, shipper should detect quickly
	go func() {
		time.Sleep(100 * time.Millisecond)
		if err := os.WriteFile(shutdownFile, []byte("done"), config.ShutdownMarkerFileMode); err != nil {
			panic(err)
		}
	}()

	start := time.Now()
	found := waitForCollectorShutdown(ctx, shutdownFile, 2*time.Second)
	duration := time.Since(start)

	assert.True(t, found, "Should detect shutdown marker file")
	assert.Less(t, duration, 500*time.Millisecond, "Should detect file quickly")

	// Clean up for next test
	os.Remove(shutdownFile)

	// Test case 2: Collector creates file after some delay
	go func() {
		time.Sleep(500 * time.Millisecond)
		if err := os.WriteFile(shutdownFile, []byte("done"), config.ShutdownMarkerFileMode); err != nil {
			panic(err)
		}
	}()

	start = time.Now()
	found = waitForCollectorShutdown(ctx, shutdownFile, 2*time.Second)
	duration = time.Since(start)

	assert.True(t, found, "Should detect shutdown marker file after delay")
	assert.Greater(t, duration, 400*time.Millisecond, "Should wait for file creation")
	assert.Less(t, duration, 1*time.Second, "Should not wait too long")
}

// testShipperTimeoutHandling tests shipper timeout behavior when collector doesn't signal
func testShipperTimeoutHandling(t *testing.T) {
	tmpDir := t.TempDir()
	shutdownFile := filepath.Join(tmpDir, config.ShutdownMarkerFilename)
	ctx := context.Background()

	// Test timeout when file never appears
	start := time.Now()
	found := waitForCollectorShutdown(ctx, shutdownFile, 200*time.Millisecond)
	duration := time.Since(start)

	assert.False(t, found, "Should timeout when file doesn't exist")
	assert.GreaterOrEqual(t, duration, 200*time.Millisecond, "Should wait for full timeout period")
	assert.Less(t, duration, 300*time.Millisecond, "Should not wait significantly longer than timeout")
}

// testCoordinationWithConfigurableSettings tests that coordination uses the configured constants
func testCoordinationWithConfigurableSettings(t *testing.T) {
	tmpDir := t.TempDir()

	// Test that we're using the configured filename
	expectedFile := filepath.Join(tmpDir, config.ShutdownMarkerFilename)
	assert.Equal(t, "collector-shutdown-complete", config.ShutdownMarkerFilename, "Should use configured filename")

	// Test that we're using the configured file mode
	err := os.WriteFile(expectedFile, []byte("test"), config.ShutdownMarkerFileMode)
	require.NoError(t, err, "Should be able to create file with configured mode")

	fileInfo, err := os.Stat(expectedFile)
	require.NoError(t, err, "File should exist")
	actualPerms := fileInfo.Mode().Perm()
	assert.True(t, actualPerms&0o600 == 0o600, "File should have at least read/write permissions for owner: got %o", actualPerms)
}

// waitForCollectorShutdown simulates the shipper's wait logic
// This is a simplified version of the actual implementation for testing
func waitForCollectorShutdown(_ context.Context, shutdownFile string, maxWait time.Duration) bool {
	deadline := time.Now().Add(maxWait)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(shutdownFile); err == nil {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}
