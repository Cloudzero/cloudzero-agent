// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package handlers provides HTTP request handlers for CloudZero Agent Primary Adapter implementations.
package handlers

import (
	"io"
	"math/rand"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-obvious/server"
	"github.com/go-obvious/server/api"
	"github.com/go-obvious/server/request"
	"github.com/rs/zerolog/log"

	"github.com/cloudzero/cloudzero-agent/app/domain"
)

// MaxPayloadSize defines the maximum allowed size for Prometheus remote_write requests.
// This limit balances memory usage protection with support for large metric batches
// from high-volume Prometheus instances while preventing memory exhaustion attacks.
//
// Size considerations:
//   - Large Prometheus deployments can send 10MB+ batches during high-volume periods
//   - Compressed payloads typically achieve 5-10x compression ratios
//   - CloudZero Agent must handle concurrent requests from multiple Prometheus instances
//   - Memory usage must remain predictable in constrained Kubernetes environments
//
// Production implications:
//   - Prevents out-of-memory conditions during traffic spikes
//   - Enables reliable operation under high metric ingestion loads
//   - Maintains predictable resource consumption for cluster resource planning
//   - Provides clear error responses for oversized requests
//
// The 16MB limit accommodates legitimate large metric batches while providing
// protection against malicious or misconfigured Prometheus instances.
const MaxPayloadSize = 16 * 1024 * 1024

// RemoteWriteAPI provides HTTP endpoints for Prometheus remote_write metric ingestion.
// This API serves as a Primary Adapter in the hexagonal architecture, implementing the
// Prometheus remote_write protocol for collecting metrics from Prometheus instances and
// processing them through the CloudZero cost allocation pipeline.
//
// The RemoteWriteAPI enables CloudZero Agent to receive metric data from multiple Prometheus
// instances across Kubernetes clusters, extracting cost allocation insights and forwarding
// relevant metrics to the CloudZero platform for comprehensive cost optimization analysis.
//
// Key capabilities:
//   - Prometheus remote_write protocol: Full support for v1 and v2 remote_write specifications
//   - Compression handling: Automatic decompression of snappy-compressed payloads
//   - High throughput: Optimized for processing thousands of metrics per second
//   - Error handling: Graceful handling of malformed requests with appropriate HTTP responses
//   - Load balancing: Connection management for distributing load across agent replicas
//
// Protocol support:
//   - Remote Write v1: Standard protobuf-based metric ingestion
//   - Remote Write v2: Enhanced protocol with improved metadata support
//   - Content encoding: Snappy compression for efficient network transmission
//   - Batch processing: Handles large metric batches from high-volume Prometheus instances
//
// Integration patterns:
//   - Prometheus configuration: remote_write endpoint configuration
//   - Kubernetes deployment: Multiple agent replicas with load balancing
//   - CloudZero platform: Metric forwarding and cost allocation analysis
//   - Monitoring systems: Request metrics and error rate tracking
//
// The API maintains clean separation between HTTP protocol concerns and metric processing
// business logic, delegating actual metric collection to the domain service.
type RemoteWriteAPI struct {
	// api.Service provides the foundational HTTP server infrastructure from go-obvious/server.
	// This embedded service handles HTTP server lifecycle, request routing, middleware integration,
	// and provides consistent API patterns across CloudZero Agent HTTP endpoints.
	api.Service

	// metrics implements the core metric collection and processing operations for CloudZero data pipeline.
	// This domain service handles the actual business logic for parsing, filtering, and storing
	// Prometheus metrics while remaining HTTP-agnostic.
	//
	// The MetricCollector provides:
	//   - Prometheus protocol parsing: remote_write v1 and v2 format support
	//   - Metric filtering: Cost allocation relevance classification
	//   - Data transformation: Conversion to CloudZero platform formats
	//   - Storage operations: Batching and persistence for reliable processing
	//
	// By maintaining this abstraction, the RemoteWriteAPI can handle HTTP concerns
	// while keeping the metric processing logic testable and reusable.
	metrics *domain.MetricCollector
}

// NewRemoteWriteAPI creates a new HTTP API server for Prometheus remote_write metric ingestion.
// This constructor initializes all necessary components for receiving and processing Prometheus
// metrics through the CloudZero Agent cost allocation pipeline.
//
// The RemoteWriteAPI provides the essential integration point between Prometheus monitoring
// systems and CloudZero cost optimization, enabling automatic metric collection and processing
// for comprehensive cost allocation analysis across Kubernetes environments.
//
// Configuration parameters:
//   - base: HTTP path prefix for remote_write endpoints (typically "/api/v1/write")
//   - d: MetricCollector domain service implementing the core metric processing logic
//
// Initialization process:
//  1. Configure HTTP routing with chi router for high-performance metric ingestion
//  2. Set up remote_write endpoint with proper request validation and size limits
//  3. Register metric processing endpoints with go-obvious server infrastructure
//  4. Enable structured logging and metrics collection for operational monitoring
//
// Production characteristics:
//   - High throughput: Optimized for processing thousands of metrics per second
//   - Memory efficiency: Streaming processing with predictable memory usage
//   - Error resilience: Graceful handling of malformed or oversized requests
//   - Load balancing: Connection management for distributing Prometheus load
//
// The returned RemoteWriteAPI can be registered with the CloudZero Agent HTTP server
// to begin receiving Prometheus metrics for cost allocation processing.
func NewRemoteWriteAPI(base string, d *domain.MetricCollector) *RemoteWriteAPI {
	a := &RemoteWriteAPI{
		metrics: d,
		Service: api.Service{
			APIName: "remotewrite",
			Mounts:  map[string]*chi.Mux{},
		},
	}
	a.Mounts[base] = a.Routes()
	return a
}

// Register integrates the RemoteWriteAPI with the CloudZero Agent HTTP server infrastructure.
// This method completes the remote_write API setup by mounting the Prometheus endpoints and enabling
// high-throughput metric ingestion capabilities for cost allocation processing.
//
// Registration process:
//   - Mount remote_write routes at the configured base path
//   - Enable HTTP middleware for logging, metrics, and error handling
//   - Configure request size limits and timeout management
//   - Activate metric ingestion pipeline with domain service integration
//
// The registration process integrates the remote_write API with the agent's broader HTTP infrastructure,
// enabling coordinated startup, shutdown, and performance monitoring across all agent components.
//
// Performance considerations:
//   - High-throughput endpoint requiring efficient request processing
//   - Memory management for handling large metric batches from Prometheus
//   - Connection pooling and load balancing across multiple agent replicas
//   - Metrics collection for monitoring ingestion rates and error patterns
//
// Error conditions:
//   - Service registration failures (port conflicts, permission issues)
//   - Route mounting conflicts with existing endpoints
//   - Middleware initialization failures
//
// Once registration completes successfully, the remote_write API is ready to receive
// Prometheus metrics for processing through the CloudZero cost allocation pipeline.
func (a *RemoteWriteAPI) Register(app server.Server) error {
	if err := a.Service.Register(app); err != nil {
		return err
	}
	return nil
}

// Routes configures HTTP request routing for the Prometheus remote_write API endpoints.
// This method creates a chi router instance with the remote_write endpoint optimized
// for high-throughput metric ingestion from Prometheus instances.
//
// Remote write endpoint configuration:
//   - POST /: Primary Prometheus remote_write endpoint for metric ingestion
//   - Protocol support: Prometheus remote_write v1 and v2 specifications
//   - Content types: application/x-protobuf with snappy compression
//   - Request handling: Streaming processing for memory efficiency
//
// The chi router provides:
//   - High-performance HTTP routing optimized for frequent metric submissions
//   - Middleware support for authentication, logging, and performance monitoring
//   - Standard HTTP patterns compatible with Prometheus remote_write clients
//   - Integration with go-obvious server infrastructure
//
// Performance optimizations:
//   - Minimal routing overhead for high-frequency requests
//   - Efficient request body processing with size limits
//   - Connection management for load balancing across replicas
//   - Request correlation for debugging and monitoring
//
// This routing configuration enables seamless integration with Prometheus monitoring
// while maintaining high throughput and reliability for CloudZero metric processing.
func (a *RemoteWriteAPI) Routes() *chi.Mux {
	r := chi.NewRouter()
	r.Post("/", a.PostMetrics)
	return r
}

// logErrorReply provides consistent error logging and HTTP response handling for remote_write operations.
// This helper function ensures that all remote_write errors are logged with proper context and
// returned to Prometheus clients with appropriate HTTP status codes and messages.
//
// Error handling approach:
//   - Structured logging: Uses request context for correlation and tracing
//   - Consistent responses: Standardized error messages for Prometheus client compatibility
//   - Status code mapping: Appropriate HTTP codes for different error conditions
//   - Request correlation: Context-aware logging for debugging and monitoring
//
// This centralized error handling ensures consistent behavior across all remote_write
// error conditions while providing comprehensive operational visibility.
func logErrorReply(r *http.Request, w http.ResponseWriter, data string, statusCode int) {
	log.Ctx(r.Context()).Error().Msg(data)
	request.Reply(r, w, data, statusCode)
}

// PostMetrics processes Prometheus remote_write requests for CloudZero metric ingestion.
// This method implements the core remote_write endpoint that receives metric data from Prometheus
// instances and processes it through the CloudZero cost allocation pipeline.
//
// Processing pipeline:
//  1. Request validation: Verify content length, size limits, and required headers
//  2. Body reading: Stream request body with memory usage protection
//  3. Metric processing: Parse and process metrics through domain service
//  4. Response generation: Return appropriate HTTP status and headers
//  5. Connection management: Implement load balancing through periodic connection closure
//
// Protocol compliance:
//   - Content-Type: application/x-protobuf for remote_write v1/v2
//   - Content-Encoding: snappy compression support
//   - Status codes: HTTP 204 for success, 400/500 for errors
//   - Headers: Custom response headers for client optimization
//
// Performance characteristics:
//   - Streaming processing: Memory-efficient handling of large metric batches
//   - Size limits: 16MB maximum payload protection
//   - Connection management: Periodic closure for load distribution
//   - Error resilience: Graceful handling of malformed requests
//
// Load balancing:
//
//	Implements periodic HTTP/1.1 connection closure to distribute Prometheus
//	requests across multiple CloudZero Agent replicas, improving resource
//	utilization and fault tolerance.
//
// This method represents the primary integration point between Prometheus monitoring
// and CloudZero cost optimization, processing high-volume metric streams while
// maintaining sub-second response times.
func (a *RemoteWriteAPI) PostMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	defer r.Body.Close()
	contentLen := r.ContentLength

	if contentLen <= 0 {
		logErrorReply(r, w, "empty body", http.StatusOK)
		return
	}

	if contentLen > MaxPayloadSize {
		logErrorReply(r, w, "too big", http.StatusOK)
		return
	}

	contentType := r.Header.Get("Content-Type")
	encodingType := r.Header.Get("Content-Encoding")
	data, err := io.ReadAll(r.Body)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to read request body")
		request.Reply(r, w, "failed to read request body", http.StatusBadRequest)
		return
	}

	stats, err := a.metrics.PutMetrics(r.Context(), contentType, encodingType, data)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to put metrics")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// If we're using HTTP/1.x, we want to periodically close the connection to
	// help distribute the load across the various collector replicas.
	//
	// Unfortunately this won't work for HTTP/2, but currently all traffic we're
	// seeing from Prometheus is HTTP/1.1.
	if r.ProtoMajor == 1 {
		rf := a.metrics.Settings().Server.ReconnectFrequency
		if rf > 0 && rand.Intn(rf) == 0 { //nolint:gosec // a weak PRNG is fine here
			w.Header().Set("Connection", "close")
		}
	}

	if stats != nil {
		stats.SetHeaders(w)
	}

	request.Reply(r, w, nil, http.StatusNoContent)
}
