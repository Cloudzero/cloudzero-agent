package lock

import (
	"context"
	"fmt"
	"path/filepath"
)

// `LockFile` acquires a lock for the specified file, executes the function, then releases the lock.
// The lock file is created in the same directory as `${filepath}.lock`
func LockFile(filePath string, fn func() error) error {
	lockPath := getFileLockPath(filePath)
	return withLock(lockPath, fn)
}

// `LockDir` acquires a lock for the specified directory, executes the function, then releases the lock.
// The lock file is created within the target directory as `.dir.lock`.
func LockDir(dirPath string, fn func() error) error {
	lockPath := getDirLockPath(dirPath)
	return withLock(lockPath, fn)
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
func withLock(lockPath string, fn func() error) error {
	// get the lock
	fl := NewFileLock(context.Background(), lockPath)
	err := fl.Acquire()
	if err != nil {
		return fmt.Errorf("failed to aquire the lock: %w", err)
	}

	// run the user defined function
	if err := fn(); err != nil {
		fl.Release()
		return err
	}

	// release the lock
	if err := fl.Release(); err != nil {
		return fmt.Errorf("failed to release the lock: %w", err)
	}

	return nil
}
