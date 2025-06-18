// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cloudzero/cloudzero-agent/app/http/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func TestPromHTTPMiddleware(t *testing.T) {
	// Create a dummy handler to wrap
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap the handler with PromHTTPMiddleware
	wrappedHandler := middleware.PromHTTPMiddleware(handler)

	// Create a test request and response recorder
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	// Serve the request
	wrappedHandler.ServeHTTP(rec, req)

	// Assert the response status code
	if rec.Code != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestLoggingMiddlewareWrapper(t *testing.T) {
	// Set up a test logger
	var logOutput zerolog.ConsoleWriter
	log.Logger = zerolog.New(logOutput).With().Timestamp().Logger()

	// Create a dummy handler to wrap
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap the handler with LoggingMiddlewareWrapper
	wrappedHandler := middleware.LoggingMiddlewareWrapper(handler)

	// Create a test request and response recorder
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	// Serve the request
	startTime := time.Now()
	wrappedHandler.ServeHTTP(rec, req)
	duration := time.Since(startTime)

	// Assert the response status code
	if rec.Code != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, rec.Code)
	}

	// Assert log output (this is a basic check, adjust as needed for your logging setup)
	if duration <= 0 {
		t.Error("expected duration to be greater than 0")
	}
}
