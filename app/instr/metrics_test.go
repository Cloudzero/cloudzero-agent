// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package instr

import (
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/require"
)

var testMetric = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "test_prom_metric",
		Help: "used for running basic prometheus tests",
	},
	[]string{"name"},
)

// resets all of the sync counters
func _testResentSync() {
	registerOnce = sync.Once{}
	httpMiddlewareStatsOnce = sync.Once{}
	spanOnce = sync.Once{}
}

// send an http request expecting prometheus metrics back
// and returns them as a string
func _testSendPromRequest(t *testing.T, url string) string {
	resp, err := http.Get(url)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return string(body)
}

func TestPrometheusMetricsRegistry(t *testing.T) {
	defer _testResentSync()

	// using internal registry
	t.Run("TestInternalRegistry", func(t *testing.T) {
		defer _testResentSync()
		p, err := NewPrometheusMetrics(
			WithPromMetrics(testMetric),
		)
		require.NoError(t, err)
		defer p.clearRegistry()

		srv := httptest.NewServer(p.Handler()) // internal handler
		defer srv.Close()

		testMetric.WithLabelValues("test").Inc()
		body := _testSendPromRequest(t, srv.URL)
		require.Contains(t, string(body), "test_prom_metric")
	})

	// using internal registry with default handler
	t.Run("TestInternalRegistryDefaultHandler", func(t *testing.T) {
		defer _testResentSync()
		p, err := NewPrometheusMetrics(
			WithPromMetrics(testMetric),
		)
		require.NoError(t, err)
		defer p.clearRegistry()

		srv := httptest.NewServer(promhttp.Handler()) // default handler
		defer srv.Close()

		testMetric.WithLabelValues("test").Inc()
		body := _testSendPromRequest(t, srv.URL)
		require.NotContains(t, string(body), "test_prom_metric")
	})

	t.Run("TestInternalRegistryNoGoMetrics", func(t *testing.T) {
		defer _testResentSync()
		p, err := NewPrometheusMetrics(
			WithPromMetrics(testMetric),
			WithNoGoMetrics(),
		)
		require.NoError(t, err)
		defer p.clearRegistry()

		srv := httptest.NewServer(p.Handler()) // internal handler
		defer srv.Close()

		testMetric.WithLabelValues("test").Inc()
		body := _testSendPromRequest(t, srv.URL)
		require.NotContains(t, string(body), "go_gc_duration_seconds")
	})

	// use default registry
	t.Run("TestDefaultRegistry", func(t *testing.T) {
		defer _testResentSync()
		p, err := NewPrometheusMetrics(
			WithGlobalRegistry(),
			WithPromMetrics(testMetric),
		)
		require.NoError(t, err)
		defer p.clearRegistry()

		srv := httptest.NewServer(p.Handler()) // internal handler
		defer srv.Close()

		testMetric.WithLabelValues("test").Inc()
		body := _testSendPromRequest(t, srv.URL)
		require.Contains(t, string(body), "test_prom_metric")
	})

	// use default registry with default handler
	t.Run("TestDefaultRegistryDefaultHandler", func(t *testing.T) {
		defer _testResentSync()
		p, err := NewPrometheusMetrics(
			WithGlobalRegistry(),
			WithPromMetrics(testMetric),
		)
		require.NoError(t, err)
		defer p.clearRegistry()

		srv := httptest.NewServer(promhttp.Handler()) // default handler
		defer srv.Close()

		testMetric.WithLabelValues("test").Inc()
		body := _testSendPromRequest(t, srv.URL)
		require.Contains(t, string(body), "test_prom_metric")
	})
}

func TestPrometheusMetricsMiddleware(t *testing.T) {
	defer _testResentSync()

	// create the prom struct
	p, err := NewPrometheusMetrics(
		WithPromMetrics(testMetric),
	)
	require.NoError(t, err)

	// prom handler
	srv := httptest.NewServer(p.Handler())
	defer srv.Close()

	// send a test request using the middleware
	req := httptest.NewRequest("GET", "http://example.com/foo", nil)
	rr := httptest.NewRecorder()
	handler := p.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	handler.ServeHTTP(rr, req)
	require.Equal(t, http.StatusOK, rr.Code)

	// send a request to prometheus
	body := _testSendPromRequest(t, srv.URL)
	require.Contains(t, body, "http_request_duration_seconds_sum{")
	require.Contains(t, body, "http_requests_total{")
}
