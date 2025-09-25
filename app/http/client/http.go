// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package http provides HTTP client utilities for CloudZero Agent external API integrations.
// This package implements reusable HTTP communication patterns used across CloudZero Agent
// for integrating with external services including the CloudZero platform, Kubernetes APIs,
// and other monitoring systems.
//
// The HTTP client utilities provide consistent error handling, timeout management, and request
// configuration patterns that ensure reliable communication with external dependencies while
// maintaining observability and operational resilience.
//
// Key capabilities:
//   - Request configuration: Headers, query parameters, and body handling
//   - Timeout management: Configurable timeouts with context cancellation support
//   - Error classification: Network error categorization for retry and alerting logic
//   - User agent management: Consistent CloudZero Agent identification
//   - Response handling: Status code processing and error mapping
//
// Integration patterns:
//   - CloudZero API: Metric upload and authentication requests
//   - Kubernetes API: Custom resource and admission webhook calls
//   - Monitoring systems: Health checks and metric collection
//   - Service discovery: Dynamic endpoint resolution and validation
//
// The client utilities maintain separation between HTTP transport concerns and business logic,
// enabling reuse across different CloudZero Agent components while providing consistent
// operational behavior and error handling patterns.
package http

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/sirupsen/logrus"
)

const (
	// connectTimeout defines the maximum duration for individual HTTP requests made by CloudZero Agent.
	// This timeout balances external service responsiveness requirements with operational reliability,
	// ensuring that HTTP calls complete within reasonable time bounds while avoiding request timeouts
	// that could disrupt CloudZero Agent operations.
	//
	// Timeout considerations:
	//   - CloudZero platform APIs typically respond within 2-5 seconds
	//   - Kubernetes API calls usually complete in 1-3 seconds
	//   - Network latency may add 100-500ms depending on deployment location
	//   - Buffer for temporary service slowdowns and retry operations
	//
	// Production implications:
	//   - Prevents hanging requests from blocking agent operations
	//   - Enables timely error detection and retry logic
	//   - Maintains predictable resource usage patterns
	//   - Supports operational SLA requirements for external integrations
	//
	// The 15-second timeout provides sufficient time for legitimate requests while preventing
	// resource exhaustion from slow or unresponsive external services.
	connectTimeout = 15 * time.Second
)

// Do executes HTTP requests with comprehensive configuration and error handling for CloudZero Agent integrations.
// This function provides a standardized HTTP client interface that handles common patterns across
// CloudZero Agent external service communications while maintaining observability and reliability.
//
// Request configuration:
//   - Method: HTTP verb (GET, POST, PUT, DELETE) for the request
//   - Headers: Custom headers including authentication and content type
//   - Query parameters: URL parameters with proper encoding
//   - Body: Request payload with streaming support
//   - Context: Request context for cancellation and timeout management
//
// Operational features:
//   - Timeout management: Automatic 15-second timeout with context cancellation
//   - User agent setting: Consistent CloudZero Agent identification
//   - Error classification: Network error categorization for retry logic
//   - Response processing: Status code extraction and error mapping
//   - Request logging: Structured logging for debugging and monitoring
//
// Error handling:
//   - Network errors: Classified and wrapped with context information
//   - HTTP errors: Status code-based error mapping
//   - Timeout errors: Context cancellation and deadline exceeded handling
//   - Connection errors: DNS, connection refused, and transport failures
//
// This function serves as the foundation for CloudZero Agent HTTP communications,
// ensuring consistent behavior across all external service integrations while
// providing comprehensive error handling and operational visibility.
//
// Returns the HTTP status code and any error encountered during request processing.
func Do(
	ctx context.Context,
	client *http.Client,
	method string,
	headers map[string]string,
	queryParams map[string]string,
	uri string,
	body io.Reader,
) (int, error) {
	// Use default HTTP client if none provided, enabling caller flexibility
	// while maintaining consistent timeout and connection pooling behavior
	if client == nil {
		client = http.DefaultClient
	}

	// Implement request-level timeout to prevent hanging operations that could
	// block CloudZero Agent processing or exhaust connection pools
	ctx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	// Create HTTP request with context for proper cancellation handling
	req, err := http.NewRequestWithContext(ctx, method, uri, body)
	if err != nil {
		return http.StatusInternalServerError, errors.Wrap(err, "create request")
	}

	// Configure request headers with caller-specified values
	// Headers may include authentication tokens, content types, and custom metadata
	for header, value := range headers {
		req.Header.Set(header, value)
	}
	// Ensure consistent User-Agent identification for CloudZero Agent requests
	setUserAgent(headers)

	// Configure URL query parameters with proper encoding
	// This ensures parameter values are correctly escaped for HTTP transmission
	values := req.URL.Query()
	for key, value := range queryParams {
		values.Add(key, value)
	}
	// Apply proper HTTP encoding to prevent URL parsing issues
	req.URL.RawQuery = values.Encode()

	// Execute HTTP request with configured client and timeout context
	resp, err := client.Do(req)
	if err != nil {
		// Log request failure with context for debugging and monitoring
		log.Ctx(ctx).Err(err).Msg("HTTP request execution failed")
	}

	// Handle nil response conditions which indicate network or transport errors
	if resp == nil {
		// Attempt to classify network errors for better error handling and retry logic
		if msg := classifyNetworkError(err); msg != "" {
			// Log classified network error with additional context
			logrus.WithError(err).WithField("message", msg).Error("network error")
			return http.StatusInternalServerError, errors.Wrap(err, "network error: "+req.URL.String())
		}
		// Log generic request failure when classification is not available
		logrus.WithError(err).Error("Failed to make request")
		return http.StatusInternalServerError, err
	}
	// Return HTTP status code and convert to appropriate error type
	// This enables callers to distinguish between different HTTP error conditions
	return resp.StatusCode, ToError(resp.StatusCode)
}
