// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package types_test

import (
	"testing"
	"time"

	"github.com/cloudzero/cloudzero-insights-controller/app/types"
	"github.com/stretchr/testify/assert"
)

func TestNewMetric(t *testing.T) {
	name := "test_metric"
	nodeName := "node-1"
	timeStamp := time.Now().UnixMilli()
	labels := map[string]string{"env": "test"}
	value := "123.45"

	metric := types.NewMetric("org", "cloudaccount", "cluster", name, nodeName, timeStamp, labels, value)

	assert.NotEmpty(t, metric.ID)
	assert.Equal(t, name, metric.Name)
	assert.Equal(t, nodeName, metric.NodeName)
	assert.NotZero(t, metric.CreatedAt)
	assert.Equal(t, timeStamp, metric.TimeStamp)
	assert.Equal(t, labels, metric.Labels)
	assert.Equal(t, value, metric.Value)
}

func TestMetricRange(t *testing.T) {
	metrics := []types.Metric{
		types.NewMetric("org", "cloudaccount", "cluster", "metric1", "node1", time.Now().UnixMilli(), map[string]string{"env": "test"}, "123.45"),
		types.NewMetric("org", "cloudaccount", "cluster", "metric2", "node1", time.Now().UnixMilli(), map[string]string{"env": "prod"}, "678.90"),
	}
	next := "next_token"

	metricRange := types.MetricRange{
		Metrics: metrics,
		Next:    &next,
	}

	assert.Len(t, metricRange.Metrics, 2)
	assert.Equal(t, "metric1", metricRange.Metrics[0].Name)
	assert.Equal(t, "metric2", metricRange.Metrics[1].Name)
	assert.Equal(t, "node1", metricRange.Metrics[0].NodeName)
	assert.Equal(t, "node1", metricRange.Metrics[1].NodeName)
	assert.NotNil(t, metricRange.Next)
	assert.Equal(t, next, *metricRange.Next)
}
