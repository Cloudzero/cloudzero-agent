// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package types_test

import (
	"testing"
	"time"

	"github.com/cloudzero/cloudzero-agent-validator/app/types"
	"github.com/cloudzero/cloudzero-agent-validator/app/types/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestNewMetric(t *testing.T) {
	initialTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	mockClock := mocks.NewMockClock(initialTime)
	timeStamp := mockClock.GetCurrentTime()

	name := "test_metric"
	nodeName := "node-1"
	labels := map[string]string{"env": "test"}
	value := "123.45"

	metric := types.Metric{
		ID:             uuid.New(),
		ClusterName:    "cluster",
		CloudAccountID: "cloudaccount",
		MetricName:     name,
		NodeName:       nodeName,
		CreatedAt:      timeStamp,
		TimeStamp:      timeStamp,
		Labels:         labels,
		Value:          value,
	}

	assert.NotEmpty(t, metric.ID)
	assert.Equal(t, name, metric.MetricName)
	assert.Equal(t, nodeName, metric.NodeName)
	assert.NotZero(t, metric.CreatedAt)
	assert.Equal(t, timeStamp, metric.TimeStamp)
	assert.Equal(t, labels, metric.Labels)
	assert.Equal(t, value, metric.Value)
}

func TestMetricRange(t *testing.T) {
	initialTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	mockClock := mocks.NewMockClock(initialTime)

	metrics := []types.Metric{
		{
			ID:             uuid.New(),
			ClusterName:    "cluster",
			CloudAccountID: "cloudaccount",
			MetricName:     "metric1",
			NodeName:       "node1",
			CreatedAt:      mockClock.GetCurrentTime(),
			TimeStamp:      mockClock.GetCurrentTime(),
			Labels:         map[string]string{"env": "test"},
			Value:          "123.45",
		},
		{
			ID:             uuid.New(),
			ClusterName:    "cluster",
			CloudAccountID: "cloudaccount",
			MetricName:     "metric2",
			NodeName:       "node1",
			CreatedAt:      mockClock.GetCurrentTime(),
			TimeStamp:      mockClock.GetCurrentTime(),
			Labels:         map[string]string{"env": "prod"},
			Value:          "678.90",
		},
	}
	next := "next_token"

	metricRange := types.MetricRange{
		Metrics: metrics,
		Next:    &next,
	}

	assert.Len(t, metricRange.Metrics, 2)
	assert.Equal(t, "metric1", metricRange.Metrics[0].MetricName)
	assert.Equal(t, "metric2", metricRange.Metrics[1].MetricName)
	assert.Equal(t, "node1", metricRange.Metrics[0].NodeName)
	assert.Equal(t, "node1", metricRange.Metrics[1].NodeName)
	assert.NotNil(t, metricRange.Next)
	assert.Equal(t, next, *metricRange.Next)
}
