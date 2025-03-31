// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package types_test

import (
	"encoding/json"
	"testing"

	"github.com/cloudzero/cloudzero-agent-validator/app/types"
	"github.com/stretchr/testify/require"
)

// Written with AI and checked by a human

func TestSetOperations(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		testSetOperations(t,
			[]string{"apple", "banana", "cherry"},
			"apple",
			"banana")
	})

	t.Run("int", func(t *testing.T) {
		testSetOperations(t,
			[]int{1, 2, 3},
			1,
			2)
	})

	t.Run("float32", func(t *testing.T) {
		testSetOperations(t,
			[]float32{1.1, 2.2, 3.3},
			1.1,
			2.2)
	})
}

func testSetOperations[T comparable](t *testing.T, items []T, removeItem, checkItem T) {
	t.Helper()

	// Test Add and Contains
	t.Run("Add", func(t *testing.T) {
		set := types.NewSet[T]()
		for _, item := range items {
			set.Add(item)
		}

		for _, item := range items {
			require.True(t, set.Contains(item), "Set should contain added item")
		}
		require.Equal(t, len(items), set.Size(), "Set size should match number of added items")
	})

	// Test Remove
	t.Run("Remove", func(t *testing.T) {
		set := types.NewSet[T]()
		for _, item := range items {
			set.Add(item)
		}

		set.Remove(removeItem)
		require.False(t, set.Contains(removeItem), "Set should not contain removed item")
		require.Equal(t, len(items)-1, set.Size(), "Set size should decrease after removal")
		require.True(t, set.Contains(checkItem), "Set should still contain other items")
	})

	// Test no duplicates
	t.Run("NoDuplicates", func(t *testing.T) {
		set := types.NewSet[T]()
		set.Add(items[0])
		set.Add(items[0]) // Add the same item twice
		set.Add(items[1])

		require.Equal(t, 2, set.Size(), "Set size should be 2 (no duplicates)")
		require.True(t, set.Contains(items[0]), "Set should contain first item")
		require.True(t, set.Contains(items[1]), "Set should contain second item")
	})

	// Test Diff
	t.Run("Diff", func(t *testing.T) {
		set := types.NewSet[T]()
		for _, item := range items {
			set.Add(item)
		}

		// Create a reference set with some overlapping and some non-overlapping items
		referenceSet := types.NewSet[T]()
		for _, item := range items[1:] { // Assume items[1:] is the reference set
			referenceSet.Add(item)
		}

		// Expected difference: items[:1]
		expectedDiff := types.NewSet[T]()
		for _, item := range items[:1] {
			expectedDiff.Add(item)
		}

		// Compute the difference
		diffSet := set.Diff(referenceSet)

		// Verify the size of the diff set
		require.Equal(t, expectedDiff.Size(), diffSet.Size(), "Diff set size should match the expected number of items")

		// Verify the contents of the diff set
		for _, item := range expectedDiff.List() {
			require.True(t, diffSet.Contains(item), "Diff set should contain expected item")
		}

		// Verify that items in the reference set are not in the diff set
		for _, item := range referenceSet.List() {
			require.False(t, diffSet.Contains(item), "Diff set should not contain items from the reference set")
		}
	})
}

func TestSetJSON(t *testing.T) {
	t.Run("string", func(t *testing.T) {
		testSetJSON(t, []string{"apple", "banana", "cherry"})
	})

	t.Run("int", func(t *testing.T) {
		testSetJSON(t, []int{1, 2, 3})
	})

	t.Run("float32", func(t *testing.T) {
		testSetJSON(t, []float32{1.1, 2.2, 3.3})
	})
}

func testSetJSON[T comparable](t *testing.T, items []T) {
	t.Helper()

	// Test JSON encoding
	t.Run("JSONEncoding", func(t *testing.T) {
		set := types.NewSet[T]()
		for _, item := range items {
			set.Add(item)
		}

		jsonData, err := json.Marshal(set)
		require.NoError(t, err, "JSON encoding should not produce an error")

		var decoded []T
		err = json.Unmarshal(jsonData, &decoded)
		require.NoError(t, err, "JSON decoding should not produce an error")

		require.ElementsMatch(t, items, decoded, "JSON array should match expected elements")
	})

	// Test JSON round trip
	t.Run("JSONRoundTrip", func(t *testing.T) {
		originalSet := types.NewSet[T]()
		for _, item := range items {
			originalSet.Add(item)
		}

		jsonData, err := json.Marshal(originalSet)
		require.NoError(t, err, "JSON encoding should not produce an error")

		decodedSet := types.NewSet[T]()
		err = json.Unmarshal(jsonData, decodedSet)
		require.NoError(t, err, "JSON decoding should not produce an error")

		require.Equal(t, originalSet.Size(), decodedSet.Size(), "Decoded set size should match original")
		for _, item := range items {
			require.True(t, decodedSet.Contains(item), "Decoded set should contain all original items")
		}
	})
}
