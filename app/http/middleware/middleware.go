// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package middleware provides standard app middlware implementations
package middleware

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
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

// PromHTTPMiddleware instruments HTTP requests with Prometheus metrics.
func PromHTTPMiddleware(next http.Handler) http.Handler {
	return promhttp.InstrumentHandlerDuration(
		promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name: "http_request_duration_seconds",
				Help: "Duration of HTTP requests in seconds.",
			},
			[]string{"code", "method"},
		),
		promhttp.InstrumentHandlerCounter(
			promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "http_requests_total",
					Help: "Count of all HTTP requests processed, labeled by route, method and status code.",
				},
				[]string{"code", "method"},
			),
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
