package lock

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	ErrLockExists          = errors.New("lock already exists")
	ErrLockStale           = errors.New("stale lock detected")
	ErrLockLost            = errors.New("lock lost")
	ErrLockAcquire         = errors.New("failed to acquire lock")
	ErrLockCorrup          = errors.New("corrupt lock file")
	DefautlStaleTimeout    = time.Millisecond * 500
	DefaultRefreshInterval = time.Millisecond * 200
	DefaultRetryInterval   = 1 * time.Second
)

type FileLock struct {
	path            string
	staleTimeout    time.Duration
	refreshInterval time.Duration
	retryInterval   time.Duration

	hostname string
	pid      int
	ctx      context.Context
	cancel   context.CancelFunc
	mu       sync.Mutex
}

type FileLockOption func(fl *FileLock)

func WithStaleTimeout(timeout time.Duration) FileLockOption {
	return func(fl *FileLock) {
		fl.staleTimeout = timeout
	}
}

func WithRetryInterval(interval time.Duration) FileLockOption {
	return func(fl *FileLock) {
		fl.retryInterval = interval
	}
}

func WithRefreshInterval(interval time.Duration) FileLockOption {
	return func(fl *FileLock) {
		fl.refreshInterval = interval
	}
}

type lockContent struct {
	Hostname  string    `json:"hostname"`
	PID       int       `json:"pid"`
	Timestamp time.Time `json:"timestamp"`
}

func NewFileLock(ctx context.Context, path string, opts ...FileLockOption) *FileLock {
	hostname, _ := os.Hostname()
	pid := os.Getpid()

	// create with defaults
	fl := &FileLock{
		path:            path,
		staleTimeout:    DefautlStaleTimeout,
		refreshInterval: DefaultRefreshInterval,
		retryInterval:   DefaultRetryInterval,
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

func (fl *FileLock) Acquire() error {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	for {
		// create lock file atomically
		file, err := os.OpenFile(fl.path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err == nil {
			// aquired the lock

			// write to the lock file
			if err := fl.writeLock(file); err != nil {
				file.Close()
				os.Remove(fl.path)
				return err
			}
			file.Close()

			// start background refresh
			ctx, cancel := context.WithCancel(fl.ctx)
			fl.cancel = cancel
			go fl.refreshLock(ctx)
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
				time.Sleep(fl.retryInterval)
				continue
			}

			// unknown issue getting the lock file
			return fmt.Errorf("%w: %v", ErrLockAcquire, err)
		}

		// check validity of the local lock file
		if time.Since(current.Timestamp) < fl.staleTimeout {
			time.Sleep(fl.retryInterval)
			continue
		}

		// stale lock file, remove and retry
		if err := os.Remove(fl.path); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("%w: failed to remove stale lock: %v", ErrLockAcquire, err)
		}
	}
}

func (fl *FileLock) Release() error {
	fl.mu.Lock()
	defer fl.mu.Unlock()

	// propigate the cancel across context
	if fl.cancel != nil {
		fl.cancel()
		fl.cancel = nil
	}

	// remove the file
	if err := os.Remove(fl.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to release lock: %w", err)
	}
	return nil
}

func (fl *FileLock) refreshLock(ctx context.Context) {
	ticker := time.NewTicker(fl.refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := fl.updateLock(); err != nil {
				// if failing to update the lock, release it so we do not lock here
				fl.Release()
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func (fl *FileLock) updateLock() error {
	// use file renames to give atomic operations
	tempFile, err := os.CreateTemp(filepath.Dir(fl.path), "lock-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name())

	// write to the temporary file
	if err := fl.writeLock(tempFile); err != nil {
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
	if err := os.Rename(tempFile.Name(), fl.path); err != nil {
		return fmt.Errorf("failed to atomically update lock: %w", err)
	}

	return nil
}

func (fl *FileLock) readLockContent() (*lockContent, error) {
	data, err := os.ReadFile(fl.path)
	if err != nil {
		return nil, err
	}

	var lc lockContent
	if err := json.Unmarshal(data, &lc); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrLockCorrup, err)
	}
	return &lc, nil
}

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
