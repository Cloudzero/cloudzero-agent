// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package metricio

import (
	"encoding/json"
	"sort"
	"strconv"
	"time"

	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/prometheus/prometheus/prompb"
)

// ConvertToTimeSeries converts a slice of ParquetMetric to prompb.TimeSeries.
// Labels are sorted to ensure consistent ordering for Prometheus/Mimir.
func ConvertToTimeSeries(metrics []types.ParquetMetric) []prompb.TimeSeries {
	timeSeries := make([]prompb.TimeSeries, 0, len(metrics))

	for _, pm := range metrics {
		ts := convertSingle(pm)
		timeSeries = append(timeSeries, ts)
	}

	return timeSeries
}

func convertSingle(pm types.ParquetMetric) prompb.TimeSeries {
	// Start with base labels
	labels := []prompb.Label{
		{Name: "__name__", Value: pm.MetricName},
	}

	// Add non-empty base fields as labels
	if pm.ClusterName != "" {
		labels = append(labels, prompb.Label{Name: "cluster_name", Value: pm.ClusterName})
	}
	if pm.CloudAccountID != "" {
		labels = append(labels, prompb.Label{Name: "cloud_account_id", Value: pm.CloudAccountID})
	}
	if pm.NodeName != "" {
		labels = append(labels, prompb.Label{Name: "node_name", Value: pm.NodeName})
	}

	// Parse and add JSON labels (excluding __name__ since we already have it)
	if pm.Labels != "" {
		var extraLabels map[string]string
		if err := json.Unmarshal([]byte(pm.Labels), &extraLabels); err == nil {
			for k, v := range extraLabels {
				// Skip __name__ as it's already set from MetricName
				if k == "__name__" {
					continue
				}
				// Skip empty values
				if v == "" {
					continue
				}
				labels = append(labels, prompb.Label{Name: k, Value: v})
			}
		}
	}

	// Sort labels by name for consistent ordering (required by Prometheus)
	sort.Slice(labels, func(i, j int) bool {
		return labels[i].Name < labels[j].Name
	})

	// Parse value
	value, _ := strconv.ParseFloat(pm.Value, 64)

	return prompb.TimeSeries{
		Labels: labels,
		Samples: []prompb.Sample{{
			Value:     value,
			Timestamp: pm.TimeStamp, // Already Unix milliseconds
		}},
	}
}

// CreateWriteRequest creates a prompb.WriteRequest from a slice of TimeSeries.
func CreateWriteRequest(timeSeries []prompb.TimeSeries) *prompb.WriteRequest {
	return &prompb.WriteRequest{
		Timeseries: timeSeries,
	}
}

// TimeSeriesToMetrics converts a prompb.TimeSeries to a slice of types.Metric.
// Each sample in the TimeSeries produces a separate Metric.
func TimeSeriesToMetrics(ts prompb.TimeSeries) []types.Metric {
	metrics := make([]types.Metric, 0, len(ts.Samples))

	// Extract labels once
	labels := make(map[string]string)
	var metricName, nodeName, clusterName, cloudAccountID string
	for _, l := range ts.Labels {
		switch l.Name {
		case "__name__":
			metricName = l.Value
		case "node":
			nodeName = l.Value
		case "cluster_name":
			clusterName = l.Value
		case "cloud_account_id":
			cloudAccountID = l.Value
		default:
			labels[l.Name] = l.Value
		}
	}

	// Create a Metric for each sample
	now := time.Now().UTC()
	for _, sample := range ts.Samples {
		metrics = append(metrics, types.Metric{
			MetricName:     metricName,
			NodeName:       nodeName,
			ClusterName:    clusterName,
			CloudAccountID: cloudAccountID,
			Labels:         labels,
			TimeStamp:      time.UnixMilli(sample.Timestamp).UTC(),
			CreatedAt:      now,
			Value:          strconv.FormatFloat(sample.Value, 'f', -1, 64),
		})
	}
	return metrics
}

// MetricToTimeSeries converts a types.Metric to a prompb.TimeSeries.
// Labels are sorted alphabetically as required by Prometheus.
func MetricToTimeSeries(m types.Metric) prompb.TimeSeries {
	// Get all labels including __name__ and node
	fullLabels := m.FullLabels()

	// Add cluster_name and cloud_account_id if present
	if m.ClusterName != "" {
		fullLabels["cluster_name"] = m.ClusterName
	}
	if m.CloudAccountID != "" {
		fullLabels["cloud_account_id"] = m.CloudAccountID
	}

	// Convert to prompb.Label slice
	labels := make([]prompb.Label, 0, len(fullLabels))
	for k, v := range fullLabels {
		if v == "" {
			continue
		}
		labels = append(labels, prompb.Label{Name: k, Value: v})
	}

	// Sort labels alphabetically (required by Prometheus)
	sort.Slice(labels, func(i, j int) bool {
		return labels[i].Name < labels[j].Name
	})

	// Parse value
	value, _ := strconv.ParseFloat(m.Value, 64)

	return prompb.TimeSeries{
		Labels: labels,
		Samples: []prompb.Sample{{
			Value:     value,
			Timestamp: m.TimeStamp.UnixMilli(),
		}},
	}
}
