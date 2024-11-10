//go:build unit
// +build unit

package types_test

import (
	"testing"
	"time"

	"github.com/cloudzero/cirrus-remote-write/app/types"
	"github.com/stretchr/testify/assert"
)

func TestNewMetric(t *testing.T) {
	name := "test_metric"
	timeStamp := time.Now().UnixMilli()
	labels := map[string]string{"env": "test"}
	value := "123.45"

	metric := types.NewMetric(name, timeStamp, labels, value)

	assert.NotEmpty(t, metric.Id)
	assert.Equal(t, name, metric.Name)
	assert.NotZero(t, metric.CreatedAt)
	assert.Equal(t, timeStamp, metric.TimeStamp)
	assert.Equal(t, labels, metric.Labels)
	assert.Equal(t, value, metric.Value)
}

func TestMetricRange(t *testing.T) {
	metrics := []types.Metric{
		types.NewMetric("metric1", time.Now().UnixMilli(), map[string]string{"env": "test"}, "123.45"),
		types.NewMetric("metric2", time.Now().UnixMilli(), map[string]string{"env": "prod"}, "678.90"),
	}
	next := "next_token"

	metricRange := types.MetricRange{
		Metrics: metrics,
		Next:    &next,
	}

	assert.Len(t, metricRange.Metrics, 2)
	assert.Equal(t, "metric1", metricRange.Metrics[0].Name)
	assert.Equal(t, "metric2", metricRange.Metrics[1].Name)
	assert.NotNil(t, metricRange.Next)
	assert.Equal(t, next, *metricRange.Next)
}
