// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package utils provides utility functions and types for CloudZero Agent operational support.
// This package implements reusable utility components that support CloudZero Agent operations
// across multiple architectural layers, providing consistent patterns for common tasks
// including time management, data processing, and operational utilities.
//
// The utilities maintain clean interfaces and testable implementations, enabling their use
// across Primary Adapters, Application Core, and Secondary Adapters while providing
// consistent behavior and operational reliability.
//
// Key capabilities:
//   - Time management: Clock abstraction with UTC standardization for consistent timestamps
//   - Data processing: Chunk processing utilities for batching and streaming operations
//   - Format utilities: Standardized formatting for storage and transmission
//   - Testing support: Mock implementations and test utilities for reliable testing
//
// Integration patterns:
//   - Domain services: Time providers for business logic operations
//   - Storage operations: Timestamp formatting for database persistence
//   - Data processing: Chunking utilities for efficient batch operations
//   - Testing infrastructure: Mock utilities for deterministic testing
//
// The package follows dependency injection patterns, providing interfaces that enable
// testing and operational flexibility while maintaining consistent behavior across
// CloudZero Agent components.
package utils

import (
	"time"

	"github.com/cloudzero/cloudzero-agent/app/types"
)

// Compile-time verification that Clock implements the TimeProvider interface.
// This ensures that the Clock struct satisfies all TimeProvider contract requirements,
// enabling its use throughout CloudZero Agent for consistent time operations.
var _ types.TimeProvider = (*Clock)(nil)

// Clock provides production time operations for CloudZero Agent components.
// This struct implements the TimeProvider interface with real system clock integration,
// providing UTC-standardized timestamps for consistent time handling across all agent operations.
//
// Design characteristics:
//   - UTC normalization: All timestamps converted to UTC for consistent storage and processing
//   - Production reliability: Uses system clock for accurate time representation
//   - Interface compliance: Implements types.TimeProvider for dependency injection
//   - Zero configuration: No setup required, ready for immediate use
//
// Usage patterns:
//   - Domain services: Inject Clock for business logic time requirements
//   - Storage operations: Generate timestamps for database records
//   - Metric processing: Timestamp metric data for temporal analysis
//   - Audit logging: Record operation times for compliance and debugging
//
// The Clock provides a simple abstraction over system time that enables testing
// through interface substitution while providing reliable time operations in production.
type Clock struct{}

// GetCurrentTime returns the current system time normalized to UTC timezone.
// This method implements the TimeProvider interface requirement, providing consistent
// time representation across all CloudZero Agent operations regardless of system timezone.
//
// UTC normalization ensures:
//   - Consistent timestamps across different deployment environments
//   - Reliable time comparisons and calculations
//   - Standard format for storage and transmission
//   - Compatibility with CloudZero platform expectations
//
// This method is called frequently throughout CloudZero Agent operations for:
//   - Database record timestamps
//   - Metric data time attribution
//   - Audit log entries
//   - Business logic time calculations
//
// Returns the current system time converted to UTC timezone.
func (c *Clock) GetCurrentTime() time.Time {
	return time.Now().UTC()
}

// FormatForStorage converts time values to standardized string format for database storage and transmission.
// This function provides consistent time formatting across all CloudZero Agent storage operations,
// ensuring reliable time representation and parsing for operational data persistence.
//
// Format specification:
//   - Pattern: "2006-01-02 15:04:05.999999999 -0700 MST"
//   - Precision: Nanosecond-level accuracy for high-precision time requirements
//   - Timezone: Includes timezone information for accurate time reconstruction
//   - Compatibility: Parseable format for database systems and CloudZero platform
//
// Usage contexts:
//   - Database persistence: Timestamp formatting for storage operations
//   - API transmission: Standardized time format for CloudZero platform communication
//   - Audit logging: Consistent time representation for operational records
//   - Metric attribution: Time formatting for temporal data analysis
//
// The format preserves full time precision and timezone information while providing
// human-readable representation suitable for both machine processing and debugging.
//
// Returns the formatted time string suitable for storage and transmission.
func FormatForStorage(t time.Time) string {
	return t.Format("2006-01-02 15:04:05.999999999 -0700 MST")
}
