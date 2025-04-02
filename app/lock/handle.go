// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package lock

import (
	"context"
	"fmt"
	"path/filepath"
)

// LockFile acquires a lock for the specified file, executes the function, then
// releases the lock. The lock file is created in the same directory as
// `${filepath}.lock`
func LockFile(ctx context.Context, filePath string, fn func() error, opts ...FileLockOption) error {
	lockPath := getFileLockPath(filePath)
	return withLock(ctx, lockPath, fn, opts...)
}

// LockDir acquires a lock for the specified directory, executes the function,
// then releases the lock. The lock file is created within the target directory
// as `.dir.lock`.
func LockDir(ctx context.Context, dirPath string, fn func() error, opts ...FileLockOption) error {
	lockPath := getDirLockPath(dirPath)
	return withLock(ctx, lockPath, fn, opts...)
}

// get a file lock
func getFileLockPath(filePath string) string {
	dir := filepath.Dir(filePath)
	base := filepath.Base(filePath)
	return filepath.Join(dir, base+".lock")
}

// get a directory lock
func getDirLockPath(dirPath string) string {
	return filepath.Join(dirPath, ".dir.lock")
}

// withLock handles the common locking logic using the FileLock type
func withLock(ctx context.Context, lockPath string, fn func() error, opts ...FileLockOption) error {
	// get the lock
	fl := NewFileLock(ctx, lockPath, opts...)
	err := fl.Acquire()
	if err != nil {
		return fmt.Errorf("failed to acquire the lock: %w", err)
	}

	// run the user defined function
	if err := fn(); err != nil {
		if err2 := fl.Release(); err2 != nil {
			return fmt.Errorf("failed to release the lock in the error context: %w - %w", err, err2)
		}
		return err
	}

	// release the lock
	if err := fl.Release(); err != nil {
		return fmt.Errorf("failed to release the lock: %w", err)
	}

	return nil
}
