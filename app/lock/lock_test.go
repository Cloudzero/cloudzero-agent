// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package lock

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLock_AcquireAndRelease(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "test.lock")

	fl := NewFileLock(context.Background(), lockPath)

	// aquire
	err := fl.Acquire()
	require.NoError(t, err)

	// ensure the file exists
	_, err = os.Stat(lockPath)
	require.NoError(t, err)

	// release
	err = fl.Release()
	require.NoError(t, err)

	// ensure the file was removed
	_, err = os.Stat(lockPath)
	require.Contains(t, err.Error(), "no such file or directory")
}

func TestLock_ConcurrentAccess(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "concurrent.lock")

	var wg sync.WaitGroup
	var locked int
	var mu sync.Mutex

	// spawn go routines that compete with the lock file
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// get the lock file
			localLock := NewFileLock(context.Background(), lockPath)
			if err := localLock.Acquire(); err != nil {
				return
			}
			defer localLock.Release()

			// increment the counter
			mu.Lock()
			locked++
			mu.Unlock()

			// hold the lock for a bit
			time.Sleep(100 * time.Millisecond)
		}()
	}

	wg.Wait()

	require.Equal(t, 5, locked)
}

func TestLock_StaleLockDetection(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "stale.lock")

	// create a manual stale lock file
	createStaleLock(t, lockPath, time.Now().Add(-1*time.Hour))

	fl := NewFileLock(context.Background(), lockPath, WithStaleTimeout(time.Minute*1), WithRefreshInterval(time.Second*10))

	// should succeed after
	err := fl.Acquire()
	require.NoError(t, err)
	defer fl.Release()

	// verify the lock content
	content, err := fl.readLockContent()
	require.NoError(t, err)
	require.LessOrEqual(t, time.Since(content.Timestamp), time.Second)
}

func TestLock_Refresh(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "refresh.lock")

	fl := NewFileLock(context.Background(), lockPath, WithStaleTimeout(time.Second*2), WithRefreshInterval(time.Millisecond*100))
	err := fl.Acquire()
	require.NoError(t, err)
	defer fl.Release()

	// Get initial timestamp
	initialTS, err := fl.readLockContent()
	require.NoError(t, err)

	// Wait for refresh
	time.Sleep(300 * time.Millisecond)

	// Check timestamp updated
	updatedTS, err := fl.readLockContent()
	require.NoError(t, err)
	require.NotEqual(t, initialTS.Timestamp, updatedTS.Timestamp)
}

func TestLock_Contention(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "contention.lock")

	fl1 := NewFileLock(context.Background(), lockPath)
	fl2 := NewFileLock(context.Background(), lockPath)

	// first aquires the lock
	err := fl1.Acquire()
	require.NoError(t, err)

	// second should block
	acquired := make(chan struct{})
	go func() {
		err := fl2.Acquire()
		require.NoError(t, err)
		close(acquired)
	}()

	// verify the second process does not aquire immediately
	select {
	case <-acquired:
		t.Fatal("Second process acquired lock too quickly")
	case <-time.After(100 * time.Millisecond):
	}

	// release the first lock
	err = fl1.Release()
	require.NoError(t, err)

	// now the second lock should aquire
	select {
	case <-acquired:
	case <-time.After(1 * time.Second):
		t.Fatal("Second process didn't acquire lock after release")
	}
}

func TestLock_DifferentHostsAndPIDs(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "host-pid.lock")

	// simulate a non-stale lock file from another host
	foreignContent := lockContent{
		Hostname:  "other-host",
		PID:       9999,
		Timestamp: time.Now(),
	}
	writeLockFile(t, lockPath, foreignContent)

	fl := NewFileLock(context.Background(), lockPath, WithStaleTimeout(time.Second*5), WithRefreshInterval(time.Millisecond*200))

	// capture aquisition results
	acquired := make(chan error, 1)
	go func() {
		acquired <- fl.Acquire()
	}()

	// verify we do not acquire within a reasonable timeframe
	select {
	case err := <-acquired:
		// Should NOT acquire the lock
		if errors.Is(err, nil) {
			current, _ := fl.readLockContent()
			t.Fatalf("acquired lock that should be held by another host. New lock content: %+v", current)
		}
		require.NoError(t, err)
	case <-time.After(1 * time.Second):
		// expected here
	}

	// verify the lock has the original content
	current, err := fl.readLockContent()
	require.NoError(t, err)

	if current.Hostname != foreignContent.Hostname || current.PID != foreignContent.PID {
		t.Fatalf("invalid lock content. Got %+v, want %+v", current, foreignContent)
	}

	// cleanup incase of errors
	fl.Release()
}

func TestLock_FileLossDetection(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "loss.lock")

	fl := NewFileLock(context.Background(), lockPath, WithRefreshInterval(time.Millisecond*200))
	err := fl.Acquire()
	require.NoError(t, err)

	// simulate lock loss by removing the file
	err = os.Remove(lockPath)
	require.NoError(t, err)

	// Wait for refresh cycle
	time.Sleep(200 * time.Millisecond)

	// Try to release - should already be released
	err = fl.Release()
	require.NoError(t, err)
}

func TestLock_MaxRetry(t *testing.T) {
	t.Parallel()
	tempDir := t.TempDir()
	lockPath := filepath.Join(tempDir, "loss.lock")

	fl1 := NewFileLock(context.Background(), lockPath)
	fl2 := NewFileLock(
		context.Background(),
		lockPath,
		WithRefreshInterval(time.Millisecond*200),
		WithMaxRetry(0), // only will try once
	)

	// aquire the lock
	err := fl1.Acquire()
	require.NoError(t, err)
	defer fl1.Release()

	// aquire from fl2, this should fail
	err = fl2.Acquire()
	require.Error(t, err)
	require.Equal(t, ErrMaxRetryExceeded, err)
}

// ---
// Helper functions
// ---

func createStaleLock(t *testing.T, path string, timestamp time.Time) {
	t.Helper()
	content := lockContent{
		Hostname:  "stale-host",
		PID:       12345,
		Timestamp: timestamp,
	}
	writeLockFile(t, path, content)
}

func writeLockFile(t *testing.T, path string, content lockContent) {
	t.Helper()
	data, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("Failed to marshal test lock: %v", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("Failed to write test lock: %v", err)
	}
}
