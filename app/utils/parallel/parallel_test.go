// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package parallel_test

import (
	"errors"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/cloudzero/cloudzero-agent/app/utils/parallel"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		workerCount int
		expected    int
	}{
		{"NegativeWorkerCount", -1, runtime.NumCPU()},
		{"ZeroWorkerCount", 0, 2},
		{"PositiveWorkerCount", 5, 5},
		{"LessThanMinWorkers", 1, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := parallel.New(tt.workerCount)
			assert.NotNil(t, manager)
		})
	}
}

func TestManagerHandlesError(t *testing.T) {
	pm := parallel.New(2)
	defer pm.Close()
	waiter := parallel.NewWaiter()

	sampleErr := errors.New("sample error")
	pm.Run(func() error {
		return sampleErr
	}, waiter)
	waiter.Wait()

	var gotErr error
	for err := range waiter.Err() {
		gotErr = err
	}
	if gotErr == nil {
		t.Fatalf("expected error but got nil")
	}
	if gotErr.Error() != sampleErr.Error() {
		t.Fatalf("expected error %q but got %q", sampleErr, gotErr)
	}
}

func TestManagerNoHangOnSuccess(t *testing.T) {
	pm := parallel.New(2)
	defer pm.Close()
	waiter := parallel.NewWaiter()

	done := make(chan struct{})
	pm.Run(func() error {
		time.Sleep(10 * time.Millisecond)
		return nil
	}, waiter)
	go func() {
		waiter.Wait()
		close(done)
	}()

	select {
	case <-done:
		// completed successfully
	case <-time.After(100 * time.Millisecond):
		t.Fatal("execution hung on successful task")
	}
}

func TestManagerMultipleTasks(t *testing.T) {
	const taskCount = 50
	pm := parallel.New(5)
	defer pm.Close()
	waiter := parallel.NewWaiter()

	var successCount uint64
	for i := 0; i < taskCount; i++ {
		idx := i
		pm.Run(func() error {
			if idx%10 == 0 {
				return errors.New("error task")
			}
			atomic.AddUint64(&successCount, 1)
			return nil
		}, waiter)
	}
	waiter.Wait()

	errCount := 0
	for err := range waiter.Err() {
		if err != nil {
			errCount++
		}
	}

	if errCount != taskCount/10 {
		t.Fatalf("expected %d errors but got %d", taskCount/10, errCount)
	}
	if successCount != uint64(taskCount-taskCount/10) {
		t.Fatalf("expected %d successes but got %d", taskCount-taskCount/10, successCount)
	}
}
