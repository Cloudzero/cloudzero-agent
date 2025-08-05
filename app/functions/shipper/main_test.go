// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

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

func TestWaitForCollectorShutdown_FileExistsImmediately(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	shutdownFile := filepath.Join(tempDir, config.ShutdownMarkerFilename)
	ctx := context.Background()

	// Create the file immediately
	err := os.WriteFile(shutdownFile, []byte("done"), config.ShutdownMarkerFileMode)
	require.NoError(t, err)

	// Test
	start := time.Now()
	result := waitForCollectorShutdown(ctx, shutdownFile, 10*time.Second)
	elapsed := time.Since(start)

	// Assertions
	assert.True(t, result, "should return true when file exists immediately")
	assert.Less(t, elapsed, 1*time.Second, "should return quickly when file exists")
}

func TestWaitForCollectorShutdown_FileAppearsLater(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	shutdownFile := filepath.Join(tempDir, config.ShutdownMarkerFilename)
	ctx := context.Background()

	// Create file after a delay in a goroutine
	go func() {
		time.Sleep(500 * time.Millisecond)
		err := os.WriteFile(shutdownFile, []byte("done"), config.ShutdownMarkerFileMode)
		if err != nil {
			panic(err) // Will be caught by test harness
		}
	}()

	// Test
	start := time.Now()
	result := waitForCollectorShutdown(ctx, shutdownFile, 2*time.Second)
	elapsed := time.Since(start)

	// Assertions
	assert.True(t, result, "should return true when file appears during wait")
	assert.GreaterOrEqual(t, elapsed, 400*time.Millisecond, "should wait at least until file appears")
	assert.Less(t, elapsed, 1*time.Second, "should return shortly after file appears")
}

func TestWaitForCollectorShutdown_Timeout(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	shutdownFile := filepath.Join(tempDir, "nonexistent-file")
	ctx := context.Background()
	timeout := 500 * time.Millisecond

	// Test
	start := time.Now()
	result := waitForCollectorShutdown(ctx, shutdownFile, timeout)
	elapsed := time.Since(start)

	// Assertions
	assert.False(t, result, "should return false when file never appears")
	assert.GreaterOrEqual(t, elapsed, timeout, "should wait at least the full timeout")
	assert.Less(t, elapsed, timeout+200*time.Millisecond, "should not wait significantly longer than timeout")
}

func TestWaitForCollectorShutdown_ShortTimeout(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	shutdownFile := filepath.Join(tempDir, "nonexistent-file")
	ctx := context.Background()
	timeout := 50 * time.Millisecond // Very short timeout

	// Test
	start := time.Now()
	result := waitForCollectorShutdown(ctx, shutdownFile, timeout)
	elapsed := time.Since(start)

	// Assertions
	assert.False(t, result, "should return false with short timeout")
	assert.GreaterOrEqual(t, elapsed, timeout, "should respect short timeout")
	assert.Less(t, elapsed, 200*time.Millisecond, "should not wait much longer than short timeout")
}

func TestWaitForCollectorShutdown_InvalidPath(t *testing.T) {
	// Setup
	shutdownFile := "/invalid/path/that/does/not/exist/file"
	ctx := context.Background()
	timeout := 200 * time.Millisecond

	// Test
	start := time.Now()
	result := waitForCollectorShutdown(ctx, shutdownFile, timeout)
	elapsed := time.Since(start)

	// Assertions
	assert.False(t, result, "should return false for invalid path")
	assert.GreaterOrEqual(t, elapsed, timeout, "should still respect timeout with invalid path")
}

func TestWaitForCollectorShutdown_FileRemovedAndRecreated(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	shutdownFile := filepath.Join(tempDir, config.ShutdownMarkerFilename)
	ctx := context.Background()

	// Use a channel to coordinate the goroutine
	done := make(chan struct{})

	// Create file, then remove it, then recreate it
	go func() {
		defer close(done)
		// Create initially
		err := os.WriteFile(shutdownFile, []byte("done"), config.ShutdownMarkerFileMode)
		if err != nil {
			t.Errorf("Failed to create initial file: %v", err)
			return
		}

		time.Sleep(100 * time.Millisecond)

		// Remove it
		err = os.Remove(shutdownFile)
		if err != nil {
			t.Errorf("Failed to remove file: %v", err)
			return
		}

		time.Sleep(200 * time.Millisecond)

		// Recreate it
		err = os.WriteFile(shutdownFile, []byte("done"), config.ShutdownMarkerFileMode)
		if err != nil {
			t.Errorf("Failed to recreate file: %v", err)
			return
		}
	}()

	// Test - should succeed on first detection
	start := time.Now()
	result := waitForCollectorShutdown(ctx, shutdownFile, 2*time.Second)
	elapsed := time.Since(start)

	// Wait for goroutine to complete
	<-done

	// Assertions
	assert.True(t, result, "should return true when file is detected (even if briefly)")
	assert.Less(t, elapsed, 150*time.Millisecond, "should detect file quickly on first appearance")
}

func TestWaitForCollectorShutdown_ZeroTimeout(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	shutdownFile := filepath.Join(tempDir, "nonexistent-file")
	ctx := context.Background()

	// Test with zero timeout
	start := time.Now()
	result := waitForCollectorShutdown(ctx, shutdownFile, 0)
	elapsed := time.Since(start)

	// Assertions
	assert.False(t, result, "should return false with zero timeout")
	assert.Less(t, elapsed, 50*time.Millisecond, "should return immediately with zero timeout")
}

func TestWaitForCollectorShutdown_FilePermissions(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	shutdownFile := filepath.Join(tempDir, config.ShutdownMarkerFilename)
	ctx := context.Background()

	// Create file with different permissions
	err := os.WriteFile(shutdownFile, []byte("done"), 0o000) // No permissions
	require.NoError(t, err)

	// Cleanup - restore permissions so test cleanup can remove the file
	defer func() {
		os.Chmod(shutdownFile, 0o644)
	}()

	// Test
	result := waitForCollectorShutdown(ctx, shutdownFile, 200*time.Millisecond)

	// Assertions - os.Stat should still work even with no read permissions
	assert.True(t, result, "should detect file existence regardless of permissions")
}
