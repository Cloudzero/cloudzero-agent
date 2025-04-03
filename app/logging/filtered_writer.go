// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package logging

import (
	"encoding/json"
	"io"
	"sync"
)

type fieldFilterWriter struct {
	w            io.Writer
	fieldsToSkip map[string]struct{}
	mu           sync.Mutex
}

// NewFieldFilterWriter creates a new writer that filters out specified fields
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

// Write implements io.Writer and filters out specified fields from the JSON
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
