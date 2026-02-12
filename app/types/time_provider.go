// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package types defines time abstraction interfaces for testable time-dependent operations.
package types

import "time"

// TimeProvider abstracts time operations to enable deterministic testing of time-sensitive logic.
// This interface allows components to use real time in production while enabling controlled
// time manipulation in tests for accurate validation of time-based behaviors.
type TimeProvider interface {
	// GetCurrentTime returns the current system time.
	// In production, this returns time.Now(). In tests, this can return controlled timestamps
	// to validate time-dependent logic such as metric timestamping and data retention.
	GetCurrentTime() time.Time
}
