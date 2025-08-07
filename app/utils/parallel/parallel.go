// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package parallel provides utilities for concurrent task execution with controlled parallelism.
//
// This package implements a semaphore-based worker pool pattern that allows controlled
// parallel execution of tasks while preventing resource exhaustion. It's specifically
// designed for scenarios where you need to process many tasks concurrently but want
// to limit the number of simultaneous operations.
//
// Key features:
//   - Semaphore-based concurrency control to prevent resource exhaustion
//   - Automatic worker count scaling based on CPU cores
//   - Error aggregation across all parallel tasks
//   - Graceful shutdown and cleanup of worker goroutines
//   - Thread-safe operations with proper synchronization
//
// Architecture:
//   - Manager: Controls the parallel execution with configurable worker limits
//   - Task: Function type that can return errors for proper error handling
//   - Waiter: Aggregates results and errors from all parallel tasks
//   - Semaphore: Limits the number of concurrent goroutines
//
// Usage patterns:
//   1. Create Manager with desired worker count
//   2. Create Waiter for result aggregation
//   3. Submit tasks to Manager with Run()
//   4. Wait() blocks until all tasks complete
//   5. Check Err() channel for any task failures
//
// This is particularly useful in the CloudZero agent for:
//   - Parallel file uploads to cloud storage
//   - Concurrent metric processing
//   - Simultaneous API requests with rate limiting
//   - Batch operations that need controlled parallelism
package parallel

import (
	"runtime"
	"sync"
)

const (
	minNumWorkers    = 2
	errChannelBuffer = 100
)

// Task is a function type for parallel manager.
type Task func() error

// Manager is a structure for running tasks in parallel.
type Manager struct {
	wg        *sync.WaitGroup
	semaphore chan struct{}
}

// New creates a new parallel.Manager.
func New(workercount int) *Manager {
	if workercount < 0 {
		workercount = runtime.NumCPU() * -workercount
	}

	if workercount < minNumWorkers {
		workercount = minNumWorkers
	}

	return &Manager{
		wg:        &sync.WaitGroup{},
		semaphore: make(chan struct{}, workercount),
	}
}

// acquire limits concurrency by trying to acquire the semaphore.
func (p *Manager) acquire() {
	p.semaphore <- struct{}{}
	p.wg.Add(1)
}

// release releases the acquired semaphore to signal that a task is finished.
func (p *Manager) release() {
	p.wg.Done()
	<-p.semaphore
}

// Run runs the given task while limiting the concurrency.
func (p *Manager) Run(fn Task, waiter *Waiter) {
	waiter.wg.Add(1)
	p.acquire()
	go func() {
		defer waiter.wg.Done()
		defer p.release()

		if err := fn(); err != nil {
			waiter.errch <- err
		}
	}()
}

// Close waits all tasks to finish.
func (p *Manager) Close() {
	p.wg.Wait()
	close(p.semaphore)
}

// Waiter is a structure for waiting and reading
// error messages created by Manager.
type Waiter struct {
	wg    sync.WaitGroup
	errch chan error
}

// NewWaiter creates a new parallel.Waiter.
func NewWaiter() *Waiter {
	return &Waiter{
		errch: make(chan error, errChannelBuffer),
	}
}

// Wait blocks until the WaitGroup counter is zero
// and closes error channel.
func (w *Waiter) Wait() {
	w.wg.Wait()
	close(w.errch)
}

// Err returns read-only error channel.
func (w *Waiter) Err() <-chan error {
	return w.errch
}
