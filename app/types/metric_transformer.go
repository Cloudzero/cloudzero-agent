// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package types

import "context"

// MetricTransformer defines the port for metric transformation operations in the Application Core.
// This interface enables conversion of vendor-specific metrics into standardized formats
// for cost allocation and resource tracking, following the hexagonal architecture pattern where
// transformation logic is abstracted through a port interface.
//
// MetricTransformer implementations provide:
//   - Vendor-specific metric detection and classification
//   - Transformation of native metrics to standardized formats
//   - Support for multiple device types (GPUs, network devices, storage, etc.)
//   - Pass-through of unrecognized metrics without modification
//
// The transformer operates as part of the metric collection pipeline between decode and
// filter stages:
//
//	Prometheus → Decode → Transform → Filter → Store
//
// Example usage:
//
//	transformer := transform.NewMetricTransformer()
//	transformed, err := transformer.Transform(ctx, metrics)
type MetricTransformer interface {
	// Transform processes a slice of metrics, converting vendor-specific metrics
	// into standardized formats while passing through all other metrics unchanged.
	//
	// The transformation is idempotent - metrics that are already in standardized
	// format or are not recognized pass through without modification.
	//
	// Parameters:
	//   - ctx: Request context for cancellation and tracing
	//   - metrics: Input metrics from Prometheus remote_write decode
	//
	// Returns:
	//   - Transformed metrics with GPU metrics in standardized format
	//   - Error if transformation fails (context cancellation, invalid data, etc.)
	//
	// The returned slice may have a different length than the input if transformation
	// results in metric expansion (e.g., one input metric generating multiple output
	// metrics for different resource attribution).
	Transform(ctx context.Context, metrics []Metric) ([]Metric, error)
}
