// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package mocks

import (
	"sync"
	"time"
)

// MockClock is a mock implementation of the TimeProvider interface.
// It allows setting and retrieving the current time manually.
type MockClock struct {
	mu          sync.RWMutex
	currentTime time.Time
}

// NewMockClock initializes a new MockClock with the specified initial time.
func NewMockClock(initialTime time.Time) *MockClock {
	return &MockClock{
		currentTime: initialTime,
	}
}

// GetCurrentTime returns the current mock time.
func (mc *MockClock) GetCurrentTime() time.Time {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.currentTime
}

// SetCurrentTime sets the current mock time to the specified time.
func (mc *MockClock) SetCurrentTime(t time.Time) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.currentTime = t
}

// AdvanceTime advances the current mock time by the specified duration.
func (mc *MockClock) AdvanceTime(d time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.currentTime = mc.currentTime.Add(d)
}
