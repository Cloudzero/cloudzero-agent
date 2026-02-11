// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"encoding/json"
)

// Set provides a generic set data structure for efficient membership testing and deduplication.
// Used throughout the CloudZero Agent for managing collections of unique metric names,
// file paths, and configuration keys where order doesn't matter but uniqueness is required.
type Set[T comparable] struct {
	// elements stores the set members using a map for O(1) lookup performance.
	// The empty struct{} value minimizes memory usage compared to bool or interface{}.
	elements map[T]struct{}
}

// NewSet creates and returns a new empty Set with the specified comparable type.
// The zero value is ready to use but this constructor ensures proper initialization.
func NewSet[T comparable]() *Set[T] {
	return &Set[T]{
		elements: make(map[T]struct{}),
	}
}

// NewSetFromList creates a set from a slice of elements, automatically deduplicating entries.
// Useful for converting configuration arrays or metric name lists into sets for efficient processing.
func NewSetFromList[T comparable](list []T) *Set[T] {
	elements := make(map[T]struct{})
	for _, item := range list {
		elements[item] = struct{}{}
	}
	return &Set[T]{
		elements: elements,
	}
}

// Add inserts an element into the set, idempotent operation that has no effect if element already exists.
// Used for building sets of unique metric names during filtering operations.
func (s *Set[T]) Add(element T) {
	s.elements[element] = struct{}{}
}

// Remove deletes an element from the set, no effect if element doesn't exist.
// Used for excluding specific metrics or configuration keys from processing.
func (s *Set[T]) Remove(element T) {
	delete(s.elements, element)
}

// Contains performs O(1) membership testing to check if an element exists in the set.
// Primary method for metric name filtering and configuration key validation.
func (s *Set[T]) Contains(element T) bool {
	_, exists := s.elements[element]
	return exists
}

// Size returns the current number of unique elements in the set.
// Used for metrics reporting and capacity planning in metric processing pipelines.
func (s *Set[T]) Size() int {
	return len(s.elements)
}

// List converts the set to a slice for iteration or serialization purposes.
// Order is not guaranteed due to the underlying map implementation.
// Used when converting sets back to arrays for configuration output.
func (s *Set[T]) List() []T {
	keys := make([]T, 0, len(s.elements))
	for key := range s.elements {
		keys = append(keys, key)
	}
	return keys
}

// Diff returns a new set containing elements present in this set but absent in the other set.
// Used for configuration comparison and incremental metric processing to identify new entries.
func (s *Set[T]) Diff(other *Set[T]) *Set[T] {
	result := NewSet[T]()

	for item := range s.elements {
		if !other.Contains(item) {
			result.Add(item)
		}
	}

	return result
}

// MarshalJSON implements the json.Marshaler interface by converting the set to a JSON array.
// The serialization format matches standard JSON arrays for compatibility with configuration files.
func (s *Set[T]) MarshalJSON() ([]byte, error) {
	// Convert the set to a slice and encode it as JSON
	return json.Marshal(s.List())
}

// UnmarshalJSON implements the json.Unmarshaler interface by reading a JSON array into the set.
// Automatically deduplicates any duplicate elements in the input JSON array.
func (s *Set[T]) UnmarshalJSON(data []byte) error {
	// Decode the JSON array into a slice
	var elements []T
	if err := json.Unmarshal(data, &elements); err != nil {
		return err
	}

	// Clear the current set and populate it with the decoded elements
	s.elements = make(map[T]struct{})
	for _, element := range elements {
		s.Add(element)
	}

	return nil
}
