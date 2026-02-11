// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package utils provides utility functions and types for CloudZero Agent operational support.
package utils

// Chunk divides a slice into smaller sub-slices of specified size for efficient batch processing.
// This generic utility function enables CloudZero Agent to process large datasets in manageable
// chunks, optimizing memory usage and enabling controlled processing rates.
//
// Batch processing applications:
//   - Metric data processing: Split large metric batches for efficient storage operations
//   - API requests: Batch CloudZero platform uploads to respect rate limits
//   - Database operations: Process large record sets in manageable transactions
//   - Memory management: Control memory usage during large data processing
//
// Parameters:
//   - list: Source slice of any type T to be divided into chunks
//   - n: Maximum size of each chunk (must be positive)
//
// Behavior characteristics:
//   - Zero/negative chunk size: Returns original slice wrapped in single chunk
//   - Uneven division: Final chunk contains remaining elements (may be smaller than n)
//   - Empty input: Returns empty slice of chunks
//   - Preserves element order: Maintains original ordering within and across chunks
//
// Performance considerations:
//   - Uses slice operations for efficient memory allocation
//   - Minimizes memory copying through slice references
//   - Scales linearly with input size
//   - Suitable for processing large datasets without memory exhaustion
//
// Returns a slice of slices, where each inner slice contains at most n elements
// from the original input, enabling controlled batch processing operations.
func Chunk[T any](list []T, n int) [][]T {
	if n <= 0 {
		return [][]T{list}
	}

	var chunks [][]T
	for i := 0; i < len(list); i += n {
		end := min(i+n, len(list))
		chunks = append(chunks, list[i:end])
	}

	return chunks
}
