// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package handlers provides HTTP request handlers for CloudZero Agent Primary Adapter implementations.
package handlers

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-obvious/server"
	"github.com/go-obvious/server/api"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PromMetricsAPI provides HTTP endpoints for CloudZero Agent Prometheus metrics collection.
// This API serves as a Primary Adapter in the hexagonal architecture, exposing internal agent
// metrics through the standard Prometheus HTTP interface for monitoring and alerting systems.
//
// The PromMetricsAPI enables external monitoring systems to collect operational metrics from
// CloudZero Agent components, providing visibility into performance, health, and business metrics
// essential for production operations and CloudZero cost optimization insights.
//
// Key capabilities:
//   - Prometheus metrics exposition: Standard /metrics endpoint for automated scraping
//   - Internal agent metrics: Performance counters, error rates, and operational health
//   - Business metrics: Cost allocation processing rates, webhook admission statistics
//   - Infrastructure metrics: Resource usage, connection health, and storage operations
//
// Metrics categories exposed:
//   - HTTP request metrics: Request rates, response times, error rates by endpoint
//   - Webhook processing: Admission request volumes, processing latencies, failure rates
//   - Metric shipping: Upload rates, batch sizes, CloudZero API response times
//   - Storage operations: Database query performance, storage utilization, error rates
//   - Resource processing: Kubernetes resource admission rates by type and operation
//
// Integration patterns:
//   - Prometheus scraping: Automated metric collection every 15-30 seconds
//   - Grafana dashboards: Visual monitoring and alerting for operational teams
//   - Alert Manager: Automated incident response based on metric thresholds
//   - CloudZero platform: Business metrics integration for cost optimization insights
//
// The API provides a thin integration layer around the standard Prometheus HTTP handler,
// maintaining compatibility with Prometheus ecosystem tooling while integrating with
// CloudZero Agent HTTP server infrastructure and operational patterns.
type PromMetricsAPI struct {
	// api.Service provides the foundational HTTP server infrastructure from go-obvious/server.
	// This embedded service handles HTTP server lifecycle, request routing, middleware integration,
	// and provides consistent API patterns across CloudZero Agent HTTP endpoints.
	api.Service
}

// NewPromMetricsAPI creates a new HTTP API server for CloudZero Agent Prometheus metrics exposition.
// This constructor initializes all necessary components for exposing internal agent metrics
// through the standard Prometheus HTTP interface for monitoring system integration.
//
// The PromMetricsAPI provides essential operational visibility into CloudZero Agent performance,
// health, and business metrics, enabling comprehensive monitoring and alerting for production
// deployments and CloudZero cost optimization operations.
//
// Configuration parameters:
//   - base: HTTP path prefix for metrics endpoints (typically "/metrics")
//
// Standard deployment path:
//
//	The metrics API should be mounted at "/metrics" to maintain compatibility with
//	Prometheus ecosystem conventions and automated service discovery patterns.
//	This enables seamless integration with existing monitoring infrastructure.
//
// Initialization process:
//  1. Configure HTTP routing with chi router for efficient metrics request handling
//  2. Set up Prometheus HTTP handler integration with standard metrics exposition format
//  3. Register metrics endpoints with go-obvious server infrastructure
//  4. Enable structured metrics access patterns for monitoring systems
//
// Metrics collection:
//
//	The API exposes all metrics registered with the default Prometheus registry,
//	including both standard Go runtime metrics and CloudZero-specific business metrics.
//	This provides comprehensive observability into agent operations.
//
// The returned PromMetricsAPI can be registered with the CloudZero Agent HTTP server
// to begin serving Prometheus metrics endpoints for monitoring system integration.
func NewPromMetricsAPI(base string) *PromMetricsAPI {
	a := &PromMetricsAPI{
		Service: api.Service{
			APIName: "metrics",
			Mounts:  map[string]*chi.Mux{},
		},
	}
	a.Mounts[base] = a.Routes()
	return a
}

// Register integrates the PromMetricsAPI with the CloudZero Agent HTTP server infrastructure.
// This method completes the metrics API setup by mounting the Prometheus endpoints and enabling
// comprehensive monitoring capabilities for operations teams and automated systems.
//
// Registration process:
//   - Mount metrics routes at the configured base path (typically /metrics)
//   - Enable HTTP middleware for logging, security, and performance tracking
//   - Configure Prometheus HTTP handler with default registry integration
//   - Activate metrics exposition for monitoring system scraping
//
// The registration process integrates the metrics API with the agent's broader HTTP infrastructure,
// enabling coordinated startup, shutdown, and operational monitoring across all agent components.
//
// Operational integration:
//   - Prometheus scraping: Enables automated metric collection from monitoring systems
//   - Service discovery: Compatible with Kubernetes service discovery and annotation patterns
//   - Health monitoring: Provides metrics for liveness and readiness probe integration
//   - Performance tracking: Exposes operational metrics for SLA monitoring and alerting
//
// Error conditions:
//   - Service registration failures (port conflicts, permission issues)
//   - Route mounting conflicts with existing endpoints
//   - Middleware initialization failures
//
// Once registration completes successfully, the metrics API is ready to serve
// Prometheus-format metrics for comprehensive CloudZero Agent monitoring.
func (a *PromMetricsAPI) Register(app server.Server) error {
	if err := a.Service.Register(app); err != nil {
		return err
	}
	return nil
}

// Routes configures HTTP request routing for the Prometheus metrics API endpoints.
// This method creates a chi router instance with the standard Prometheus metrics endpoint
// integrated with the official Prometheus HTTP handler.
//
// Metrics endpoint configuration:
//   - GET /: Standard Prometheus metrics exposition endpoint
//   - Content-Type: text/plain; version=0.0.4; charset=utf-8
//   - Format: Standard Prometheus exposition format with metric families
//
// Metrics exposed include:
//   - Go runtime metrics: Memory usage, garbage collection, goroutines
//   - HTTP server metrics: Request rates, response times, status codes
//   - CloudZero business metrics: Webhook processing, metric shipping, cost allocation
//   - Storage metrics: Database operations, query performance, error rates
//
// The chi router provides:
//   - High-performance HTTP routing optimized for frequent metrics scraping
//   - Middleware support for access control and performance monitoring
//   - Standard HTTP patterns compatible with Prometheus ecosystem tools
//   - Integration with go-obvious server infrastructure
//
// This routing configuration enables seamless integration with Prometheus monitoring
// systems while maintaining compatibility with CloudZero Agent operational patterns
// and providing comprehensive observability into agent performance and health.
func (a *PromMetricsAPI) Routes() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/", promhttp.Handler().ServeHTTP)
	return r
}
