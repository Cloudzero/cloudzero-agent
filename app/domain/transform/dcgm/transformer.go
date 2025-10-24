// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package dcgm provides NVIDIA DCGM metric transformation for cost allocation.
//
// This package implements types.MetricTransformer following hexagonal architecture principles.
// It transforms NVIDIA DCGM exporter metrics into standardized GPU metrics suitable for
// cost allocation and monitoring.
//
// # Transformation Rules
//
//   - DCGM_FI_DEV_GPU_UTIL → container_resources_gpu_usage_percent (pass-through percentage)
//   - DCGM_FI_DEV_FB_USED + FB_FREE → container_resources_gpu_memory_usage_percent (calculated percentage)
//
// # Processing Strategy
//
// Memory metrics are buffered during Transform() and calculated during the final flush phase
// to ensure paired USED/FREE metrics are processed together for accurate percentage calculation.
//
// # Architecture
//
//	catalog.Transformer (routes to specialized transformers)
//	  └── dcgm.Transformer (handles NVIDIA DCGM metrics)
//
// Future GPU vendors (Intel XPU, AMD ROCm) would be implemented as peer packages.
package dcgm

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/cloudzero/cloudzero-agent/app/types"
)

// NVIDIA DCGM metric names that we transform.
const (
	dcgmGPUUtilization = "DCGM_FI_DEV_GPU_UTIL" // GPU utilization percentage
	dcgmMemoryUsed     = "DCGM_FI_DEV_FB_USED"  // Framebuffer memory used (MiB)
	dcgmMemoryFree     = "DCGM_FI_DEV_FB_FREE"  // Framebuffer memory free (MiB)
)

// Standardized container-level GPU metric names.
const (
	standardGPUUsage       = "container_resources_gpu_usage_percent"
	standardGPUMemoryUsage = "container_resources_gpu_memory_usage_percent"
)

// Required labels for GPU metric attribution.
var requiredLabels = []string{"namespace", "pod", "container"}

// Transformer implements types.MetricTransformer for NVIDIA DCGM metrics.
//
// This transformer converts native DCGM exporter metrics into standardized
// container-level GPU resource metrics. It handles both immediate
// transformations (GPU utilization) and buffered transformations (memory
// percentage calculation requiring paired USED/FREE metrics).
type Transformer struct {
	// memoryBuffer stores memory metrics awaiting paired calculation.
	// Key format: "namespace/pod/container/gpu"
	memoryBuffer map[string]*memoryPair
}

// memoryPair tracks USED and FREE memory metrics for percentage calculation.
type memoryPair struct {
	used *types.Metric
	free *types.Metric
}

// NewTransformer creates a new DCGM metric transformer.
func NewTransformer() *Transformer {
	return &Transformer{
		memoryBuffer: make(map[string]*memoryPair),
	}
}

// Transform converts DCGM metrics to standardized format while passing through
// non-DCGM metrics unchanged.
//
// Processing flow:
//  1. For each metric, check if it's a DCGM metric
//  2. If DCGM GPU utilization, transform immediately
//  3. If DCGM memory (USED/FREE), buffer for later calculation
//  4. If not DCGM, pass through unchanged
//  5. Flush memory buffer to calculate percentages from paired metrics
//
// This implements the types.MetricTransformer interface.
func (t *Transformer) Transform(ctx context.Context, metrics []types.Metric) ([]types.Metric, error) {
	if len(metrics) == 0 {
		return metrics, nil
	}

	// Estimate result capacity (metrics may expand during transformation).
	result := make([]types.Metric, 0, len(metrics))

	// Transform each metric
	for _, metric := range metrics {
		transformed, err := t.transformSingle(ctx, metric)
		if err != nil {
			return nil, err
		}
		result = append(result, transformed...)
	}

	// Flush memory buffer to get calculated memory percentage metrics.
	flushed, err := t.flushMemory(ctx)
	if err != nil {
		return nil, err
	}
	result = append(result, flushed...)

	return result, nil
}

// transformSingle transforms a single metric. Returns the metric unchanged if
// it's not a DCGM metric.
func (t *Transformer) transformSingle(ctx context.Context, metric types.Metric) ([]types.Metric, error) {
	// Check if this is a DCGM metric
	if !strings.HasPrefix(metric.MetricName, "DCGM_FI_DEV_") {
		// Not a DCGM metric - pass through unchanged
		return []types.Metric{metric}, nil
	}

	// Validate required labels for cost attribution
	if !hasRequiredLabels(metric) {
		log.Ctx(ctx).Debug().
			Str("metric", metric.MetricName).
			Interface("labels", metric.Labels).
			Msg("dropping DCGM metric missing required labels")
		return []types.Metric{}, nil
	}

	switch metric.MetricName {
	case dcgmGPUUtilization:
		// GPU utilization is already a percentage - just rename and return
		return transformGPUUtilization(metric), nil

	case dcgmMemoryUsed:
		// Buffer for later percentage calculation
		t.bufferMemoryMetric(metric, true)
		return []types.Metric{}, nil

	case dcgmMemoryFree:
		// Buffer for later percentage calculation
		t.bufferMemoryMetric(metric, false)
		return []types.Metric{}, nil

	default:
		// Unknown DCGM metric - pass through unchanged
		return []types.Metric{metric}, nil
	}
}

// flushMemory calculates and returns GPU memory percentage metrics from buffered USED/FREE pairs.
// After flush, the memory buffer is cleared.
//
// Memory percentage is calculated as: (used / (used + free)) * 100
//
// Incomplete pairs (missing either USED or FREE) are dropped with debug logging.
func (t *Transformer) flushMemory(ctx context.Context) ([]types.Metric, error) {
	if len(t.memoryBuffer) == 0 {
		return []types.Metric{}, nil
	}

	result := make([]types.Metric, 0, len(t.memoryBuffer))

	for key, pair := range t.memoryBuffer {
		if pair.used == nil || pair.free == nil {
			log.Ctx(ctx).Debug().
				Str("key", key).
				Bool("hasUsed", pair.used != nil).
				Bool("hasFree", pair.free != nil).
				Msg("dropping incomplete memory metric pair")
			continue
		}

		// Parse string values to floats for calculation
		used, err := parseFloat(pair.used.Value)
		if err != nil {
			log.Ctx(ctx).Debug().
				Str("key", key).
				Str("usedValue", pair.used.Value).
				Err(err).
				Msg("dropping memory metric with invalid used value")
			continue
		}

		free, err := parseFloat(pair.free.Value)
		if err != nil {
			log.Ctx(ctx).Debug().
				Str("key", key).
				Str("freeValue", pair.free.Value).
				Err(err).
				Msg("dropping memory metric with invalid free value")
			continue
		}

		total := used + free
		if total == 0 {
			log.Ctx(ctx).Debug().
				Str("key", key).
				Msg("dropping memory metric with zero total")
			continue
		}

		percentage := (used / total) * 100.0

		// Extract node name from field or labels. DCGM uses "Hostname" label,
		// other exporters may use "node".
		nodeName := pair.used.NodeName
		if nodeName == "" {
			nodeName = pair.used.Labels["node"]
		}
		if nodeName == "" {
			nodeName = pair.used.Labels["Hostname"]
		}

		// Create standardized memory usage metric. Use timestamp and metadata
		// from the USED metric.
		memoryMetric := types.Metric{
			ID:             uuid.New(),
			ClusterName:    pair.used.ClusterName,
			CloudAccountID: pair.used.CloudAccountID,
			MetricName:     standardGPUMemoryUsage,
			NodeName:       nodeName,
			Value:          formatFloat(percentage),
			TimeStamp:      pair.used.TimeStamp,
			CreatedAt:      pair.used.CreatedAt,
			Labels:         transformLabels(pair.used.Labels),
		}

		result = append(result, memoryMetric)
	}

	// Clear buffer after flush
	t.memoryBuffer = make(map[string]*memoryPair)

	return result, nil
}

// transformGPUUtilization converts DCGM GPU utilization to standardized format.
func transformGPUUtilization(metric types.Metric) []types.Metric {
	// Extract node name from field or labels. DCGM uses "Hostname" label, other
	// exporters may use "node".
	nodeName := metric.NodeName
	if nodeName == "" {
		nodeName = metric.Labels["node"]
	}
	if nodeName == "" {
		nodeName = metric.Labels["Hostname"]
	}

	return []types.Metric{
		{
			ID:             uuid.New(),
			ClusterName:    metric.ClusterName,
			CloudAccountID: metric.CloudAccountID,
			MetricName:     standardGPUUsage,
			NodeName:       nodeName,
			Value:          metric.Value,
			TimeStamp:      metric.TimeStamp,
			CreatedAt:      metric.CreatedAt,
			Labels:         transformLabels(metric.Labels),
		},
	}
}

// bufferMemoryMetric stores a memory metric for later percentage calculation.
func (t *Transformer) bufferMemoryMetric(metric types.Metric, isUsed bool) {
	key := makeMemoryKey(metric)

	pair, exists := t.memoryBuffer[key]
	if !exists {
		pair = &memoryPair{}
		t.memoryBuffer[key] = pair
	}

	if isUsed {
		pair.used = &metric
	} else {
		pair.free = &metric
	}
}

// makeMemoryKey creates a unique key for buffering memory metrics. Format:
// "namespace/pod/container/gpu"
func makeMemoryKey(metric types.Metric) string {
	return fmt.Sprintf("%s/%s/%s/%s",
		metric.Labels["namespace"],
		metric.Labels["pod"],
		metric.Labels["container"],
		metric.Labels["gpu"],
	)
}

// hasRequiredLabels checks if metric has all required labels for cost
// attribution.
func hasRequiredLabels(metric types.Metric) bool {
	for _, label := range requiredLabels {
		if _, exists := metric.Labels[label]; !exists {
			return false
		}
	}
	return true
}

// transformLabels creates a shallow copy of the labels map with standardization
// transformations. This performs vendor-specific label transformations to
// ensure compatibility with standardized GPU metrics.
func transformLabels(labels map[string]string) map[string]string {
	if labels == nil {
		return nil
	}

	result := make(map[string]string, len(labels))
	for k, v := range labels {
		// Rename DCGM's "UUID" label to standardized "gpu_uuid"
		// Strip "GPU-" prefix: "GPU-4980eea4-..." becomes "4980eea4-..."
		if k == "UUID" {
			result["gpu_uuid"] = strings.TrimPrefix(v, "GPU-")
		} else {
			result[k] = v
		}
	}
	return result
}

// parseFloat converts a string value to float64.
func parseFloat(value string) (float64, error) {
	return strconv.ParseFloat(value, 64)
}

// formatFloat converts a float64 value to string.
func formatFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
