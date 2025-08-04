// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package middleware provides standard app middlware implementations
package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

var (
	httpRequestDuration *prometheus.HistogramVec
	httpRequestsTotal   *prometheus.CounterVec
	metricsOnce         sync.Once
)

// getPrometheusMetrics returns initialized prometheus metrics, creating them only once
func getPrometheusMetrics() (*prometheus.HistogramVec, *prometheus.CounterVec) {
	metricsOnce.Do(func() {
		httpRequestDuration = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "http_request_duration_seconds",
				Help: "Duration of HTTP requests in seconds.",
			},
			[]string{"code", "method"},
		)
		httpRequestsTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Count of all HTTP requests processed, labeled by route, method and status code.",
			},
			[]string{"code", "method"},
		)
		// Register metrics with error handling to avoid panics on duplicate registration
		if err := prometheus.Register(httpRequestDuration); err != nil {
			if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
				// Only panic if it's not an AlreadyRegisteredError
				panic(err)
			}
		}
		if err := prometheus.Register(httpRequestsTotal); err != nil {
			if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
				// Only panic if it's not an AlreadyRegisteredError
				panic(err)
			}
		}
	})
	return httpRequestDuration, httpRequestsTotal
}

// PromHTTPMiddleware instruments HTTP requests with Prometheus metrics.
func PromHTTPMiddleware(next http.Handler) http.Handler {
	duration, counter := getPrometheusMetrics()
	return promhttp.InstrumentHandlerDuration(
		duration,
		promhttp.InstrumentHandlerCounter(
			counter,
			next,
		),
	)
}

func LoggingMiddlewareWrapper(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(recorder, r)

		duration := time.Since(startTime)
		statusCode := recorder.status
		route := r.URL.Path
		method := r.Method

		level := zerolog.DebugLevel
		if route == "/healthz" || route == "/metrics" {
			level = zerolog.TraceLevel
		}

		// Log the request details
		log.Ctx(r.Context()).WithLevel(level).
			Str("method", method).
			Str("route", route).
			Int("statusCode", statusCode).
			Str("status", http.StatusText(statusCode)).
			Dur("duration", duration).
			Str("client", r.RemoteAddr).
			Msg("HTTP request")
	})
}
