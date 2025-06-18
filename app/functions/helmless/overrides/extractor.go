// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package overrides provides functionality for extracting configuration overrides
// by comparing configured values against default values from Helm charts.
package overrides

import (
	"github.com/google/go-cmp/cmp"
)

// Extractor handles the extraction of configuration overrides by comparing
// configured values against defaults.
type Extractor struct {
	// ExcludeKeys is a set of keys to exclude from comparison
	ExcludeKeys map[string]bool
}

// NewExtractor creates a new Extractor with optional configuration.
func NewExtractor(excludeKeys ...string) *Extractor {
	excludeMap := make(map[string]bool)
	for _, key := range excludeKeys {
		excludeMap[key] = true
	}
	return &Extractor{
		ExcludeKeys: excludeMap,
	}
}

// Extract compares configured values against defaults and returns a map containing
// only the keys whose values differ from the defaults. It recursively compares
// maps and arrays, including all values that differ regardless of their content.
func (e *Extractor) Extract(configured, defaults map[string]interface{}) map[string]interface{} {
	// Create a copy of configured values without excluded keys
	configuredFiltered := make(map[string]interface{})
	for key, value := range configured {
		if !e.ExcludeKeys[key] {
			configuredFiltered[key] = value
		}
	}

	return e.diffMaps(configuredFiltered, defaults)
}

// diffMaps returns a map containing only the keys whose values differ from the
// defaults. It recursively compares maps and arrays, including all values that
// differ regardless of their content.
func (e *Extractor) diffMaps(configured, defaults map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for key, confVal := range configured {
		defVal, exists := defaults[key]
		if !exists {
			// If key doesn't exist in defaults, it's an override
			result[key] = confVal
			continue
		}

		// Both exist, compare values
		if !cmp.Equal(confVal, defVal) {
			// If they're both maps, recursively compare them
			confMap, confIsMap := confVal.(map[string]interface{})
			defMap, defIsMap := defVal.(map[string]interface{})
			if confIsMap && defIsMap {
				diff := e.diffMaps(confMap, defMap)
				if len(diff) > 0 {
					result[key] = diff
				} else if len(confMap) == 0 && len(defMap) > 0 {
					// Special case: empty map overriding non-empty default
					result[key] = confMap
				}
				continue
			}

			// For non-map values, include if different from default
			result[key] = confVal
		}
	}
	return result
}
