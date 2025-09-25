// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package logging provides structured logging infrastructure for CloudZero Agent operations.
package logging

import (
	"encoding/json"
	"io"
	"sync"
)

// fieldFilterWriter provides selective field filtering for CloudZero Agent log output.
// This writer removes specified fields from JSON log events before writing to the underlying
// destination, enabling sensitive data protection and output customization for different environments.
//
// Field filtering applications:
//   - Security: Remove sensitive information like authentication tokens, API keys
//   - Privacy: Filter personally identifiable information from log outputs
//   - Compliance: Ensure regulatory compliance by removing restricted data fields
//   - Testing: Clean log output by removing noise fields like trace IDs
//   - Performance: Reduce log volume by removing verbose debugging fields
//
// The writer maintains thread safety through mutex protection and preserves the original
// write size contract while filtering the actual output content.
type fieldFilterWriter struct {
	// w is the underlying writer that receives filtered log content.
	// This writer receives JSON log events with specified fields removed,
	// enabling clean output to console, files, or external logging systems.
	w io.Writer

	// fieldsToSkip contains field names to remove from JSON log events.
	// Using a map provides O(1) lookup performance for efficient filtering
	// of log events containing many fields.
	fieldsToSkip map[string]struct{}

	// mu protects concurrent access to the writer during multi-threaded logging.
	// This mutex ensures thread-safe field filtering when multiple goroutines
	// write log events simultaneously.
	mu sync.Mutex
}

// NewFieldFilterWriter creates a JSON log field filtering writer for CloudZero Agent logging.
// This constructor creates a writer that removes specified fields from JSON log events,
// enabling selective field filtering for security, privacy, and output customization.
//
// Field filtering use cases:
//   - Security filtering: Remove authentication tokens, API keys, passwords
//   - Privacy protection: Filter user IDs, email addresses, personal information
//   - Trace cleanup: Remove tracing fields (spanId, parentSpanId) for cleaner output
//   - Environment filtering: Remove development-specific fields in production
//   - Volume reduction: Filter verbose fields to reduce log storage requirements
//
// Parameters:
//   - w: Underlying writer for filtered output (stdout, files, external systems)
//   - fieldsToSkip: Field names to remove from JSON log events
//
// Filtering behavior:
//   - Only affects JSON log events; non-JSON content passes through unchanged
//   - Preserves all other fields and log structure
//   - Maintains newline formatting for consistent output
//   - Thread-safe for concurrent logging operations
//
// The returned writer implements io.Writer and can be used anywhere
// standard writers are accepted in the CloudZero Agent logging pipeline.
func NewFieldFilterWriter(w io.Writer, fieldsToSkip []string) io.Writer {
	skipMap := make(map[string]struct{}, len(fieldsToSkip))
	for _, field := range fieldsToSkip {
		skipMap[field] = struct{}{}
	}
	return &fieldFilterWriter{
		w:            w,
		fieldsToSkip: skipMap,
	}
}

// Write implements io.Writer interface with JSON field filtering for CloudZero Agent log events.
// This method processes log content by parsing JSON events, removing specified fields,
// and writing the filtered content to the underlying writer while maintaining io.Writer contracts.
//
// Filtering process:
//  1. Parse incoming bytes as JSON log event
//  2. Remove all fields specified in the fieldsToSkip map
//  3. Re-marshal the filtered event back to JSON
//  4. Preserve original formatting including newlines
//  5. Write filtered content to underlying writer
//
// Error handling:
//   - Non-JSON content: Passed through unmodified to underlying writer
//   - JSON parse errors: Original content written without filtering
//   - Marshal errors: Original content written as fallback
//   - Write errors: Propagated from underlying writer
//
// Writer contract compliance:
//   - Returns original byte count to satisfy io.Writer interface
//   - Reports underlying writer errors appropriately
//   - Handles short writes with proper error indication
//   - Thread-safe through mutex protection
//
// This implementation ensures that CloudZero Agent logging continues to function
// even when field filtering encounters errors, prioritizing operational reliability
// over perfect field filtering.
func (f *fieldFilterWriter) Write(p []byte) (n int, err error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Return original length regardless of filtered length
	originalLen := len(p)

	// Handle empty write
	if originalLen == 0 {
		return 0, nil
	}

	// Parse the JSON
	var logEntry map[string]interface{}
	if err = json.Unmarshal(p, &logEntry); err != nil {
		// If we can't parse it, write it as-is
		written, writeErr := f.w.Write(p)
		if writeErr != nil {
			return written, writeErr
		}
		if written < originalLen {
			return written, io.ErrShortWrite
		}
		return originalLen, nil
	}

	// Remove the fields we want to skip
	for field := range f.fieldsToSkip {
		delete(logEntry, field)
	}

	// Marshal it back to JSON
	filtered, err := json.Marshal(logEntry)
	if err != nil {
		// If we can't remarshal, write the original
		written, writeErr := f.w.Write(p)
		if writeErr != nil {
			return written, writeErr
		}
		if written < originalLen {
			return written, io.ErrShortWrite
		}
		return originalLen, nil
	}

	// Add a newline if the original ended with one
	if p[len(p)-1] == '\n' {
		filtered = append(filtered, '\n')
	}

	// Write the filtered content
	written, writeErr := f.w.Write(filtered)
	if writeErr != nil {
		return written, writeErr
	}

	// Even if the filtered content was fully written,
	// report the original size to satisfy Writer contract
	return originalLen, nil
}
