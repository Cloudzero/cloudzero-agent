// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package metrics provides utilities for generating metrics.
package metrics

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cloudzero/cloudzero-agent-validator/app/types"
	"github.com/prometheus/prometheus/prompb"
)

type MetricRecordInput struct {
	OrganizationID string
	CloudAccountID string
	ClusterName    string
	StartTime      time.Time
	EndTime        time.Time
}

func (m MetricRecordInput) GetStartTime() time.Time {
	return m.StartTime.UTC()
}

func (m MetricRecordInput) GetEndTime() time.Time {
	return m.EndTime.UTC()
}

func GenerateClusterMetrics(
	organizationID, cloudAccountID, clusterName string,
	startTime, endTime time.Time,
	cpuPerNode, memPerNode int64,
	numNodes, podsPerNode int,
) []types.Metric {
	metrics := make([]types.Metric, 0)

	// create the input struct
	input := &MetricRecordInput{
		OrganizationID: organizationID,
		CloudAccountID: cloudAccountID,
		ClusterName:    clusterName,
		StartTime:      startTime,
		EndTime:        endTime,
	}

	// create the node records
	for i := range numNodes {
		metrics = append(metrics, GenerateNodeRecords(input, fmt.Sprintf("node-%d", i), fmt.Sprintf("us-west-%d", i+1), cpuPerNode, memPerNode, podsPerNode)...)
	}

	return metrics
}

// EncodeV1 takes a slice of types.Metric and converts it into a prompb.WriteRequest.
func EncodeV1(metrics []types.Metric) (*prompb.WriteRequest, error) {
	// A helper type for grouping time series. The key is a string representing the labels.
	type tsGroup struct {
		labels  []prompb.Label
		samples []prompb.Sample
	}

	// Use a map to group by a canonical representation of the metric's labels.
	groups := make(map[string]*tsGroup)

	// iterate through metrics
	for _, m := range metrics {
		// Create a copy of labels to work with. We expect that when decoding,
		// a metric's labels were imported and thus m.Labels holds the set of labels.
		// It is assumed that every metric has the same label representation as used to
		// originally create the Metric (for example, including the __name__ label).
		// If additional labels (such as NodeName) are stored separately (i.e. in a
		// dedicated field of types.Metric), you might need to add them here.
		// For now, we assume m.Labels contains everything.
		key, labels := buildKeyAndLabels(m.Labels)

		// Convert the sample’s timestamp from time.Time to int64, assuming Unix
		// milliseconds. Adjust if needed.
		timestamp := m.TimeStamp.UnixMilli()

		value, err := strconv.ParseFloat(m.Value, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse value %q: %w", m.Value, err)
		}

		sample := prompb.Sample{
			Timestamp: timestamp,
			Value:     value,
		}

		// Group the samples; if a series with the same labels already exists, add
		// the sample to it. Otherwise, create a new timeseries.
		if group, ok := groups[key]; ok {
			group.samples = append(group.samples, sample)
		} else {
			groups[key] = &tsGroup{
				labels:  labels,
				samples: []prompb.Sample{sample},
			}
		}
	}

	// Convert the groups map into a slice of *prompb.TimeSeries
	var tsList []prompb.TimeSeries
	for _, group := range groups {
		// Optionally, sort samples by timestamp if order is important.
		sort.Slice(group.samples, func(i, j int) bool {
			return group.samples[i].Timestamp < group.samples[j].Timestamp
		})
		ts := prompb.TimeSeries{
			Labels:  group.labels,
			Samples: group.samples,
		}
		tsList = append(tsList, ts)
	}

	// Build the WriteRequest
	req := &prompb.WriteRequest{
		Timeseries: tsList,
	}
	return req, nil
}

// buildKeyAndLabels converts a map of labels (key-value pairs) into a canonical key string
// and a slice of prompb.Labels. The order of labels is fixed (alphabetical by key)
// to ensure that two label maps with the same content produce the same key.
func buildKeyAndLabels(labelsMap map[string]string) (string, []prompb.Label) {
	// extract keys and sort them for a canonical order
	var keys []string
	for k := range labelsMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var labels []prompb.Label
	var sb strings.Builder
	for _, k := range keys {
		v := labelsMap[k]
		labels = append(labels, prompb.Label{
			Name:  k,
			Value: v,
		})
		// Build a key string (e.g., "k1=v1,k2=v2,...")
		sb.WriteString(k)
		sb.WriteString("=")
		sb.WriteString(v)
		sb.WriteString(",")
	}
	return sb.String(), labels
}
