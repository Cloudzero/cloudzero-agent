// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package overrides

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNewExtractor(t *testing.T) {
	tests := []struct {
		name        string
		excludeKeys []string
		expected    map[string]bool
	}{
		{
			name:        "no exclude keys",
			excludeKeys: []string{},
			expected:    map[string]bool{},
		},
		{
			name:        "single exclude key",
			excludeKeys: []string{"kubeStateMetrics"},
			expected:    map[string]bool{"kubeStateMetrics": true},
		},
		{
			name:        "multiple exclude keys",
			excludeKeys: []string{"kubeStateMetrics", "prometheus", "grafana"},
			expected:    map[string]bool{"kubeStateMetrics": true, "prometheus": true, "grafana": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := NewExtractor(tt.excludeKeys...)
			if diff := cmp.Diff(extractor.ExcludeKeys, tt.expected); diff != "" {
				t.Errorf("NewExtractor() ExcludeKeys mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExtractor_Extract(t *testing.T) {
	tests := []struct {
		name        string
		configured  map[string]interface{}
		defaults    map[string]interface{}
		excludeKeys []string
		expected    map[string]interface{}
	}{
		{
			name: "identical maps",
			configured: map[string]interface{}{
				"replicas": 3,
				"image":    "nginx:latest",
			},
			defaults: map[string]interface{}{
				"replicas": 3,
				"image":    "nginx:latest",
			},
			expected: map[string]interface{}{},
		},
		{
			name: "simple override",
			configured: map[string]interface{}{
				"replicas": 5,
				"image":    "nginx:latest",
			},
			defaults: map[string]interface{}{
				"replicas": 3,
				"image":    "nginx:latest",
			},
			expected: map[string]interface{}{
				"replicas": 5,
			},
		},
		{
			name: "new key in configured",
			configured: map[string]interface{}{
				"replicas":  3,
				"image":     "nginx:latest",
				"customKey": "customValue",
			},
			defaults: map[string]interface{}{
				"replicas": 3,
				"image":    "nginx:latest",
			},
			expected: map[string]interface{}{
				"customKey": "customValue",
			},
		},
		{
			name: "nested map differences",
			configured: map[string]interface{}{
				"config": map[string]interface{}{
					"database": map[string]interface{}{
						"host": "custom.db.com",
						"port": 5432,
					},
					"cache": map[string]interface{}{
						"enabled": true,
					},
				},
			},
			defaults: map[string]interface{}{
				"config": map[string]interface{}{
					"database": map[string]interface{}{
						"host": "default.db.com",
						"port": 5432,
					},
					"cache": map[string]interface{}{
						"enabled": false,
					},
				},
			},
			expected: map[string]interface{}{
				"config": map[string]interface{}{
					"database": map[string]interface{}{
						"host": "custom.db.com",
					},
					"cache": map[string]interface{}{
						"enabled": true,
					},
				},
			},
		},
		{
			name: "excluded keys",
			configured: map[string]interface{}{
				"kubeStateMetrics": map[string]interface{}{
					"enabled": true,
				},
				"replicas": 5,
			},
			defaults: map[string]interface{}{
				"kubeStateMetrics": map[string]interface{}{
					"enabled": false,
				},
				"replicas": 3,
			},
			excludeKeys: []string{"kubeStateMetrics"},
			expected: map[string]interface{}{
				"replicas": 5,
			},
		},
		{
			name: "empty string values",
			configured: map[string]interface{}{
				"name":        "",
				"description": "non-empty",
			},
			defaults: map[string]interface{}{
				"name":        "default",
				"description": "default-desc",
			},
			expected: map[string]interface{}{
				"name":        "",
				"description": "non-empty",
			},
		},
		{
			name: "boolean values",
			configured: map[string]interface{}{
				"enabled":  true,
				"disabled": false,
			},
			defaults: map[string]interface{}{
				"enabled":  false,
				"disabled": true,
			},
			expected: map[string]interface{}{
				"enabled":  true,
				"disabled": false,
			},
		},
		{
			name: "zero values",
			configured: map[string]interface{}{
				"count":       0,
				"replicas":    3,
				"percentage":  0.0,
				"temperature": 25.5,
			},
			defaults: map[string]interface{}{
				"count":       1,
				"replicas":    3,
				"percentage":  50.0,
				"temperature": 25.5,
			},
			expected: map[string]interface{}{
				"count":      0,
				"percentage": 0.0,
			},
		},
		{
			name: "array values",
			configured: map[string]interface{}{
				"items":       []interface{}{"a", "b", "c"},
				"emptyArray":  []interface{}{},
				"numberArray": []interface{}{1, 2, 3},
			},
			defaults: map[string]interface{}{
				"items":       []interface{}{"a", "b"},
				"emptyArray":  []interface{}{"default"},
				"numberArray": []interface{}{1, 2, 3},
			},
			expected: map[string]interface{}{
				"items":      []interface{}{"a", "b", "c"},
				"emptyArray": []interface{}{},
			},
		},
		{
			name: "nested keys with excluded names should be included",
			configured: map[string]interface{}{
				"kubeStateMetrics": map[string]interface{}{
					"enabled": true,
				},
				"components": map[string]interface{}{
					"kubeStateMetrics": map[string]interface{}{
						"replicas": 3,
						"image":    "custom-image:latest",
					},
				},
			},
			defaults: map[string]interface{}{
				"kubeStateMetrics": map[string]interface{}{
					"enabled": false,
				},
				"components": map[string]interface{}{
					"kubeStateMetrics": map[string]interface{}{
						"replicas": 1,
						"image":    "default-image:latest",
					},
				},
			},
			excludeKeys: []string{"kubeStateMetrics"},
			expected: map[string]interface{}{
				"components": map[string]interface{}{
					"kubeStateMetrics": map[string]interface{}{
						"replicas": 3,
						"image":    "custom-image:latest",
					},
				},
			},
		},
		{
			name: "zero and false values should be included when they differ from defaults",
			configured: map[string]interface{}{
				"replicas":          0,     // zero value vs non-zero default
				"enabledByDefault":  false, // false vs true default
				"customTimeout":     0,     // zero timeout vs default
				"debugMode":         false, // false vs true default
				"retryCount":        0,     // zero retries vs default
				"emptyStringConfig": "",    // empty string vs default
				"unchangedValue":    42,    // same as default
			},
			defaults: map[string]interface{}{
				"replicas":          1,         // non-zero default
				"enabledByDefault":  true,      // true default
				"customTimeout":     30,        // non-zero default
				"debugMode":         true,      // true default
				"retryCount":        3,         // non-zero default
				"emptyStringConfig": "default", // non-empty default
				"unchangedValue":    42,        // same value
			},
			expected: map[string]interface{}{
				"replicas":          0,
				"enabledByDefault":  false,
				"customTimeout":     0,
				"debugMode":         false,
				"retryCount":        0,
				"emptyStringConfig": "", // included because it differs from default
				// unchangedValue is not included because it matches the default
			},
		},
		{
			name: "nil values should be included when they differ from defaults",
			configured: map[string]interface{}{
				"stringWithDefault": nil,    // nil vs string default
				"numberWithDefault": nil,    // nil vs number default
				"boolWithDefault":   nil,    // nil vs bool default
				"mapWithDefault":    nil,    // nil vs map default
				"arrayWithDefault":  nil,    // nil vs array default
				"validString":       "test", // non-nil vs string default
				"sameAsDefault":     "same", // same as default
			},
			defaults: map[string]interface{}{
				"stringWithDefault": "default-string",
				"numberWithDefault": 42,
				"boolWithDefault":   true,
				"mapWithDefault": map[string]interface{}{
					"key": "value",
				},
				"arrayWithDefault": []interface{}{"item1", "item2"},
				"validString":      "different-default",
				"sameAsDefault":    "same",
			},
			expected: map[string]interface{}{
				"stringWithDefault": nil,
				"numberWithDefault": nil,
				"boolWithDefault":   nil,
				"mapWithDefault":    nil,
				"arrayWithDefault":  nil,
				"validString":       "test",
				// sameAsDefault is excluded because it matches the default
			},
		},
		{
			name: "false values should be included when they differ from true defaults",
			configured: map[string]interface{}{
				"annotations": map[string]interface{}{
					"enabled": false, // explicit false vs default true
				},
				"labels": map[string]interface{}{
					"enabled": true, // same as default
				},
			},
			defaults: map[string]interface{}{
				"annotations": map[string]interface{}{
					"enabled": true, // default true
				},
				"labels": map[string]interface{}{
					"enabled": true, // default true
				},
			},
			expected: map[string]interface{}{
				"annotations": map[string]interface{}{
					"enabled": false, // should be included because it differs
				},
				// labels.enabled should be excluded because it matches default
			},
		},
		{
			name: "empty string values should be included when they differ from defaults",
			configured: map[string]interface{}{
				"name":        "",     // empty string vs non-empty default
				"description": "",     // empty string vs non-empty default
				"unchanged":   "same", // same as default
			},
			defaults: map[string]interface{}{
				"name":        "default-name",
				"description": "default-description",
				"unchanged":   "same",
			},
			expected: map[string]interface{}{
				"name":        "", // included because it differs from default
				"description": "", // included because it differs from default
				// unchanged is excluded because it matches default
			},
		},
		{
			name: "different array values should be included",
			configured: map[string]interface{}{
				"patterns": []interface{}{".*"},     // different from default
				"items":    []interface{}{},         // empty vs non-empty default
				"same":     []interface{}{"a", "b"}, // same as default
			},
			defaults: map[string]interface{}{
				"patterns": []interface{}{"app.kubernetes.io/component"},
				"items":    []interface{}{"default"},
				"same":     []interface{}{"a", "b"},
			},
			expected: map[string]interface{}{
				"patterns": []interface{}{".*"},
				"items":    []interface{}{}, // now included because empty array differs from non-empty default
				// same would be excluded because it matches default
			},
		},
		{
			name: "insightsController nested configuration",
			configured: map[string]interface{}{
				"insightsController": map[string]interface{}{
					"annotations": map[string]interface{}{
						"enabled":  false,               // explicit false vs default true
						"patterns": []interface{}{".*"}, // same array as default
					},
					"labels": map[string]interface{}{
						"enabled":  true,                // same as default
						"patterns": []interface{}{".*"}, // different array from default
					},
				},
			},
			defaults: map[string]interface{}{
				"insightsController": map[string]interface{}{
					"annotations": map[string]interface{}{
						"enabled":  false,               // default false (same as configured!)
						"patterns": []interface{}{".*"}, // same pattern
					},
					"labels": map[string]interface{}{
						"enabled":  true,                                         // default true
						"patterns": []interface{}{"app.kubernetes.io/component"}, // different pattern
					},
				},
			},
			expected: map[string]interface{}{
				"insightsController": map[string]interface{}{
					// annotations section would be excluded since all values match defaults
					"labels": map[string]interface{}{
						"patterns": []interface{}{".*"}, // should be included (different array)
						// enabled would be excluded since it's the same
					},
				},
			},
		},
		{
			name: "empty and nil values should be included when they differ from defaults",
			configured: map[string]interface{}{
				"emptyString": "",                       // empty string vs non-empty default
				"emptyArray":  []interface{}{},          // empty array vs non-empty default
				"emptyMap":    map[string]interface{}{}, // empty map vs non-empty default
				"nilValue":    nil,                      // nil vs non-nil default
				"zeroInt":     0,                        // zero vs non-zero default
				"falseValue":  false,                    // false vs true default
				"sameValue":   "unchanged",              // same as default (should be excluded)
			},
			defaults: map[string]interface{}{
				"emptyString": "default-string",
				"emptyArray":  []interface{}{"default"},
				"emptyMap":    map[string]interface{}{"key": "value"},
				"nilValue":    "default-value",
				"zeroInt":     42,
				"falseValue":  true,
				"sameValue":   "unchanged",
			},
			expected: map[string]interface{}{
				"emptyString": "",
				"emptyArray":  []interface{}{},
				"emptyMap":    map[string]interface{}{},
				"nilValue":    nil,
				"zeroInt":     0,
				"falseValue":  false,
				// sameValue correctly excluded since it matches default
			},
		},
		{
			name: "comprehensive test of all value types that differ from defaults",
			configured: map[string]interface{}{
				"emptyString": "",                       // empty string vs non-empty default
				"emptyArray":  []interface{}{},          // empty array vs non-empty default
				"emptyMap":    map[string]interface{}{}, // empty map vs non-empty default
				"nilValue":    nil,                      // nil vs non-nil default
				"zeroInt":     0,                        // zero vs non-zero default
				"falseValue":  false,                    // false vs true default
				"sameValue":   "unchanged",              // same as default (should be excluded)
			},
			defaults: map[string]interface{}{
				"emptyString": "default-string",
				"emptyArray":  []interface{}{"default"},
				"emptyMap":    map[string]interface{}{"key": "value"},
				"nilValue":    "default-value",
				"zeroInt":     42,
				"falseValue":  true,
				"sameValue":   "unchanged",
			},
			expected: map[string]interface{}{
				// All values that differ from defaults should be included
				"emptyString": "",
				"emptyArray":  []interface{}{},
				"emptyMap":    map[string]interface{}{},
				"nilValue":    nil,
				"zeroInt":     0,
				"falseValue":  false,
				// "sameValue" correctly excluded since it matches default
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			extractor := NewExtractor(tt.excludeKeys...)
			result := extractor.Extract(tt.configured, tt.defaults)
			if diff := cmp.Diff(result, tt.expected); diff != "" {
				t.Errorf("Extract() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExtractor_diffMaps(t *testing.T) {
	extractor := NewExtractor()

	tests := []struct {
		name       string
		configured map[string]interface{}
		defaults   map[string]interface{}
		expected   map[string]interface{}
	}{
		{
			name: "deeply nested structure",
			configured: map[string]interface{}{
				"app": map[string]interface{}{
					"config": map[string]interface{}{
						"database": map[string]interface{}{
							"host": "prod.db.com",
							"port": 5432,
							"ssl":  true,
						},
						"redis": map[string]interface{}{
							"host": "redis.prod.com",
							"port": 6379,
						},
					},
				},
			},
			defaults: map[string]interface{}{
				"app": map[string]interface{}{
					"config": map[string]interface{}{
						"database": map[string]interface{}{
							"host": "localhost",
							"port": 5432,
							"ssl":  false,
						},
						"redis": map[string]interface{}{
							"host": "redis.prod.com",
							"port": 6379,
						},
					},
				},
			},
			expected: map[string]interface{}{
				"app": map[string]interface{}{
					"config": map[string]interface{}{
						"database": map[string]interface{}{
							"host": "prod.db.com",
							"ssl":  true,
						},
					},
				},
			},
		},
		{
			name: "mixed types comparison",
			configured: map[string]interface{}{
				"stringVal": "custom",
				"intVal":    100,
				"floatVal":  99.9,
				"boolVal":   true,
				"arrayVal":  []interface{}{1, 2, 3},
				"nilVal":    nil,
				"emptyStr":  "",
				"mapVal": map[string]interface{}{
					"nested": "value",
				},
			},
			defaults: map[string]interface{}{
				"stringVal": "default",
				"intVal":    50,
				"floatVal":  50.5,
				"boolVal":   false,
				"arrayVal":  []interface{}{1, 2},
				"nilVal":    "default",
				"emptyStr":  "default",
				"mapVal": map[string]interface{}{
					"nested": "default",
				},
			},
			expected: map[string]interface{}{
				"stringVal": "custom",
				"intVal":    100,
				"floatVal":  99.9,
				"boolVal":   true,
				"arrayVal":  []interface{}{1, 2, 3},
				"nilVal":    nil, // now included because it differs from default
				"emptyStr":  "",  // now included because it differs from default
				"mapVal": map[string]interface{}{
					"nested": "value",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractor.diffMaps(tt.configured, tt.defaults)
			if diff := cmp.Diff(result, tt.expected); diff != "" {
				t.Errorf("diffMaps() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExtractor_DiffMapsAllValueTypes(t *testing.T) {
	// This test verifies that diffMaps correctly includes all different values
	// including empty strings, empty arrays, empty maps, nil values, and zero/false values

	extractor := &Extractor{
		ExcludeKeys: map[string]bool{"excludeMe": true},
	}

	configured := map[string]interface{}{
		"emptyString": "",                       // empty string vs non-empty default
		"emptyArray":  []interface{}{},          // empty array vs non-empty default
		"emptyMap":    map[string]interface{}{}, // empty map vs non-empty default
		"nilValue":    nil,                      // nil vs non-nil default
		"zeroInt":     0,                        // zero vs non-zero default
		"falseValue":  false,                    // false vs true default
		"sameValue":   "unchanged",              // same as default (should be excluded)
	}

	defaults := map[string]interface{}{
		"emptyString": "default-string",
		"emptyArray":  []interface{}{"default"},
		"emptyMap":    map[string]interface{}{"key": "value"},
		"nilValue":    "default-value",
		"zeroInt":     42,
		"falseValue":  true,
		"sameValue":   "unchanged",
	}

	result := extractor.diffMaps(configured, defaults)

	expected := map[string]interface{}{
		"emptyString": "",
		"emptyArray":  []interface{}{},
		"emptyMap":    map[string]interface{}{},
		"nilValue":    nil,
		"zeroInt":     0,
		"falseValue":  false,
		// "sameValue" should be excluded since it matches default
	}

	if diff := cmp.Diff(result, expected); diff != "" {
		t.Errorf("diffMaps() mismatch (-want +got):\n%s", diff)
	}
}

func TestExtractor_NestedEmptyStringOverrides(t *testing.T) {
	extractor := NewExtractor()

	// Test nested empty string overriding non-empty defaults
	configured := map[string]interface{}{
		"server": map[string]interface{}{
			"emptyDir": map[string]interface{}{
				"sizeLimit": "", // empty string vs non-empty default
			},
		},
		"aggregator": map[string]interface{}{
			"database": map[string]interface{}{
				"emptyDir": map[string]interface{}{
					"sizeLimit": "", // empty string vs same empty default (should be excluded)
				},
			},
		},
	}

	defaults := map[string]interface{}{
		"server": map[string]interface{}{
			"emptyDir": map[string]interface{}{
				"sizeLimit": "8Gi", // non-empty default
			},
		},
		"aggregator": map[string]interface{}{
			"database": map[string]interface{}{
				"emptyDir": map[string]interface{}{
					"sizeLimit": "", // same empty default
				},
			},
		},
	}

	// Empty string should be included when it differs from default
	currentResult := extractor.Extract(configured, defaults)
	currentExpected := map[string]interface{}{
		"server": map[string]interface{}{
			"emptyDir": map[string]interface{}{
				"sizeLimit": "", // included because it differs from "8Gi"
			},
		},
	}

	if diff := cmp.Diff(currentResult, currentExpected); diff != "" {
		t.Errorf("Extract() mismatch (-want +got):\n%s", diff)
	}

	// Verify that diffMaps directly produces the expected result
	directResult := extractor.diffMaps(configured, defaults)
	if diff := cmp.Diff(directResult, currentExpected); diff != "" {
		t.Errorf("diffMaps() mismatch (-want +got):\n%s", diff)
	}
}
