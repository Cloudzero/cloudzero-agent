// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package dcgm

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/cloudzero/cloudzero-agent/app/types"
)

// Test that non-DCGM metrics pass through unchanged.
func TestTransformer_PassThrough(t *testing.T) {
	transformer := NewTransformer()
	ctx := context.Background()
	timestamp := time.Now()

	tests := []struct {
		name  string
		input []types.Metric
	}{
		{
			name: "CPU metrics pass through",
			input: []types.Metric{
				{
					MetricName: "container_cpu_usage_seconds_total",
					Value:      "1.5",
					TimeStamp:  timestamp,
				},
			},
		},
		{
			name: "memory metrics pass through",
			input: []types.Metric{
				{
					MetricName: "container_memory_working_set_bytes",
					Value:      "1073741824",
					TimeStamp:  timestamp,
				},
			},
		},
		{
			name: "network metrics pass through",
			input: []types.Metric{
				{
					MetricName: "container_network_receive_bytes_total",
					Value:      "12345",
					TimeStamp:  timestamp,
				},
			},
		},
		{
			name:  "empty input returns empty output",
			input: []types.Metric{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := transformer.Transform(ctx, tt.input)
			if err != nil {
				t.Fatalf("Transform() error = %v", err)
			}

			// Non-DCGM metrics should pass through unchanged
			if diff := cmp.Diff(tt.input, got, cmpopts.IgnoreFields(types.Metric{}, "ID")); diff != "" {
				t.Errorf("Transform() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// Test GPU utilization transformation.
func TestTransformer_GPUUtilization(t *testing.T) {
	transformer := NewTransformer()
	ctx := context.Background()
	timestamp := time.Now()

	tests := []struct {
		name     string
		input    []types.Metric
		expected []types.Metric
	}{
		{
			name: "transforms DCGM GPU utilization with node label",
			input: []types.Metric{
				{
					ClusterName:    "test-cluster",
					CloudAccountID: "123456789",
					MetricName:     "DCGM_FI_DEV_GPU_UTIL",
					Value:          "85.5",
					TimeStamp:      timestamp,
					CreatedAt:      timestamp,
					Labels: map[string]string{
						"namespace": "default",
						"pod":       "gpu-pod-1",
						"container": "app",
						"gpu":       "0",
						"node":      "gpu-node-1",
					},
				},
			},
			expected: []types.Metric{
				{
					ClusterName:    "test-cluster",
					CloudAccountID: "123456789",
					MetricName:     "container_resources_gpu_usage_percent",
					NodeName:       "gpu-node-1",
					Value:          "85.5",
					TimeStamp:      timestamp,
					CreatedAt:      timestamp,
					Labels: map[string]string{
						"namespace": "default",
						"pod":       "gpu-pod-1",
						"container": "app",
						"gpu":       "0",
						"node":      "gpu-node-1",
					},
				},
			},
		},
		{
			name: "transforms DCGM GPU utilization with Hostname label",
			input: []types.Metric{
				{
					ClusterName:    "test-cluster",
					CloudAccountID: "123456789",
					MetricName:     "DCGM_FI_DEV_GPU_UTIL",
					Value:          "92.0",
					TimeStamp:      timestamp,
					CreatedAt:      timestamp,
					Labels: map[string]string{
						"namespace": "default",
						"pod":       "gpu-pod-2",
						"container": "cuda",
						"gpu":       "1",
						"Hostname":  "ip-10-30-23-129.ec2.internal",
					},
				},
			},
			expected: []types.Metric{
				{
					ClusterName:    "test-cluster",
					CloudAccountID: "123456789",
					MetricName:     "container_resources_gpu_usage_percent",
					NodeName:       "ip-10-30-23-129.ec2.internal",
					Value:          "92.0",
					TimeStamp:      timestamp,
					CreatedAt:      timestamp,
					Labels: map[string]string{
						"namespace": "default",
						"pod":       "gpu-pod-2",
						"container": "cuda",
						"gpu":       "1",
						"Hostname":  "ip-10-30-23-129.ec2.internal",
					},
				},
			},
		},
		{
			name: "renames UUID label to gpu_uuid and strips GPU- prefix",
			input: []types.Metric{
				{
					ClusterName:    "test-cluster",
					CloudAccountID: "123456789",
					MetricName:     "DCGM_FI_DEV_GPU_UTIL",
					Value:          "75.0",
					TimeStamp:      timestamp,
					CreatedAt:      timestamp,
					Labels: map[string]string{
						"namespace": "default",
						"pod":       "gpu-pod-3",
						"container": "ml-app",
						"gpu":       "0",
						"UUID":      "GPU-4980eea4-963e-7b82-ecb9-36ee26fdceb8",
						"modelName": "Tesla T4",
						"Hostname":  "ip-10-30-23-129.ec2.internal",
					},
				},
			},
			expected: []types.Metric{
				{
					ClusterName:    "test-cluster",
					CloudAccountID: "123456789",
					MetricName:     "container_resources_gpu_usage_percent",
					NodeName:       "ip-10-30-23-129.ec2.internal",
					Value:          "75.0",
					TimeStamp:      timestamp,
					CreatedAt:      timestamp,
					Labels: map[string]string{
						"namespace": "default",
						"pod":       "gpu-pod-3",
						"container": "ml-app",
						"gpu":       "0",
						"gpu_uuid":  "4980eea4-963e-7b82-ecb9-36ee26fdceb8",
						"modelName": "Tesla T4",
						"Hostname":  "ip-10-30-23-129.ec2.internal",
					},
				},
			},
		},
		{
			name: "drops GPU utilization missing required labels",
			input: []types.Metric{
				{
					MetricName: "DCGM_FI_DEV_GPU_UTIL",
					Value:      "85.0",
					Labels: map[string]string{
						"gpu": "0",
					},
				},
			},
			expected: []types.Metric{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := transformer.Transform(ctx, tt.input)
			if err != nil {
				t.Fatalf("Transform() error = %v", err)
			}

			opts := []cmp.Option{
				cmpopts.IgnoreFields(types.Metric{}, "ID"),
			}

			if diff := cmp.Diff(tt.expected, got, opts...); diff != "" {
				t.Errorf("Transform() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// Test GPU memory percentage calculation from paired USED/FREE metrics.
func TestTransformer_GPUMemory(t *testing.T) {
	transformer := NewTransformer()
	ctx := context.Background()
	timestamp := time.Now()

	tests := []struct {
		name     string
		input    []types.Metric
		expected []types.Metric
	}{
		{
			name: "calculates memory percentage from paired USED/FREE metrics",
			input: []types.Metric{
				{
					ClusterName:    "test-cluster",
					CloudAccountID: "123456789",
					MetricName:     "DCGM_FI_DEV_FB_USED",
					Value:          "4294967296", // 4GB
					TimeStamp:      timestamp,
					CreatedAt:      timestamp,
					Labels: map[string]string{
						"namespace": "default",
						"pod":       "gpu-pod-1",
						"container": "app",
						"gpu":       "0",
						"node":      "gpu-node-1",
					},
				},
				{
					ClusterName:    "test-cluster",
					CloudAccountID: "123456789",
					MetricName:     "DCGM_FI_DEV_FB_FREE",
					Value:          "12884901888", // 12GB
					TimeStamp:      timestamp,
					CreatedAt:      timestamp,
					Labels: map[string]string{
						"namespace": "default",
						"pod":       "gpu-pod-1",
						"container": "app",
						"gpu":       "0",
						"node":      "gpu-node-1",
					},
				},
			},
			expected: []types.Metric{
				{
					ClusterName:    "test-cluster",
					CloudAccountID: "123456789",
					MetricName:     "container_resources_gpu_memory_usage_percent",
					NodeName:       "gpu-node-1",
					Value:          "25", // 4GB / (4GB + 12GB) * 100 = 25%
					TimeStamp:      timestamp,
					CreatedAt:      timestamp,
					Labels: map[string]string{
						"namespace": "default",
						"pod":       "gpu-pod-1",
						"container": "app",
						"gpu":       "0",
						"node":      "gpu-node-1",
					},
				},
			},
		},
		{
			name: "handles multiple GPU memory metrics",
			input: []types.Metric{
				// GPU 0
				{
					MetricName: "DCGM_FI_DEV_FB_USED",
					Value:      "8589934592", // 8GB
					TimeStamp:  timestamp,
					Labels: map[string]string{
						"namespace": "default",
						"pod":       "pod-1",
						"container": "app",
						"gpu":       "0",
						"node":      "node-1",
					},
				},
				{
					MetricName: "DCGM_FI_DEV_FB_FREE",
					Value:      "8589934592", // 8GB
					TimeStamp:  timestamp,
					Labels: map[string]string{
						"namespace": "default",
						"pod":       "pod-1",
						"container": "app",
						"gpu":       "0",
						"node":      "node-1",
					},
				},
				// GPU 1
				{
					MetricName: "DCGM_FI_DEV_FB_USED",
					Value:      "4294967296", // 4GB
					TimeStamp:  timestamp,
					Labels: map[string]string{
						"namespace": "default",
						"pod":       "pod-1",
						"container": "app",
						"gpu":       "1",
						"node":      "node-1",
					},
				},
				{
					MetricName: "DCGM_FI_DEV_FB_FREE",
					Value:      "12884901888", // 12GB
					TimeStamp:  timestamp,
					Labels: map[string]string{
						"namespace": "default",
						"pod":       "pod-1",
						"container": "app",
						"gpu":       "1",
						"node":      "node-1",
					},
				},
			},
			expected: []types.Metric{
				{
					MetricName: "container_resources_gpu_memory_usage_percent",
					NodeName:   "node-1",
					Value:      "50", // GPU 0: 8GB / 16GB = 50%
					TimeStamp:  timestamp,
					Labels: map[string]string{
						"namespace": "default",
						"pod":       "pod-1",
						"container": "app",
						"gpu":       "0",
						"node":      "node-1",
					},
				},
				{
					MetricName: "container_resources_gpu_memory_usage_percent",
					NodeName:   "node-1",
					Value:      "25", // GPU 1: 4GB / 16GB = 25%
					TimeStamp:  timestamp,
					Labels: map[string]string{
						"namespace": "default",
						"pod":       "pod-1",
						"container": "app",
						"gpu":       "1",
						"node":      "node-1",
					},
				},
			},
		},
		{
			name: "drops incomplete memory pairs (only USED)",
			input: []types.Metric{
				{
					MetricName: "DCGM_FI_DEV_FB_USED",
					Value:      "4294967296",
					Labels: map[string]string{
						"namespace": "default",
						"pod":       "pod-1",
						"container": "app",
						"gpu":       "0",
					},
				},
			},
			expected: []types.Metric{},
		},
		{
			name: "drops incomplete memory pairs (only FREE)",
			input: []types.Metric{
				{
					MetricName: "DCGM_FI_DEV_FB_FREE",
					Value:      "12884901888",
					Labels: map[string]string{
						"namespace": "default",
						"pod":       "pod-1",
						"container": "app",
						"gpu":       "0",
					},
				},
			},
			expected: []types.Metric{},
		},
		{
			name: "drops memory metrics missing required labels",
			input: []types.Metric{
				{
					MetricName: "DCGM_FI_DEV_FB_USED",
					Value:      "4294967296",
					Labels: map[string]string{
						"gpu": "0",
					},
				},
				{
					MetricName: "DCGM_FI_DEV_FB_FREE",
					Value:      "12884901888",
					Labels: map[string]string{
						"gpu": "0",
					},
				},
			},
			expected: []types.Metric{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := transformer.Transform(ctx, tt.input)
			if err != nil {
				t.Fatalf("Transform() error = %v", err)
			}

			opts := []cmp.Option{
				cmpopts.IgnoreFields(types.Metric{}, "ID", "ClusterName", "CloudAccountID", "CreatedAt"),
				cmpopts.SortSlices(func(a, b types.Metric) bool {
					return a.Labels["gpu"] < b.Labels["gpu"]
				}),
			}

			if diff := cmp.Diff(tt.expected, got, opts...); diff != "" {
				t.Errorf("Transform() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

// Test mixed batches of DCGM and non-DCGM metrics.
func TestTransformer_MixedBatch(t *testing.T) {
	transformer := NewTransformer()
	ctx := context.Background()
	timestamp := time.Now()

	input := []types.Metric{
		// Non-DCGM metrics (should pass through)
		{
			MetricName: "container_cpu_usage_seconds_total",
			Value:      "1.5",
			TimeStamp:  timestamp,
		},
		// DCGM GPU utilization (should transform)
		{
			MetricName: "DCGM_FI_DEV_GPU_UTIL",
			Value:      "85.0",
			TimeStamp:  timestamp,
			Labels: map[string]string{
				"namespace": "default",
				"pod":       "gpu-pod",
				"container": "app",
				"gpu":       "0",
				"node":      "gpu-node",
			},
		},
		// More non-DCGM metrics
		{
			MetricName: "container_memory_working_set_bytes",
			Value:      "1073741824",
			TimeStamp:  timestamp,
		},
		// DCGM memory USED (should buffer)
		{
			MetricName: "DCGM_FI_DEV_FB_USED",
			Value:      "4294967296",
			TimeStamp:  timestamp,
			Labels: map[string]string{
				"namespace": "default",
				"pod":       "gpu-pod",
				"container": "app",
				"gpu":       "0",
				"node":      "gpu-node",
			},
		},
		// DCGM memory FREE (should buffer and calculate)
		{
			MetricName: "DCGM_FI_DEV_FB_FREE",
			Value:      "12884901888",
			TimeStamp:  timestamp,
			Labels: map[string]string{
				"namespace": "default",
				"pod":       "gpu-pod",
				"container": "app",
				"gpu":       "0",
				"node":      "gpu-node",
			},
		},
	}

	got, err := transformer.Transform(ctx, input)
	if err != nil {
		t.Fatalf("Transform() error = %v", err)
	}

	// Should get back:
	// - 2 pass-through metrics (CPU, memory)
	// - 1 transformed GPU utilization
	// - 1 calculated GPU memory percentage
	if len(got) != 4 {
		t.Errorf("Transform() returned %d metrics, want 4", len(got))
	}

	// Check that we got the right metric types
	metricNames := make(map[string]int)
	for _, m := range got {
		metricNames[m.MetricName]++
	}

	expectedNames := map[string]int{
		"container_cpu_usage_seconds_total":            1,
		"container_memory_working_set_bytes":           1,
		"container_resources_gpu_usage_percent":        1,
		"container_resources_gpu_memory_usage_percent": 1,
	}

	if diff := cmp.Diff(expectedNames, metricNames); diff != "" {
		t.Errorf("Metric names mismatch (-want +got):\n%s", diff)
	}
}

// Test unknown DCGM metrics pass through.
func TestTransformer_UnknownDCGMMetrics(t *testing.T) {
	transformer := NewTransformer()
	ctx := context.Background()

	input := []types.Metric{
		{
			MetricName: "DCGM_FI_DEV_UNKNOWN_METRIC",
			Value:      "123",
			Labels: map[string]string{
				"namespace": "default",
				"pod":       "gpu-pod",
				"container": "app",
			},
		},
	}

	got, err := transformer.Transform(ctx, input)
	if err != nil {
		t.Fatalf("Transform() error = %v", err)
	}

	// Unknown DCGM metrics should pass through unchanged
	if diff := cmp.Diff(input, got, cmpopts.IgnoreFields(types.Metric{}, "ID")); diff != "" {
		t.Errorf("Transform() mismatch (-want +got):\n%s", diff)
	}
}

// Test edge cases in memory percentage calculation.
func TestTransformer_GPUMemoryEdgeCases(t *testing.T) {
	transformer := NewTransformer()
	ctx := context.Background()
	timestamp := time.Now()

	tests := []struct {
		name     string
		input    []types.Metric
		expected []types.Metric
	}{
		{
			name: "drops memory pair with invalid USED value",
			input: []types.Metric{
				{
					MetricName: "DCGM_FI_DEV_FB_USED",
					Value:      "not-a-number",
					TimeStamp:  timestamp,
					Labels: map[string]string{
						"namespace": "default",
						"pod":       "gpu-pod",
						"container": "app",
						"gpu":       "0",
						"node":      "gpu-node",
					},
				},
				{
					MetricName: "DCGM_FI_DEV_FB_FREE",
					Value:      "12884901888",
					TimeStamp:  timestamp,
					Labels: map[string]string{
						"namespace": "default",
						"pod":       "gpu-pod",
						"container": "app",
						"gpu":       "0",
						"node":      "gpu-node",
					},
				},
			},
			expected: []types.Metric{},
		},
		{
			name: "drops memory pair with invalid FREE value",
			input: []types.Metric{
				{
					MetricName: "DCGM_FI_DEV_FB_USED",
					Value:      "4294967296",
					TimeStamp:  timestamp,
					Labels: map[string]string{
						"namespace": "default",
						"pod":       "gpu-pod",
						"container": "app",
						"gpu":       "0",
						"node":      "gpu-node",
					},
				},
				{
					MetricName: "DCGM_FI_DEV_FB_FREE",
					Value:      "invalid",
					TimeStamp:  timestamp,
					Labels: map[string]string{
						"namespace": "default",
						"pod":       "gpu-pod",
						"container": "app",
						"gpu":       "0",
						"node":      "gpu-node",
					},
				},
			},
			expected: []types.Metric{},
		},
		{
			name: "drops memory pair with zero total",
			input: []types.Metric{
				{
					MetricName: "DCGM_FI_DEV_FB_USED",
					Value:      "0",
					TimeStamp:  timestamp,
					Labels: map[string]string{
						"namespace": "default",
						"pod":       "gpu-pod",
						"container": "app",
						"gpu":       "0",
						"node":      "gpu-node",
					},
				},
				{
					MetricName: "DCGM_FI_DEV_FB_FREE",
					Value:      "0",
					TimeStamp:  timestamp,
					Labels: map[string]string{
						"namespace": "default",
						"pod":       "gpu-pod",
						"container": "app",
						"gpu":       "0",
						"node":      "gpu-node",
					},
				},
			},
			expected: []types.Metric{},
		},
		{
			name: "uses Hostname label when node label and NodeName field are missing",
			input: []types.Metric{
				{
					MetricName: "DCGM_FI_DEV_FB_USED",
					Value:      "4294967296",
					TimeStamp:  timestamp,
					Labels: map[string]string{
						"namespace": "default",
						"pod":       "gpu-pod",
						"container": "app",
						"gpu":       "0",
						"Hostname":  "ip-10-30-23-129.ec2.internal",
					},
				},
				{
					MetricName: "DCGM_FI_DEV_FB_FREE",
					Value:      "12884901888",
					TimeStamp:  timestamp,
					Labels: map[string]string{
						"namespace": "default",
						"pod":       "gpu-pod",
						"container": "app",
						"gpu":       "0",
						"Hostname":  "ip-10-30-23-129.ec2.internal",
					},
				},
			},
			expected: []types.Metric{
				{
					MetricName: "container_resources_gpu_memory_usage_percent",
					NodeName:   "ip-10-30-23-129.ec2.internal",
					Value:      "25",
					TimeStamp:  timestamp,
					Labels: map[string]string{
						"namespace": "default",
						"pod":       "gpu-pod",
						"container": "app",
						"gpu":       "0",
						"Hostname":  "ip-10-30-23-129.ec2.internal",
					},
				},
			},
		},
		{
			name: "handles nil labels map",
			input: []types.Metric{
				{
					MetricName: "DCGM_FI_DEV_GPU_UTIL",
					Value:      "85.0",
					NodeName:   "gpu-node",
					TimeStamp:  timestamp,
					Labels:     nil,
				},
			},
			expected: []types.Metric{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := transformer.Transform(ctx, tt.input)
			if err != nil {
				t.Fatalf("Transform() error = %v", err)
			}

			opts := []cmp.Option{
				cmpopts.IgnoreFields(types.Metric{}, "ID", "ClusterName", "CloudAccountID", "CreatedAt"),
			}

			if diff := cmp.Diff(tt.expected, got, opts...); diff != "" {
				t.Errorf("Transform() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
