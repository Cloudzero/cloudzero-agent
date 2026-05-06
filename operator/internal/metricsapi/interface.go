// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package metricsapi provides the interface and adapter for pod memory metrics retrieval
// used by the CloudZeroAgent reconciler.
package metricsapi

//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=mocks/metrics_client_mock.go -package=mocks -source=interface.go

import "context"

// ComponentMetrics holds the aggregated memory metrics for a single agent component.
type ComponentMetrics struct {
	// UsageBytes is the current memory usage in bytes, aggregated across matching pods/containers.
	UsageBytes int64
	// LimitBytes is the configured memory limit in bytes, aggregated across matching pods/containers.
	LimitBytes int64
}

// MetricsClient abstracts pod memory metrics retrieval for use by the reconciler.
// It is separate from the Kubernetes metrics API so the reconciler can be tested
// without a real metrics server.
type MetricsClient interface {
	// GetComponentMemory returns the aggregated memory usage and limit for pods matching the
	// given label selector. containerName identifies which container to inspect; an empty
	// string aggregates all containers in matching pods.
	// Returns an error if the metrics server is unavailable or no pods are found.
	GetComponentMemory(ctx context.Context, namespace string, labelSelector map[string]string, containerName string) (*ComponentMetrics, error)
}
