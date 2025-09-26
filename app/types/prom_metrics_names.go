// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package types provides Prometheus metric naming utilities for the CloudZero Agent.
// This package contains functions for creating properly prefixed metric names that
// comply with CloudZero's metric taxonomy and avoid naming conflicts with existing
// Prometheus metrics or reserved CloudZero prefixes.
//
// The CloudZero Agent distinguishes between two primary metric categories:
//   - Cost metrics: Prefixed with "cloudzero_" for cost allocation and financial analysis
//   - Observability metrics: Prefixed with "czo_" for operational monitoring and system health
//
// These functions enforce naming conventions that prevent conflicts with existing
// CloudZero infrastructure metrics and ensure consistent metric categorization
// across the entire cost optimization platform.
package types

import "strings"

// CostMetric creates a properly prefixed cost metric name for CloudZero financial analysis.
// This function transforms user-provided metric names into the standardized "cloudzero_" prefix
// format required by the CloudZero cost allocation platform for billing and cost optimization.
//
// Cost metrics are used for:
//   - Resource tagging events and cost attribution
//   - Custom billing dimensions and cost center allocation
//   - Financial reporting and cost optimization analysis
//   - Integration with CloudZero's cost intelligence platform
//
// The function enforces strict naming conventions to prevent conflicts with:
//   - Existing CloudZero infrastructure metrics
//   - Reserved CloudZero prefixes (czo, cz, cloudzero)
//   - Empty or malformed metric names
//
// Security and validation:
//   - Panics on empty input to fail fast during development
//   - Panics on forbidden prefixes to prevent metric namespace pollution
//   - Ensures consistent metric categorization across the platform
//
// Example transformations:
//   - "tag_event" → "cloudzero_tag_event"
//   - "custom_billing" → "cloudzero_custom_billing"
//   - "cost_center" → "cloudzero_cost_center"
//
// Usage in metric processing pipeline:
//
//	Used during metric classification to distinguish cost-related metrics
//	from observability metrics, enabling proper routing to financial systems.
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

// ObservabilityMetric creates a properly prefixed observability metric name for CloudZero operational monitoring.
// This function transforms user-provided metric names into the standardized "czo_" prefix
// format used by CloudZero for system health monitoring and operational intelligence.
//
// Observability metrics are used for:
//   - System performance monitoring and alerting
//   - Agent health checks and diagnostic information
//   - Resource utilization tracking and capacity planning
//   - Operational dashboards and SLA monitoring
//
// The "czo_" prefix distinguishes these metrics from cost metrics ("cloudzero_" prefix),
// enabling proper routing to monitoring systems versus financial analysis platforms.
// This separation ensures observability data doesn't interfere with cost calculations
// while maintaining comprehensive system visibility.
//
// The function enforces the same strict naming conventions as CostMetric to prevent:
//   - Conflicts with existing CloudZero infrastructure metrics
//   - Namespace pollution from reserved prefixes
//   - Inconsistent metric categorization across environments
//
// Security and validation:
//   - Panics on empty input to ensure robust error handling during development
//   - Panics on forbidden prefixes to maintain metric namespace integrity
//   - Enforces consistent observability metric identification
//
// Example transformations:
//   - "latency" → "czo_latency"
//   - "cpu_usage" → "czo_cpu_usage"
//   - "memory_pressure" → "czo_memory_pressure"
//
// Usage in monitoring pipeline:
//
//	Used during metric classification to route operational metrics to monitoring
//	systems while keeping cost metrics separate for financial analysis.
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
