// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package metricio

import (
	"testing"
	"time"

	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/prometheus/prometheus/prompb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTimeSeriesToMetrics(t *testing.T) {
	tests := []struct {
		name     string
		ts       prompb.TimeSeries
		expected int
		validate func(t *testing.T, metrics []types.Metric)
	}{
		{
			name: "single sample",
			ts: prompb.TimeSeries{
				Labels: []prompb.Label{
					{Name: "__name__", Value: "test_metric"},
					{Name: "cluster_name", Value: "test-cluster"},
					{Name: "node", Value: "test-node"},
					{Name: "pod", Value: "test-pod"},
				},
				Samples: []prompb.Sample{
					{Value: 42.5, Timestamp: 1704067200000},
				},
			},
			expected: 1,
			validate: func(t *testing.T, metrics []types.Metric) {
				m := metrics[0]
				assert.Equal(t, "test_metric", m.MetricName)
				assert.Equal(t, "test-cluster", m.ClusterName)
				assert.Equal(t, "test-node", m.NodeName)
				assert.Equal(t, "42.5", m.Value)
				assert.Equal(t, "test-pod", m.Labels["pod"])
				assert.Equal(t, int64(1704067200000), m.TimeStamp.UnixMilli())
			},
		},
		{
			name: "multiple samples",
			ts: prompb.TimeSeries{
				Labels: []prompb.Label{
					{Name: "__name__", Value: "cpu_usage"},
					{Name: "node", Value: "worker-1"},
				},
				Samples: []prompb.Sample{
					{Value: 10.0, Timestamp: 1704067200000},
					{Value: 20.0, Timestamp: 1704067260000},
					{Value: 30.0, Timestamp: 1704067320000},
				},
			},
			expected: 3,
			validate: func(t *testing.T, metrics []types.Metric) {
				assert.Equal(t, "cpu_usage", metrics[0].MetricName)
				assert.Equal(t, "10", metrics[0].Value)
				assert.Equal(t, "20", metrics[1].Value)
				assert.Equal(t, "30", metrics[2].Value)
			},
		},
		{
			name: "with cloud_account_id",
			ts: prompb.TimeSeries{
				Labels: []prompb.Label{
					{Name: "__name__", Value: "memory_usage"},
					{Name: "cloud_account_id", Value: "123456789"},
				},
				Samples: []prompb.Sample{
					{Value: 1024.0, Timestamp: 1704067200000},
				},
			},
			expected: 1,
			validate: func(t *testing.T, metrics []types.Metric) {
				assert.Equal(t, "123456789", metrics[0].CloudAccountID)
			},
		},
		{
			name: "empty samples",
			ts: prompb.TimeSeries{
				Labels: []prompb.Label{
					{Name: "__name__", Value: "empty_metric"},
				},
				Samples: []prompb.Sample{},
			},
			expected: 0,
			validate: func(t *testing.T, metrics []types.Metric) {
				assert.Empty(t, metrics)
			},
		},
		{
			name: "extra labels preserved",
			ts: prompb.TimeSeries{
				Labels: []prompb.Label{
					{Name: "__name__", Value: "test"},
					{Name: "environment", Value: "production"},
					{Name: "app", Value: "myapp"},
					{Name: "version", Value: "1.0.0"},
				},
				Samples: []prompb.Sample{
					{Value: 1.0, Timestamp: 1704067200000},
				},
			},
			expected: 1,
			validate: func(t *testing.T, metrics []types.Metric) {
				assert.Equal(t, "production", metrics[0].Labels["environment"])
				assert.Equal(t, "myapp", metrics[0].Labels["app"])
				assert.Equal(t, "1.0.0", metrics[0].Labels["version"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := TimeSeriesToMetrics(tt.ts)
			assert.Len(t, metrics, tt.expected)
			tt.validate(t, metrics)
		})
	}
}

func TestMetricToTimeSeries(t *testing.T) {
	tests := []struct {
		name     string
		metric   types.Metric
		validate func(t *testing.T, ts prompb.TimeSeries)
	}{
		{
			name: "basic metric",
			metric: types.Metric{
				MetricName:     "test_metric",
				NodeName:       "test-node",
				ClusterName:    "test-cluster",
				CloudAccountID: "123456",
				TimeStamp:      time.UnixMilli(1704067200000).UTC(),
				Value:          "42.5",
				Labels: map[string]string{
					"pod": "test-pod",
				},
			},
			validate: func(t *testing.T, ts prompb.TimeSeries) {
				// Check that labels are sorted
				for i := 1; i < len(ts.Labels); i++ {
					assert.True(t, ts.Labels[i-1].Name < ts.Labels[i].Name,
						"labels should be sorted: %s should come before %s",
						ts.Labels[i-1].Name, ts.Labels[i].Name)
				}

				// Check label values
				labelMap := make(map[string]string)
				for _, l := range ts.Labels {
					labelMap[l.Name] = l.Value
				}
				assert.Equal(t, "test_metric", labelMap["__name__"])
				assert.Equal(t, "test-node", labelMap["node"])
				assert.Equal(t, "test-cluster", labelMap["cluster_name"])
				assert.Equal(t, "123456", labelMap["cloud_account_id"])
				assert.Equal(t, "test-pod", labelMap["pod"])

				// Check sample
				require.Len(t, ts.Samples, 1)
				assert.Equal(t, 42.5, ts.Samples[0].Value)
				assert.Equal(t, int64(1704067200000), ts.Samples[0].Timestamp)
			},
		},
		{
			name: "metric with empty fields",
			metric: types.Metric{
				MetricName: "simple_metric",
				TimeStamp:  time.UnixMilli(1704067200000).UTC(),
				Value:      "100",
				Labels:     map[string]string{},
			},
			validate: func(t *testing.T, ts prompb.TimeSeries) {
				// Should have __name__ label
				require.GreaterOrEqual(t, len(ts.Labels), 1)
				assert.Equal(t, "__name__", ts.Labels[0].Name)
				assert.Equal(t, "simple_metric", ts.Labels[0].Value)
			},
		},
		{
			name: "metric with negative value",
			metric: types.Metric{
				MetricName: "negative_metric",
				TimeStamp:  time.UnixMilli(1704067200000).UTC(),
				Value:      "-123.456",
			},
			validate: func(t *testing.T, ts prompb.TimeSeries) {
				require.Len(t, ts.Samples, 1)
				assert.Equal(t, -123.456, ts.Samples[0].Value)
			},
		},
		{
			name: "metric with integer value",
			metric: types.Metric{
				MetricName: "int_metric",
				TimeStamp:  time.UnixMilli(1704067200000).UTC(),
				Value:      "42",
			},
			validate: func(t *testing.T, ts prompb.TimeSeries) {
				require.Len(t, ts.Samples, 1)
				assert.Equal(t, 42.0, ts.Samples[0].Value)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := MetricToTimeSeries(tt.metric)
			tt.validate(t, ts)
		})
	}
}

func TestRoundTrip(t *testing.T) {
	// Test that Metric -> TimeSeries -> Metrics preserves data
	original := types.Metric{
		MetricName:     "roundtrip_test",
		NodeName:       "node-1",
		ClusterName:    "cluster-1",
		CloudAccountID: "account-123",
		TimeStamp:      time.UnixMilli(1704067200000).UTC(),
		Value:          "99.99",
		Labels: map[string]string{
			"environment": "test",
			"app":         "myapp",
		},
	}

	// Convert to TimeSeries
	ts := MetricToTimeSeries(original)

	// Convert back to Metric
	metrics := TimeSeriesToMetrics(ts)
	require.Len(t, metrics, 1)
	result := metrics[0]

	// Verify core fields
	assert.Equal(t, original.MetricName, result.MetricName)
	assert.Equal(t, original.NodeName, result.NodeName)
	assert.Equal(t, original.ClusterName, result.ClusterName)
	assert.Equal(t, original.CloudAccountID, result.CloudAccountID)
	assert.Equal(t, original.TimeStamp.UnixMilli(), result.TimeStamp.UnixMilli())
	assert.Equal(t, original.Value, result.Value)
	assert.Equal(t, original.Labels["environment"], result.Labels["environment"])
	assert.Equal(t, original.Labels["app"], result.Labels["app"])
}

func TestConvertToTimeSeries(t *testing.T) {
	parquetMetrics := []types.ParquetMetric{
		{
			MetricName:     "metric1",
			NodeName:       "node1",
			ClusterName:    "cluster1",
			CloudAccountID: "account1",
			TimeStamp:      1704067200000,
			Value:          "10",
			Labels:         `{"env":"prod"}`,
		},
		{
			MetricName:     "metric2",
			NodeName:       "node2",
			ClusterName:    "cluster2",
			CloudAccountID: "account2",
			TimeStamp:      1704067260000,
			Value:          "20",
			Labels:         `{}`,
		},
	}

	timeSeries := ConvertToTimeSeries(parquetMetrics)

	assert.Len(t, timeSeries, 2)

	// Check first metric
	ts1 := timeSeries[0]
	labelMap := make(map[string]string)
	for _, l := range ts1.Labels {
		labelMap[l.Name] = l.Value
	}
	assert.Equal(t, "metric1", labelMap["__name__"])
	assert.Equal(t, "node1", labelMap["node_name"])
	assert.Equal(t, "prod", labelMap["env"])
	require.Len(t, ts1.Samples, 1)
	assert.Equal(t, 10.0, ts1.Samples[0].Value)
}

func TestCreateWriteRequest(t *testing.T) {
	timeSeries := []prompb.TimeSeries{
		{
			Labels:  []prompb.Label{{Name: "__name__", Value: "test"}},
			Samples: []prompb.Sample{{Value: 1.0, Timestamp: 1000}},
		},
	}

	req := CreateWriteRequest(timeSeries)

	assert.NotNil(t, req)
	assert.Equal(t, timeSeries, req.Timeseries)
}
