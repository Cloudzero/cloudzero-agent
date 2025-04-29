package types_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudzero/cloudzero-agent/app/types"
)

func TestObservablityMetric(t *testing.T) {
	testCases := []struct {
		name        string
		metricName  string
		expected    string
		expectPanic bool
	}{
		{
			name:       "Valid metric name",
			metricName: "valid_metric",
			expected:   "czo_valid_metric",
		},
		{
			name:        "Empty metric name",
			metricName:  "",
			expectPanic: true,
		},
		{
			name:        "Metric name with forbidden prefix 'czo'",
			metricName:  "czo_metric",
			expectPanic: true,
		},
		{
			name:        "Metric name with forbidden prefix 'cz'",
			metricName:  "cz_metric",
			expectPanic: true,
		},
		{
			name:        "Metric name with forbidden prefix 'cloudzero'",
			metricName:  "cloudzero_metric",
			expectPanic: true,
		},
		{
			name:        "Metric name with empty prefix",
			metricName:  "_metric",
			expectPanic: true,
		},
		{
			name:        "Metric name with no parts after splitting",
			metricName:  "valid",
			expectPanic: false,
			expected:    "czo_valid",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.expectPanic {
				assert.Panics(t, func() {
					types.ObservabilityMetric(tc.metricName)
				})
			} else {
				result := types.ObservabilityMetric(tc.metricName)
				require.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestCostMetric(t *testing.T) {
	testCases := []struct {
		name        string
		metricName  string
		expected    string
		expectPanic bool
	}{
		{
			name:       "Valid metric name",
			metricName: "valid_metric",
			expected:   "cloudzero_valid_metric",
		},
		{
			name:        "Empty metric name",
			metricName:  "",
			expectPanic: true,
		},
		{
			name:        "Metric name with forbidden prefix 'czo'",
			metricName:  "czo_metric",
			expectPanic: true,
		},
		{
			name:        "Metric name with forbidden prefix 'cz'",
			metricName:  "cz_metric",
			expectPanic: true,
		},
		{
			name:        "Metric name with forbidden prefix 'cloudzero'",
			metricName:  "cloudzero_metric",
			expectPanic: true,
		},
		{
			name:        "Metric name with empty prefix",
			metricName:  "_metric",
			expectPanic: true,
		},
		{
			name:        "Metric name with no parts after splitting",
			metricName:  "valid",
			expectPanic: false,
			expected:    "cloudzero_valid",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.expectPanic {
				assert.Panics(t, func() {
					types.CostMetric(tc.metricName)
				})
			} else {
				result := types.CostMetric(tc.metricName)
				require.Equal(t, tc.expected, result)
			}
		})
	}
}
