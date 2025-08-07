// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package healthz provides a comprehensive HTTP health check system for the CloudZero agent
// that enables monitoring and operational visibility of agent component health.
//
// This package implements a singleton-based health check registry that allows different
// components of the agent to register their health validation functions and expose
// a unified HTTP endpoint for health status reporting.
//
// Key features:
//   - Singleton design: Global health check registry accessible throughout the application
//   - Extensible registration: Components can register named health check functions
//   - HTTP endpoint: Standard /healthz endpoint for monitoring systems integration
//   - Fast-fail behavior: Returns error immediately if any check fails
//   - Thread-safe operations: Concurrent access to health checks is protected
//
// Health check patterns:
//   - Component readiness: Database connectivity, external service availability
//   - Resource validation: Disk space, memory usage, file system access
//   - Configuration validation: Required settings, credential availability
//   - Integration testing: API connectivity, permission validation
//
// Monitoring integration:
//   The health endpoint follows standard Kubernetes health check conventions:
//   - GET /healthz returns 200 OK when all checks pass
//   - Returns 500 Internal Server Error with details when checks fail
//   - Provides textual error information for debugging
//
// Usage patterns:
//   // Component registration during initialization
//   healthz.Register("database", func() error {
//       return db.Ping()
//   })
//   
//   healthz.Register("storage", func() error {
//       return checkDiskSpace()
//   })
//   
//   // HTTP server integration
//   http.HandleFunc("/healthz", healthz.NewHealthz().EndpointHandler())
//
// Production considerations:
//   - Health checks should be fast and lightweight
//   - Avoid expensive operations that could timeout
//   - Consider circuit breaker patterns for external dependencies
//   - Use specific, actionable error messages for operational teams
package healthz

import (
	"net/http"
	"sync"
)

// HealthCheck defines a function type for individual health validation checks.
//
// Health check functions should:
//   - Execute quickly (< 1 second recommended)
//   - Return nil for healthy state
//   - Return descriptive error for unhealthy state
//   - Be side-effect free (no state modifications)
//   - Handle their own timeouts and resource management
//
// Examples:
//   func databaseHealthCheck() error {
//       ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
//       defer cancel()
//       return db.PingContext(ctx)
//   }
type HealthCheck func() error

// HealthChecker defines the interface for health check management and HTTP endpoint provision.
//
// This interface abstracts the health check system to enable testing and different
// implementations while providing a standard way to integrate with HTTP servers.
type HealthChecker interface {
	// EndpointHandler returns an HTTP handler function that executes all registered
	// health checks and returns appropriate HTTP status codes and response bodies.
	//
	// Response behavior:
	//   - 200 OK: All health checks passed, body contains "ok"
	//   - 500 Internal Server Error: At least one check failed, body contains error details
	//
	// The handler executes all registered checks and fails fast on the first error.
	EndpointHandler() http.HandlerFunc
}

// Register adds a named health check function to the global health check registry.
//
// This function provides a convenient way to register health checks from any part
// of the application. The check will be executed whenever the health endpoint is accessed.
//
// Parameters:
//   - name: Unique identifier for the health check (used in error messages)
//   - fn: Health check function that returns nil for healthy, error for unhealthy
//
// Thread safety:
//   This function is thread-safe and can be called concurrently from multiple goroutines.
//
// Usage:
//   healthz.Register("collector", func() error {
//       if !collector.IsRunning() {
//           return errors.New("collector is not running")
//       }
//       return nil
//   })
func Register(name string, fn HealthCheck) {
	// get the interface and cast to internal type
	chkr, success := NewHealthz().(*checker)
	if !success {
		panic("unexpected type mismatch")
	}
	chkr.add(name, fn)
}

var (
	// h is the global singleton health checker instance.
	// Protected by sync.Once to ensure thread-safe initialization.
	h    *checker
	// once ensures the singleton health checker is initialized exactly once,
	// even when accessed concurrently from multiple goroutines.
	once sync.Once
)

// checker implements the HealthChecker interface and manages a registry of named health checks.
//
// This internal type provides the concrete implementation of health check management
// with thread-safe access to the check registry.
type checker struct {
	// mu protects concurrent access to the checks map
	mu     sync.Mutex
	// checks maps health check names to their corresponding check functions
	checks map[string]HealthCheck
}

// NewHealthz returns the singleton HealthChecker instance.
//
// This function implements the singleton pattern to ensure there is exactly one
// health checker instance throughout the application lifecycle. The instance is
// created on first access and reused for subsequent calls.
//
// Returns:
//   - HealthChecker: The global singleton health checker instance
//
// Thread safety:
//   This function is thread-safe and can be called concurrently.
func NewHealthz() HealthChecker {
	once.Do(func() {
		h = &checker{}
	})
	return h
}

// add registers a named health check function in the checker's registry.
//
// This internal method handles the thread-safe addition of health checks to
// the registry, initializing the map if necessary.
//
// Parameters:
//   - name: Unique identifier for the health check
//   - fn: Health check function to register
func (x *checker) add(name string, fn HealthCheck) {
	// lock and unlock on return
	x.mu.Lock()
	defer x.mu.Unlock()
	if x.checks == nil {
		x.checks = make(map[string]HealthCheck)
	}
	x.checks[name] = fn
}

// EndpointHandler returns an HTTP handler that executes all registered health checks.
//
// The handler implements a fail-fast approach where the first failing health check
// immediately returns an error response. If all checks pass, it returns a success response.
//
// HTTP Response Details:
//   - Success (200 OK): All checks passed, response body: "ok"
//   - Failure (500 Internal Server Error): At least one check failed,
//     response body: "<check-name> failed: <error-message>"
//
// The handler is safe for concurrent access and will execute all checks
// on each request without caching results.
//
// Returns:
//   - http.HandlerFunc: HTTP handler ready for use with HTTP servers
func (x *checker) EndpointHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		for name, check := range x.checks {
			if err := check(); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(name + " failed: " + err.Error()))
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok")) // ignore return values
	}
}
