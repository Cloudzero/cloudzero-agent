// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package lock provides robust file-based distributed locking mechanisms for coordinating
// access to shared resources across multiple processes and hosts in the CloudZero agent.
//
// This package implements a sophisticated file-based locking system with features designed
// for production reliability and multi-process coordination:
//
//   - Atomic lock acquisition using O_EXCL file creation
//   - Stale lock detection and automatic cleanup
//   - Process and hostname-based lock ownership verification
//   - Background lock refresh to maintain ownership
//   - Configurable retry logic with exponential backoff
//   - Context-aware cancellation and timeout support
//   - Safe cleanup and release mechanisms
//
// Key features:
//   - Distributed coordination: Works across multiple processes/hosts
//   - Stale lock recovery: Automatically detects and cleans up abandoned locks
//   - Ownership verification: Each lock contains hostname and PID information
//   - Background refresh: Periodic timestamp updates prevent false stale detection
//   - Atomic operations: Uses file system atomicity for race-free lock operations
//
// Lock file format:
//   The lock file contains JSON with ownership and timestamp information:
//   {
//     "hostname": "worker-node-1",
//     "pid": 12345,
//     "timestamp": "2023-11-15T10:30:00Z"
//   }
//
// Usage patterns:
//   // Simple lock acquisition
//   lock := NewFileLock(ctx, "/tmp/my-process.lock")
//   if err := lock.Acquire(); err != nil {
//       log.Fatal("Failed to acquire lock:", err)
//   }
//   defer lock.Release()
//
//   // Configurable lock with custom timeouts
//   lock := NewFileLock(ctx, "/tmp/my-process.lock",
//       WithStaleTimeout(30*time.Second),
//       WithRetryInterval(5*time.Second),
//       WithMaxRetry(10))
//
// Use cases in CloudZero agent:
//   - Preventing multiple collector instances from running simultaneously
//   - Coordinating shipper uploads to avoid conflicts
//   - Ensuring single-instance webhook server deployment
//   - Synchronizing file processing across multiple agents
package lock

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	// lockFilePermissions defines the file permissions for lock files (readable by all, writable by owner)
	lockFilePermissions = 0o644
)

var (
	// Error definitions for various lock operation failures
	ErrLockExists           = errors.New("lock already exists")
	ErrLockStale            = errors.New("stale lock detected")
	ErrLockLost             = errors.New("lock lost")
	ErrLockAcquire          = errors.New("failed to acquire lock")
	ErrLockContextCancelled = errors.New("context was cancelled while obtaining the lock")
	ErrLockCorrup           = errors.New("corrupt lock file")
	ErrMaxRetryExceeded     = errors.New("failed to acquire lock, max retries exceeded")
	
	// Default configuration values for lock behavior
	// DefaultStaleTimeout: How long before a lock is considered abandoned
	DefaultStaleTimeout     = time.Millisecond * 500
	// DefaultRefreshInterval: How often to update lock timestamp to prove ownership
	DefaultRefreshInterval  = time.Millisecond * 200
	// DefaultRetryInterval: How long to wait between lock acquisition attempts
	DefaultRetryInterval    = 1 * time.Second
	// DefaultMaxRetry: Maximum number of acquisition attempts before giving up
	DefaultMaxRetry         = 5
)

// FileLock represents a file-based distributed lock with stale detection and ownership tracking.
//
// This structure manages the complete lifecycle of a file-based lock, including acquisition,
// ownership maintenance, and safe release. It supports configurable behavior through functional
// options and provides robust coordination across multiple processes.
//
// Key components:
//   - Lock file management: Creates and maintains lock files with JSON metadata
//   - Ownership tracking: Records hostname and PID for verification
//   - Background refresh: Periodically updates timestamps to maintain ownership
//   - Stale detection: Automatically cleans up abandoned locks
//   - Context integration: Respects cancellation and timeouts
//
// Thread safety:
//   All operations are protected by internal mutex to ensure thread-safe usage
//   from multiple goroutines within the same process.
type FileLock struct {
	// Configuration fields
	filepath        string        // Path to the lock file
	staleTimeout    time.Duration // How long before considering a lock stale
	refreshInterval time.Duration // How often to refresh the lock timestamp
	retryInterval   time.Duration // How long to wait between acquisition attempts
	maxRetry        int           // Maximum number of retry attempts

	// Identity fields for ownership verification
	hostname string // Hostname of the lock owner
	pid      int    // Process ID of the lock owner
	
	// Concurrency management
	ctx      context.Context    // Base context for operations
	cancel   context.CancelFunc // Cancel function for background refresh
	mu       sync.Mutex         // Mutex protecting concurrent access
	wg       sync.WaitGroup     // WaitGroup for background goroutine coordination
}

// FileLockOption represents a configuration option for customizing FileLock behavior.
// Options are applied using the functional options pattern to provide flexible
// configuration while maintaining backward compatibility.
type FileLockOption func(fl *FileLock)

// WithStaleTimeout sets the duration after which a lock is considered stale and can be forcibly acquired.
//
// A lock is considered stale when its timestamp hasn't been updated within the stale timeout period.
// This mechanism prevents permanently stuck locks from processes that crash without cleaning up.
//
// Parameters:
//   - timeout: Duration after which a lock becomes stale (should be > refresh interval)
//
// Returns:
//   - FileLockOption: Configuration function for NewFileLock
//
// Recommended values:
//   - Fast operations: 1-5 seconds
//   - Long-running operations: 30-60 seconds
//   - Critical sections: 5-15 seconds
func WithStaleTimeout(timeout time.Duration) FileLockOption {
	return func(fl *FileLock) {
		fl.staleTimeout = timeout
	}
}

// WithRetryInterval sets the wait time between lock acquisition attempts.
//
// When a lock is already held by another process, the acquisition will wait
// for this interval before attempting again. This prevents busy-waiting and
// reduces system load during contention.
//
// Parameters:
//   - interval: Time to wait between acquisition attempts
//
// Returns:
//   - FileLockOption: Configuration function for NewFileLock
//
// Recommended values:
//   - High contention: 100ms-1s
//   - Normal usage: 1-5s
//   - Background processes: 5-30s
func WithRetryInterval(interval time.Duration) FileLockOption {
	return func(fl *FileLock) {
		fl.retryInterval = interval
	}
}

// WithRefreshInterval sets how often the lock timestamp is updated to prove ownership.
//
// The background refresh process periodically updates the lock file timestamp
// to prove that the owning process is still alive. This should be significantly
// shorter than the stale timeout to prevent false stale detection.
//
// Parameters:
//   - interval: Time between timestamp refresh operations
//
// Returns:
//   - FileLockOption: Configuration function for NewFileLock
//
// Recommended values:
//   - Should be 1/3 to 1/5 of stale timeout
//   - Typical range: 100ms-5s
//   - Balance between responsiveness and I/O overhead
func WithRefreshInterval(interval time.Duration) FileLockOption {
	return func(fl *FileLock) {
		fl.refreshInterval = interval
	}
}

// WithMaxRetry sets the maximum number of lock acquisition attempts before giving up.
//
// After this many failed attempts, the Acquire() method will return ErrMaxRetryExceeded.
// This prevents indefinite blocking when a lock cannot be acquired.
//
// Parameters:
//   - retry: Maximum number of acquisition attempts (0 = try once, no retries)
//
// Returns:
//   - FileLockOption: Configuration function for NewFileLock
//
// Recommended values:
//   - Critical operations: 10-50 retries
//   - Best-effort operations: 1-5 retries
//   - Use WithNoMaxRetry() for indefinite attempts
func WithMaxRetry(retry int) FileLockOption {
	return func(fl *FileLock) {
		fl.maxRetry = retry
	}
}

// WithNoMaxRetry configures the lock to retry indefinitely until acquisition succeeds or context is cancelled.
//
// This option sets the maximum retry count to effectively infinite, meaning the lock
// acquisition will continue retrying until either:
//   1. The lock is successfully acquired
//   2. The context is cancelled or times out
//   3. An unrecoverable error occurs
//
// Returns:
//   - FileLockOption: Configuration function for NewFileLock
//
// Use cases:
//   - Critical system processes that must eventually acquire the lock
//   - Services that can wait indefinitely for resource access
//   - Operations where failure is not acceptable
//
// Warning:
//   Always use with a context timeout to prevent indefinite blocking
func WithNoMaxRetry() FileLockOption {
	return func(fl *FileLock) {
		fl.maxRetry = math.MaxInt
	}
}

// lockContent represents the JSON structure stored in lock files for ownership verification.
//
// This structure is serialized to JSON and stored in the lock file to provide:
//   - Process identification for ownership verification
//   - Timestamp information for stale lock detection
//   - Host identification for distributed system coordination
//
// The JSON format enables easy inspection and debugging of lock files.
type lockContent struct {
	Hostname  string    `json:"hostname"`  // Hostname of the lock owner
	PID       int       `json:"pid"`       // Process ID of the lock owner
	Timestamp time.Time `json:"timestamp"` // Last update timestamp
}

// NewFileLock creates a new file-based distributed lock with the specified configuration.
//
// This constructor initializes a FileLock with default settings and applies any provided
// options. The lock is ready for use but not yet acquired - call Acquire() to obtain it.
//
// Default configuration:
//   - Stale timeout: 500ms (locks older than this are considered abandoned)
//   - Refresh interval: 200ms (how often to update lock timestamp)
//   - Retry interval: 1s (wait time between acquisition attempts)
//   - Max retries: 5 (maximum acquisition attempts before giving up)
//
// Parameters:
//   - ctx: Context for cancellation and timeout control
//   - filepath: Absolute path where the lock file will be created
//   - opts: Optional configuration functions to customize behavior
//
// Returns:
//   - *FileLock: Configured lock instance ready for acquisition
//
// Example:
//   ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//   defer cancel()
//   
//   lock := NewFileLock(ctx, "/tmp/myapp.lock",
//       WithStaleTimeout(10*time.Second),
//       WithMaxRetry(3))
//   
//   if err := lock.Acquire(); err != nil {
//       log.Fatal("Cannot acquire lock:", err)
//   }
//   defer lock.Release()
func NewFileLock(ctx context.Context, filepath string, opts ...FileLockOption) *FileLock {
	hostname, _ := os.Hostname()
	pid := os.Getpid()

	// create with defaults
	fl := &FileLock{
		filepath:        filepath,
		staleTimeout:    DefaultStaleTimeout,
		refreshInterval: DefaultRefreshInterval,
		retryInterval:   DefaultRetryInterval,
		maxRetry:        DefaultMaxRetry,
		hostname:        hostname,
		pid:             pid,
		ctx:             ctx,
	}

	// apply the options
	for _, opt := range opts {
		opt(fl)
	}

	return fl
}

// Acquire attempts to obtain the file lock, retrying according to configuration.
//
// This method implements the core lock acquisition logic with the following behavior:
//   1. Attempts atomic lock creation using O_EXCL flag
//   2. If lock exists, checks if it's stale (timestamp too old)
//   3. Removes stale locks and retries acquisition
//   4. Respects context cancellation and retry limits
//   5. Starts background refresh goroutine on successful acquisition
//
// Lock acquisition is atomic and race-free through filesystem O_EXCL semantics.
// The method will retry until successful, cancelled, or max retries exceeded.
//
// Returns:
//   - nil: Lock successfully acquired and background refresh started
//   - ErrMaxRetryExceeded: Exceeded maximum retry attempts
//   - ErrLockContextCancelled: Context was cancelled during acquisition
//   - ErrLockAcquire: Other acquisition failures (permissions, I/O errors)
//
// Behavior after successful acquisition:
//   - Background goroutine starts refreshing lock timestamp
//   - Lock file contains hostname, PID, and current timestamp
//   - Lock is held until Release() is called or process terminates
//
// Example:
//   if err := lock.Acquire(); err != nil {
//       if errors.Is(err, ErrMaxRetryExceeded) {
//           log.Printf("Could not acquire lock after retries")
//       }
//       return err
//   }
//   defer lock.Release()
func (fl *FileLock) Acquire() error {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	// ensure the directory exists
	_, err := os.ReadDir(filepath.Dir(fl.filepath))
	if os.IsNotExist(err) {
		return err
	}

	// track retry count
	retry := 0

	for {
		select {
		case <-fl.ctx.Done():
			return fmt.Errorf("%w: context cancelled", ErrLockContextCancelled)
		default:
			// break if max retry is met
			if retry > fl.maxRetry {
				return ErrMaxRetryExceeded
			}

			// create lock file atomically
			file, err := os.OpenFile(fl.filepath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, lockFilePermissions)
			if err == nil {
				// acquired the lock

				// write to the lock file
				if err2 := fl.writeLock(file); err2 != nil {
					file.Close()
					os.Remove(fl.filepath)
					return err2
				}
				file.Close()

				// start background refresh
				ctx, cancel := context.WithCancel(fl.ctx)
				fl.cancel = cancel
				fl.wg.Add(1)
				go func() {
					defer fl.wg.Done()
					fl.refreshLock(ctx)
				}()
				return nil
			}

			// check the existing lock file
			current, err := fl.readLockContent()
			if err != nil {
				// lock was removed, retry
				if os.IsNotExist(err) {
					continue
				}

				// count corrupt files as valid, so wait for lock to expire
				if strings.Contains(err.Error(), ErrLockCorrup.Error()) {
					// lock file valid
					retry += 1
					time.Sleep(fl.retryInterval)
					continue
				}

				// unknown issue getting the lock file
				return fmt.Errorf("%w: %v", ErrLockAcquire, err)
			}

			// check validity of the local lock file
			if time.Since(current.Timestamp) < fl.staleTimeout {
				// lock file valid
				retry += 1
				time.Sleep(fl.retryInterval)
				continue
			}

			// stale lock file, remove and retry
			if err := os.Remove(fl.filepath); err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return fmt.Errorf("%w: failed to remove stale lock: %v", ErrLockAcquire, err)
			}
		}
	}
}

// Release safely releases the file lock and cleans up all associated resources.
//
// This method performs complete cleanup of the lock:
//   1. Cancels the background refresh goroutine
//   2. Waits for the refresh goroutine to terminate
//   3. Removes the lock file from the filesystem
//   4. Resets internal state for potential reuse
//
// The method is safe to call multiple times and will not error if the lock
// is already released or the file doesn't exist.
//
// Returns:
//   - nil: Lock successfully released and cleaned up
//   - error: File system error during lock file removal
//
// Thread safety:
//   This method is thread-safe and can be called from multiple goroutines.
//   It uses internal synchronization to ensure clean shutdown.
//
// Usage:
//   defer lock.Release()  // Typical usage with defer
//   
//   // Or explicit release
//   if err := lock.Release(); err != nil {
//       log.Printf("Warning: failed to release lock: %v", err)
//   }
func (fl *FileLock) Release() error {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	// propagate the cancel across context
	if fl.cancel != nil {
		fl.cancel()
		fl.cancel = nil
		// Wait for the background goroutine to complete before proceeding
		fl.wg.Wait()
	}

	return fl.releaseFile()
}

// releaseFile removes the lock file from the filesystem without waiting for background goroutines.
//
// This internal method handles the actual file removal and is used both by Release()
// and by the background refresh goroutine when it detects lock loss. It's safe to call
// multiple times and ignores "file not found" errors.
//
// Returns:
//   - nil: Lock file successfully removed or didn't exist
//   - error: File system error during removal (permissions, I/O failure)
func (fl *FileLock) releaseFile() error {
	// remove the file
	if err := os.Remove(fl.filepath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to release lock: %w", err)
	}
	return nil
}

// refreshLock runs in a background goroutine to maintain lock ownership by updating timestamps.
//
// This method implements the lock keepalive mechanism:
//   1. Periodically updates the lock file timestamp at refresh intervals
//   2. Verifies continued ownership before each update
//   3. Automatically releases the lock if ownership is lost
//   4. Terminates when context is cancelled (during Release())
//
// The refresh process prevents the lock from being considered stale by other processes
// and ensures that crashed processes don't hold locks indefinitely.
//
// Parameters:
//   - ctx: Context for controlling the refresh loop (cancelled during Release())
//
// Behavior:
//   - Updates lock timestamp every refresh interval
//   - Releases lock if update fails (indicating ownership loss)
//   - Gracefully terminates when context is cancelled
func (fl *FileLock) refreshLock(ctx context.Context) {
	ticker := time.NewTicker(fl.refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := fl.updateLock(); err != nil {
				// if failing to update the lock, release it so we do not lock here
				_ = fl.releaseFile()
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

// updateLock atomically updates the lock file timestamp to maintain ownership.
//
// This method performs an atomic update of the lock file:
//   1. Creates a temporary file with new timestamp
//   2. Verifies current lock still belongs to this process
//   3. Atomically replaces the lock file with the temporary file
//   4. Cleans up temporary file regardless of success/failure
//
// The atomic replacement using rename() ensures that other processes never see
// a partially updated or missing lock file during the update process.
//
// Returns:
//   - nil: Lock timestamp successfully updated
//   - ErrLockLost: Lock no longer belongs to this process
//   - error: File system error during update operation
//
// This method is called periodically by the background refresh goroutine.
func (fl *FileLock) updateLock() error {
	// use file renames to give atomic operations
	tempFile, err := os.CreateTemp(filepath.Dir(fl.filepath), "lock-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())

	// write to the temporary file
	if err = fl.writeLock(tempFile); err != nil {
		return fmt.Errorf("failed to write temp lock: %w", err)
	}
	tempFile.Close()

	// ensure the lock belongs to this process
	current, err := fl.readLockContent()
	if err != nil {
		return ErrLockLost
	}
	if current.Hostname != fl.hostname || current.PID != fl.pid {
		return ErrLockLost
	}

	// replace the current lock file data atomically
	if err := os.Rename(tempFile.Name(), fl.filepath); err != nil {
		return fmt.Errorf("failed to atomically update lock: %w", err)
	}

	return nil
}

// readLockContent reads and parses the JSON content from the lock file.
//
// This method safely reads the lock file and deserializes its JSON content
// to extract ownership and timestamp information. It handles file I/O errors
// and JSON parsing errors appropriately.
//
// Returns:
//   - *lockContent: Parsed lock information (hostname, PID, timestamp)
//   - error: File read error or JSON parsing error (wrapped as ErrLockCorrup)
//
// The returned content is used for:
//   - Ownership verification during updates
//   - Stale lock detection during acquisition
//   - Debugging and monitoring lock state
func (fl *FileLock) readLockContent() (*lockContent, error) {
	data, err := os.ReadFile(fl.filepath)
	if err != nil {
		return nil, err
	}

	var lc lockContent
	if err := json.Unmarshal(data, &lc); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrLockCorrup, err)
	}
	return &lc, nil
}

// writeLock writes the current process information as JSON to the provided file.
//
// This method creates the JSON content for the lock file containing:
//   - Current hostname for distributed system identification
//   - Current process ID for local process identification
//   - Current timestamp for stale detection
//
// The method ensures data is fully written and synced to disk before returning
// to guarantee that other processes can immediately see the lock.
//
// Parameters:
//   - f: Open file handle to write the lock content to
//
// Returns:
//   - nil: Lock content successfully written and synced
//   - error: JSON encoding error, write error, or sync error
//
// The file must be opened for writing and will be written to at its current position.
func (fl *FileLock) writeLock(f *os.File) error {
	data, err := json.Marshal(lockContent{
		Hostname:  fl.hostname,
		PID:       fl.pid,
		Timestamp: time.Now(),
	})
	if err != nil {
		return fmt.Errorf("failed to encode lock content to json: %w", err)
	}

	// write and sync data
	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("failed to write lock file: %w", err)
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("failed to sync lock file: %w", err)
	}

	return nil
}
