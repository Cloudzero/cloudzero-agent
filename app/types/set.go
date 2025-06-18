// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"encoding/json"
)

// Set is a convenience type around a `map[string]struct{}`
type Set[T comparable] struct {
	elements map[T]struct{}
}

// NewSet creates and returns a new Set
func NewSet[T comparable]() *Set[T] {
	return &Set[T]{
		elements: make(map[T]struct{}),
	}
}

// NewSetFromList creates a set from a list of elements
func NewSetFromList[T comparable](list []T) *Set[T] {
	elements := make(map[T]struct{})
	for _, item := range list {
		elements[item] = struct{}{}
	}
	return &Set[T]{
		elements: elements,
	}
}

// Add adds an element to the set
func (s *Set[T]) Add(element T) {
	s.elements[element] = struct{}{}
}

// Remove removes an element from the set
func (s *Set[T]) Remove(element T) {
	delete(s.elements, element)
}

// Contains checks if an element is in the set
func (s *Set[T]) Contains(element T) bool {
	_, exists := s.elements[element]
	return exists
}

// Size returns the number of elements in the set
func (s *Set[T]) Size() int {
	return len(s.elements)
}

// List returns all elements in the set as a slice
func (s *Set[T]) List() []T {
	keys := make([]T, 0, len(s.elements))
	for key := range s.elements {
		keys = append(keys, key)
	}
	return keys
}

// Diff returns a new set containing elements that exist in the source set but not in `b`
func (s *Set[T]) Diff(other *Set[T]) *Set[T] {
	result := NewSet[T]()

	for item := range s.elements {
		if !other.Contains(item) {
			result.Add(item)
		}
	}

	return result
}

// MarshalJSON implements the json.Marshaler interface
func (s *Set[T]) MarshalJSON() ([]byte, error) {
	// Convert the set to a slice and encode it as JSON
	return json.Marshal(s.List())
}

// UnmarshalJSON implements the json.Unmarshaler interface
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
