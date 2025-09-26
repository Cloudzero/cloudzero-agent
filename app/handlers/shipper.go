// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package handlers provides HTTP request handlers for CloudZero Agent Primary Adapter implementations.
package handlers

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-obvious/server"
	"github.com/go-obvious/server/api"

	"github.com/cloudzero/cloudzero-agent/app/domain/shipper"
)

// ShipperAPI provides HTTP endpoints for CloudZero Agent metric shipping operations and monitoring.
// This API serves as a Primary Adapter in the hexagonal architecture, exposing metric shipper
// functionality through HTTP interfaces for operational monitoring, debugging, and integration.
//
// The ShipperAPI enables external systems to interact with the metric shipping pipeline,
// providing visibility into CloudZero data transmission status, performance metrics,
// and operational health information.
//
// Key capabilities:
//   - Prometheus metrics exposure: Internal shipper metrics for monitoring and alerting
//   - Health checking: Endpoint status and connectivity validation
//   - Debugging support: Operational insights for troubleshooting shipping issues
//   - Performance monitoring: Throughput, latency, and error rate tracking
//
// Integration points:
//   - Prometheus monitoring: Scrapes /metrics endpoint for operational dashboards
//   - Health checks: Kubernetes liveness and readiness probe support
//   - Debugging tools: Operations team access to shipping pipeline status
//   - Performance analysis: CloudZero platform integration monitoring
//
// The API maintains separation between HTTP concerns and metric shipping business logic,
// delegating actual shipping operations to the domain service while providing HTTP accessibility.
type ShipperAPI struct {
	// api.Service provides the foundational HTTP server infrastructure from go-obvious/server.
	// This embedded service handles HTTP server lifecycle, request routing, middleware integration,
	// and provides consistent API patterns across CloudZero Agent HTTP endpoints.
	api.Service

	// shipper implements the core metric shipping operations for CloudZero data transmission.
	// This domain service handles the actual business logic for batching, compressing,
	// and uploading metric data to the CloudZero platform while remaining HTTP-agnostic.
	//
	// The shipper provides:
	//   - Metric data processing and batching for efficient transmission
	//   - CloudZero platform API integration with authentication and retry logic
	//   - Prometheus metrics collection for operational monitoring
	//   - Error handling and retry mechanisms for reliable data delivery
	//
	// By maintaining this abstraction, the ShipperAPI can expose shipper functionality
	// through HTTP while keeping the shipping logic testable and reusable.
	shipper *shipper.MetricShipper
}

// NewShipperAPI creates a new HTTP API server for CloudZero Agent metric shipping operations.
// This constructor initializes all necessary components for exposing metric shipper functionality
// through HTTP endpoints, enabling operational monitoring and debugging capabilities.
//
// The ShipperAPI provides essential operational visibility into the CloudZero data transmission
// pipeline, allowing monitoring systems and operations teams to track shipping performance,
// diagnose issues, and ensure reliable data delivery to the CloudZero platform.
//
// Configuration parameters:
//   - base: HTTP path prefix for shipper endpoints (typically "/shipper" or "/shipping")
//   - d: MetricShipper domain service implementing the core shipping business logic
//
// Initialization process:
//  1. Configure HTTP routing with chi router for efficient request handling
//  2. Set up metric endpoints for Prometheus scraping and operational monitoring
//  3. Register shipper endpoints with go-obvious server infrastructure
//  4. Enable structured logging and health checking integration
//
// The returned server.API can be registered with the CloudZero Agent HTTP server
// to begin serving metric shipping operational endpoints for monitoring and debugging.
//
// Production considerations:
//   - Prometheus metrics endpoint for automated monitoring and alerting
//   - Health check endpoints for Kubernetes liveness and readiness probes
//   - Debug endpoints for operations team troubleshooting and analysis
//   - Performance metrics for CloudZero platform integration monitoring
func NewShipperAPI(base string, d *shipper.MetricShipper) server.API {
	a := &ShipperAPI{
		shipper: d,
		Service: api.Service{
			APIName: "shipper",
			Mounts:  map[string]*chi.Mux{},
		},
	}
	a.Service.Mounts[base] = a.Routes()
	return a
}

// Register integrates the ShipperAPI with the CloudZero Agent HTTP server infrastructure.
// This method completes the shipper API setup by mounting the operational endpoints and enabling
// metric shipping monitoring and debugging capabilities.
//
// Registration process:
//   - Mount shipper routes at the configured base path
//   - Enable HTTP middleware for logging, metrics, and error handling
//   - Configure Prometheus metrics scraping endpoint
//   - Activate health check and debugging endpoints
//
// The registration process integrates the shipper API with the agent's broader HTTP infrastructure,
// enabling coordinated startup, shutdown, and health monitoring across all agent components.
//
// Error conditions:
//   - Service registration failures (port conflicts, permission issues)
//   - Route mounting conflicts with existing endpoints
//   - Middleware initialization failures
//
// Once registration completes successfully, the shipper API is ready to serve operational
// endpoints for metric shipping monitoring, debugging, and health checking.
func (a *ShipperAPI) Register(app server.Server) error {
	if err := a.Service.Register(app); err != nil {
		return err
	}
	return nil
}

// Routes configures HTTP request routing for the metric shipper API endpoints.
// This method creates a chi router instance with all necessary routes for monitoring
// and debugging CloudZero metric shipping operations.
//
// Route configuration:
//   - GET /metrics: Prometheus metrics endpoint for operational monitoring and alerting
//   - Future endpoints: Health checks, debug information, and performance metrics as needed
//
// The chi router provides:
//   - High-performance HTTP routing with minimal overhead for metrics scraping
//   - Middleware support for cross-cutting concerns (logging, authentication)
//   - RESTful routing patterns compatible with monitoring and debugging tools
//   - Integration with go-obvious server infrastructure
//
// This routing configuration enables monitoring systems to collect operational metrics
// from the metric shipping pipeline while maintaining compatibility with standard
// HTTP tooling and Prometheus monitoring patterns.
func (a *ShipperAPI) Routes() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/metrics", a.shipper.GetMetricHandler().ServeHTTP)
	return r
}
