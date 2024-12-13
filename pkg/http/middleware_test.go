// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package http_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"

	requestMiddleware "github.com/cloudzero/cloudzero-insights-controller/pkg/http"
)

// TestMetricsMiddlewareWrapper ensures that our instrumentation produces the expected Prometheus metrics.
// Specifically, we test:
// 1. The counter metric: http_requests_total
// 2. The histogram metric: http_request_duration_seconds
//
// For http_requests_total, we use an exact textual comparison since it’s stable.
// For http_request_duration_seconds, we gather the metrics and check them programmatically since the sum can vary slightly.
func TestMetricsMiddlewareWrapper(t *testing.T) {
	// make sure other tests don't have an impact on this one
	// since they use the middleware as well
	requestMiddleware.RequestCount.Reset()
	requestMiddleware.RequestDuration.Reset()

	// Create a test handler and wrap it with the MetricsMiddlewareWrapper
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
	handler := requestMiddleware.MetricsMiddlewareWrapper(testHandler)

	// Execute a request to produce some metrics
	req, err := http.NewRequest(http.MethodGet, "/test", nil)
	assert.NoError(t, err)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	// Verify the response
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "OK", rr.Body.String())

	//-----------------------------------------------------------------------
	// 1. Exact Textual Comparison for http_requests_total
	//
	// This metric should be stable and not vary between test runs. We expect a single counter increment.
	// The label order is method, route, then status_code.
	//-----------------------------------------------------------------------
	expectedCount := `# HELP http_requests_total Count of all HTTP requests processed, labeled by route, method and status code.
# TYPE http_requests_total counter
http_requests_total{method="GET",route="/test",status_code="200"} 1
`
	assert.NoError(t, testutil.CollectAndCompare(
		requestMiddleware.RequestCount,
		strings.NewReader(expectedCount),
		"http_requests_total",
	))

	//-----------------------------------------------------------------------
	// 2. Programmatic Check for http_request_duration_seconds
	//
	// We know this metric is exported, but the sum will vary slightly due to timing.
	// Instead of relying on exact textual comparison, we gather all metrics and verify:
	//  - The metric family exists
	//  - The metric with our desired labels exists
	//  - The count is 1 (we made one request)
	//  - The sum is >= 0 (no invalid negative values)
	//
	// This approach future-proofs the test against minor timing differences.
	//-----------------------------------------------------------------------
	metrics, err := prometheus.DefaultGatherer.Gather()
	assert.NoError(t, err)

	var histMetric *dto.Metric
	for _, mf := range metrics {
		switch mf.GetName() {
		case "http_request_duration_seconds":
			// Find the metric with labels {method="GET", route="/test", status_code="200"}
			for _, m := range mf.Metric {
				if hasLabels(m, map[string]string{
					"method":      "GET",
					"route":       "/test",
					"status_code": "200",
				}) {
					histMetric = m
					break
				}
			}
		}
	}

	assert.NotNil(t, histMetric, "expected to find histogram metric for GET /test 200")
	hist := histMetric.GetHistogram()
	assert.Equal(t, uint64(1), hist.GetSampleCount(), "expected exactly one request recorded in the histogram")
	assert.True(t, hist.GetSampleSum() >= 0.0, "expected a non-negative sum")
}

// Helper function to check that a metric has all the expected labels.
func hasLabels(m *dto.Metric, expected map[string]string) bool {
	for k, v := range expected {
		found := false
		for _, lp := range m.Label {
			if lp.GetName() == k && lp.GetValue() == v {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
