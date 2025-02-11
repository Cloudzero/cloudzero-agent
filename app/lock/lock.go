package lock

import (
	"os"
	"path/filepath"

	"github.com/gofrs/flock"
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

// withLock handles the common locking logic using flock.
func withLock(lockPath string, fn func() error) error {
	fileLock := flock.New(lockPath)

	// get a lock with flock
	if err := fileLock.Lock(); err != nil {
		return err
	}
	defer func() {
		// clear the file lock
		_ = fileLock.Unlock()
		_ = os.Remove(lockPath)
	}()

	// execute the function
	return fn()
}
