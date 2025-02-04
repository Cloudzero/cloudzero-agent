// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package instr

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	httpMiddlewareStatsOnce sync.Once

	requestCount = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Count of all HTTP requests processed, labeled by route, method and status code.",
		},
		[]string{"route", "method", "status_code"},
	)

	requestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Histogram of request durations, labeled by route, method and status code.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"route", "method", "status_code"},
	)
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Http middleware that tracks request count and duration
func (p *PrometheusMetrics) Middleware(next http.Handler) http.Handler {
	// register the middleware-specific routes
	httpMiddlewareStatsOnce.Do(func() {
		p.registry.MustRegister(requestCount, requestDuration)
		(*p.metrics) = append((*p.metrics), requestCount)
		(*p.metrics) = append((*p.metrics), requestDuration)
	})

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(recorder, r)

		duration := time.Since(startTime).Seconds()
		statusCode := strconv.Itoa(recorder.status)
		route := r.URL.Path
		method := r.Method

		// Increment the request count
		requestCount.WithLabelValues(route, method, statusCode).Inc()

		// Observe the request duration
		requestDuration.WithLabelValues(route, method, statusCode).Observe(duration)
	})
}
