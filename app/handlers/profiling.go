// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package handlers provides HTTP request handlers for CloudZero Agent Primary Adapter implementations.
package handlers

import (
	"net/http"
	"net/http/pprof"
	rtprof "runtime/pprof"

	"github.com/go-chi/chi/v5"
	"github.com/go-obvious/server"
	"github.com/go-obvious/server/api"
)

// ProfilingAPI provides HTTP endpoints for CloudZero Agent performance profiling and debugging.
// This API serves as a Primary Adapter in the hexagonal architecture, exposing Go pprof
// profiling capabilities through HTTP interfaces for development, debugging, and performance analysis.
//
// The ProfilingAPI enables developers and operations teams to analyze CloudZero Agent runtime
// performance, memory usage patterns, goroutine behavior, and CPU utilization characteristics
// in production environments.
//
// Key capabilities:
//   - CPU profiling: Sample-based CPU usage analysis for performance optimization
//   - Memory profiling: Heap allocation tracking and memory leak detection
//   - Goroutine profiling: Concurrency analysis and deadlock debugging
//   - Block profiling: Synchronization bottleneck identification
//   - Trace collection: Detailed execution flow analysis for complex debugging scenarios
//
// Development use cases:
//   - Performance optimization: Identify CPU and memory bottlenecks in metric processing
//   - Memory leak debugging: Track heap allocations and garbage collection patterns
//   - Concurrency analysis: Understand goroutine interactions and synchronization behavior
//   - Production debugging: Investigate performance issues in live environments
//
// Security considerations:
//   - Should only be enabled in development and controlled production environments
//   - Provides detailed runtime information that could be sensitive
//   - Typically deployed behind authentication and network access controls
//   - CPU profiling may impact performance during profile collection
//
// The API follows Go standard pprof conventions, making it compatible with standard
// profiling tools like "go tool pprof" and web-based profile analysis interfaces.
type ProfilingAPI struct {
	// api.Service provides the foundational HTTP server infrastructure from go-obvious/server.
	// This embedded service handles HTTP server lifecycle, request routing, middleware integration,
	// and provides consistent API patterns across CloudZero Agent HTTP endpoints.
	api.Service
}

// NewProfilingAPI creates a new HTTP API server for CloudZero Agent performance profiling and debugging.
// This constructor initializes all necessary components for exposing Go runtime profiling capabilities
// through HTTP endpoints, enabling development teams to analyze agent performance characteristics.
//
// The ProfilingAPI provides essential development and debugging capabilities for CloudZero Agent,
// allowing performance analysis, memory usage tracking, and concurrency debugging in both
// development and controlled production environments.
//
// Configuration parameters:
//   - base: HTTP path prefix for profiling endpoints (typically "/debug/pprof")
//
// Standard deployment path:
//
//	The profiling API should be mounted at "/debug/pprof" to maintain compatibility
//	with Go ecosystem tooling and established pprof conventions. This enables direct
//	integration with "go tool pprof" and other profiling analysis tools.
//
// Initialization process:
//  1. Configure HTTP routing with chi router for efficient profiling request handling
//  2. Set up all standard pprof endpoints with proper Go runtime integration
//  3. Register dynamic profile handlers for all available runtime profiles
//  4. Enable structured profiling endpoint access patterns
//
// Production considerations:
//   - Enable only in development and controlled production environments
//   - Consider network access restrictions and authentication requirements
//   - Monitor performance impact during active profiling sessions
//   - Implement proper access logging for security audit requirements
//
// The returned ProfilingAPI can be registered with the CloudZero Agent HTTP server
// to begin serving performance profiling endpoints for development and debugging.
func NewProfilingAPI(base string) *ProfilingAPI {
	a := &ProfilingAPI{
		Service: api.Service{
			APIName: "profiling",
			Mounts:  map[string]*chi.Mux{},
		},
	}
	a.Mounts[base] = a.Routes()
	return a
}

// Register integrates the ProfilingAPI with the CloudZero Agent HTTP server infrastructure.
// This method completes the profiling API setup by mounting the debugging endpoints and enabling
// performance analysis capabilities for development and operations teams.
//
// Registration process:
//   - Mount profiling routes at the configured base path (typically /debug/pprof)
//   - Enable HTTP middleware for logging, security, and access control
//   - Configure all standard Go pprof endpoints with runtime integration
//   - Activate dynamic profile handlers for comprehensive profiling support
//
// The registration process integrates the profiling API with the agent's broader HTTP infrastructure,
// enabling coordinated startup, shutdown, and access control across all agent components.
//
// Security considerations:
//   - Profiling endpoints provide detailed runtime information
//   - Consider authentication and network access restrictions
//   - Enable proper access logging for security audit requirements
//   - Monitor for potential performance impact during profiling
//
// Error conditions:
//   - Service registration failures (port conflicts, permission issues)
//   - Route mounting conflicts with existing endpoints
//   - Middleware initialization failures
//
// Once registration completes successfully, the profiling API is ready to serve
// performance analysis endpoints for development and debugging activities.
func (a *ProfilingAPI) Register(app server.Server) error {
	if err := a.Service.Register(app); err != nil {
		return err
	}
	return nil
}

// Routes configures HTTP request routing for the performance profiling API endpoints.
// This method creates a chi router instance with all standard Go pprof endpoints plus
// dynamic handlers for runtime-available profiles.
//
// Standard pprof endpoints:
//   - GET /: Profile index page with links to all available profiles
//   - GET /cmdline: Command line arguments used to start the agent
//   - GET /profile: CPU profile collection (30-second default sampling)
//   - GET /symbol: Symbol table lookup for profile analysis
//   - GET /trace: Execution trace collection for detailed flow analysis
//
// Dynamic profile endpoints:
//
//	Automatically registers handlers for all runtime-available profiles including:
//	- /heap: Memory heap allocation profiles
//	- /goroutine: Active goroutine stack traces
//	- /allocs: All memory allocation profiles since startup
//	- /block: Blocking synchronization profiles
//	- /mutex: Mutex contention profiles
//	- /threadcreate: Thread creation profiles
//
// The chi router provides:
//   - High-performance HTTP routing optimized for profiling request patterns
//   - Middleware support for access control and audit logging
//   - RESTful routing compatible with standard Go profiling tools
//   - Integration with go-obvious server infrastructure
//
// This routing configuration enables seamless integration with "go tool pprof"
// and other Go ecosystem profiling tools while maintaining compatibility with
// CloudZero Agent operational patterns.
func (a *ProfilingAPI) Routes() *chi.Mux {
	r := chi.NewRouter()

	r.Get("/", pprof.Index)
	r.Get("/cmdline", pprof.Cmdline)
	r.Get("/profile", pprof.Profile)
	r.Get("/symbol", pprof.Symbol)
	r.Get("/trace", pprof.Trace)

	for _, profile := range rtprof.Profiles() {
		r.Get("/"+profile.Name(), func(w http.ResponseWriter, r *http.Request) {
			pprof.Handler(profile.Name()).ServeHTTP(w, r)
		})
	}

	return r
}
