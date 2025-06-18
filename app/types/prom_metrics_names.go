// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package types

import "strings"

// CostMetric takes a metric name as input and returns a new metric name
// prefixed with "cloudzero_". This function is used when creating new
// cost metrics for customers, such as tags (e.g., "cloudzero_tag_event").
//
// The input metric name must not be empty and must not start with any
// of the forbidden prefixes: "", "czo", "cz", or "cloudzero". If these
// conditions are violated, the function will panic.
//
// Example usage:
//
//	metric := CostMetric("tag_event") // Returns "cloudzero_tag_event"
func CostMetric(metricName string) string {
	parts := strings.SplitN(metricName, "_", 2)
	if len(parts) == 0 {
		panic("metricName is invalid: no parts found after splitting")
	}
	prefix := parts[0]
	if prefix == "" || prefix == "czo" || prefix == "cz" || prefix == "cloudzero" {
		panic("metricName contains a forbidden prefix or is empty")
	}
	return "cloudzero_" + metricName
}

// ObservabilityMetric takes a metric name as input and returns a new metric name
// prefixed with "czo_". This function is used when creating new
// observability metrics for customers.
//
// The input metric name must not be empty and must not start with any
// of the forbidden prefixes: "", "czo", "cz", or "cloudzero". If these
// conditions are violated, the function will panic.
//
// Example usage:
//
//	metric := ObservabilityMetric("latency") // Returns "cloudzero_obs_latency"
func ObservabilityMetric(metricName string) string {
	parts := strings.SplitN(metricName, "_", 2)
	if len(parts) == 0 {
		panic("metricName is invalid: no parts found after splitting")
	}
	prefix := parts[0]
	if prefix == "" || prefix == "czo" || prefix == "cz" || prefix == "cloudzero" {
		panic("metricName contains a forbidden prefix or is empty")
	}
	return "czo_" + metricName
}
