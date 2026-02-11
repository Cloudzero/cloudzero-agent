// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package http provides HTTP client utilities for CloudZero Agent external API integrations.
package http

import (
	"net"
	"net/http"
	"syscall"

	"github.com/pkg/errors"
)

// HTTP error variables provide typed error representations for all standard HTTP status codes.
// These errors enable CloudZero Agent components to handle HTTP failures with appropriate
// business logic, retry strategies, and error reporting patterns.
//
// Error categorization supports:
//   - Client errors (4xx): Request issues requiring caller correction
//   - Server errors (5xx): Remote service issues enabling retry logic
//   - Authentication errors: Credential and authorization failures
//   - Rate limiting errors: Request throttling requiring backoff
//
// These typed errors enable consistent error handling across CloudZero Agent HTTP integrations
// while providing clear error classification for monitoring and alerting systems.
var (
	ErrStatusBadRequest                    = errors.New("badrequest error")
	ErrStatusUnauthorized                  = errors.New("unauthorized error")
	ErrStatusPaymentRequired               = errors.New("paymentrequired error")
	ErrStatusForbidden                     = errors.New("forbidden error")
	ErrStatusNotFound                      = errors.New("not found error")
	ErrStatusMethodNotAllowed              = errors.New("method not allowed error")
	ErrStatusNotAcceptable                 = errors.New("not acceptable error")
	ErrStatusProxyAuthRequired             = errors.New("proxy authrequired error")
	ErrStatusRequestTimeout                = errors.New("request timeout error")
	ErrStatusConflict                      = errors.New("conflict error")
	ErrStatusGone                          = errors.New("gone error")
	ErrStatusLengthRequired                = errors.New("length required error")
	ErrStatusPreconditionFailed            = errors.New("precondition failed error")
	ErrStatusRequestEntityTooLarge         = errors.New("requestentity too large error")
	ErrStatusRequestURITooLong             = errors.New("request uri too long error")
	ErrStatusUnsupportedMediaType          = errors.New("unsupported media type error")
	ErrStatusRequestedRangeNotSatisfiable  = errors.New("requested range not satisfiable error")
	ErrStatusExpectationFailed             = errors.New("expectation failed error")
	ErrStatusTeapot                        = errors.New("teapot error")
	ErrStatusMisdirectedRequest            = errors.New("misdirected request error")
	ErrStatusUnprocessableEntity           = errors.New("unprocessable entity error")
	ErrStatusLocked                        = errors.New("locked error")
	ErrStatusFailedDependency              = errors.New("failed dependency error")
	ErrStatusTooEarly                      = errors.New("too early error")
	ErrStatusUpgradeRequired               = errors.New("upgrade required error")
	ErrStatusPreconditionRequired          = errors.New("precondition required error")
	ErrStatusTooManyRequests               = errors.New("too many requests error")
	ErrStatusRequestHeaderFieldsTooLarge   = errors.New("request header fields too large error")
	ErrStatusInternalServerError           = errors.New("internal server error")
	ErrStatusNotImplemented                = errors.New("not implemented error")
	ErrStatusBadGateway                    = errors.New("bad gateway error")
	ErrStatusServiceUnavailable            = errors.New("service unavailable error")
	ErrStatusGatewayTimeout                = errors.New("gateway timeout error")
	ErrStatusHTTPVersionNotSupported       = errors.New("http version not supported error")
	ErrStatusVariantAlsoNegotiates         = errors.New("variant also negotiates error")
	ErrStatusInsufficientStorage           = errors.New("insufficient storage error")
	ErrStatusLoopDetected                  = errors.New("loop detected error")
	ErrStatusNotExtended                   = errors.New("not extended error")
	ErrStatusNetworkAuthenticationRequired = errors.New("network authentication required error")
)

// classifyNetworkError analyzes network-level errors to provide specific error classifications.
// This function examines error chains and underlying causes to determine the specific type
// of network failure, enabling appropriate retry logic and error reporting.
//
// Error classification categories:
//   - DNS resolution failures: "name not found" for hostname lookup errors
//   - Connection failures: "connection refused" for service unavailability
//   - Timeout errors: "timeout" for deadline exceeded conditions
//   - Unknown errors: Empty string for unclassified network issues
//
// Error unwrapping:
//
//	The function iterates through error chains using the Unwrap pattern (Go 1.13+)
//	to examine underlying causes and provide accurate classification even for
//	wrapped errors from various network libraries.
//
// This classification enables CloudZero Agent to implement appropriate retry strategies:
//   - DNS errors may require longer backoff periods
//   - Connection refused errors may indicate service outages
//   - Timeout errors may require request optimization
//
// Returns a human-readable classification string for the network error,
// or empty string if the error cannot be classified.
func classifyNetworkError(err error) string {
	cause := err
	for {
		// Unwrap was added in Go 1.13.
		// See https://github.com/golang/go/issues/36781
		if unwrap, ok := cause.(interface{ Unwrap() error }); ok {
			cause = unwrap.Unwrap()
			continue
		}
		break
	}

	// DNSError.IsNotFound was added in Go 1.13.
	// See https://github.com/golang/go/issues/28635
	if cause, ok := cause.(*net.DNSError); ok && cause.Err == "no such host" {
		return "name not found"
	}

	if cause, ok := cause.(syscall.Errno); ok {
		if cause == 10061 || cause == syscall.ECONNREFUSED { //nolint:revive // The source for 10061 is unknown
			return "connection refused"
		}
	}

	if cause, ok := cause.(net.Error); ok && cause.Timeout() {
		return "timeout"
	}

	return ""
}

// ToError converts HTTP status codes to typed error values for consistent error handling.
// This function maps standard HTTP status codes to corresponding error types, enabling
// CloudZero Agent components to handle HTTP responses with appropriate business logic.
//
// Error mapping approach:
//   - Client errors (4xx): Map to specific client-side error types
//   - Server errors (5xx): Map to server-side error types
//   - Success codes (2xx): Return nil (no error)
//   - Informational (1xx): Return nil (no error)
//   - Redirection (3xx): Return nil (no error, handled by HTTP client)
//
// Usage patterns:
//   - API integration: Determine retry strategies based on error type
//   - Authentication: Identify credential and authorization failures
//   - Rate limiting: Detect throttling conditions requiring backoff
//   - Service health: Monitor external service availability and performance
//
// This conversion enables CloudZero Agent to implement sophisticated error handling
// strategies that distinguish between different types of HTTP failures and respond
// appropriately with retry logic, alerting, and graceful degradation.
//
// Returns the corresponding error type for the status code, or nil for success codes.
func ToError(code int) error {
	switch code {
	case http.StatusBadRequest:
		return ErrStatusBadRequest
	case http.StatusUnauthorized:
		return ErrStatusUnauthorized
	case http.StatusPaymentRequired:
		return ErrStatusPaymentRequired
	case http.StatusForbidden:
		return ErrStatusForbidden
	case http.StatusNotFound:
		return ErrStatusNotFound
	case http.StatusMethodNotAllowed:
		return ErrStatusMethodNotAllowed
	case http.StatusNotAcceptable:
		return ErrStatusNotAcceptable
	case http.StatusProxyAuthRequired:
		return ErrStatusProxyAuthRequired
	case http.StatusRequestTimeout:
		return ErrStatusRequestTimeout
	case http.StatusConflict:
		return ErrStatusConflict
	case http.StatusGone:
		return ErrStatusGone
	case http.StatusLengthRequired:
		return ErrStatusLengthRequired
	case http.StatusPreconditionFailed:
		return ErrStatusPreconditionFailed
	case http.StatusRequestEntityTooLarge:
		return ErrStatusRequestEntityTooLarge
	case http.StatusRequestURITooLong:
		return ErrStatusRequestURITooLong
	case http.StatusUnsupportedMediaType:
		return ErrStatusUnsupportedMediaType
	case http.StatusRequestedRangeNotSatisfiable:
		return ErrStatusRequestedRangeNotSatisfiable
	case http.StatusExpectationFailed:
		return ErrStatusExpectationFailed
	case http.StatusTeapot:
		return ErrStatusTeapot
	case http.StatusMisdirectedRequest:
		return ErrStatusMisdirectedRequest
	case http.StatusUnprocessableEntity:
		return ErrStatusUnprocessableEntity
	case http.StatusLocked:
		return ErrStatusLocked
	case http.StatusFailedDependency:
		return ErrStatusFailedDependency
	case http.StatusTooEarly:
		return ErrStatusTooEarly
	case http.StatusUpgradeRequired:
		return ErrStatusUpgradeRequired
	case http.StatusPreconditionRequired:
		return ErrStatusPreconditionRequired
	case http.StatusTooManyRequests:
		return ErrStatusTooManyRequests
	case http.StatusRequestHeaderFieldsTooLarge:
		return ErrStatusRequestHeaderFieldsTooLarge
	case http.StatusInternalServerError:
		return ErrStatusInternalServerError
	case http.StatusNotImplemented:
		return ErrStatusNotImplemented
	case http.StatusBadGateway:
		return ErrStatusBadGateway
	case http.StatusServiceUnavailable:
		return ErrStatusServiceUnavailable
	case http.StatusGatewayTimeout:
		return ErrStatusGatewayTimeout
	case http.StatusHTTPVersionNotSupported:
		return ErrStatusHTTPVersionNotSupported
	case http.StatusVariantAlsoNegotiates:
		return ErrStatusVariantAlsoNegotiates
	case http.StatusInsufficientStorage:
		return ErrStatusInsufficientStorage
	case http.StatusLoopDetected:
		return ErrStatusLoopDetected
	case http.StatusNotExtended:
		return ErrStatusNotExtended
	case http.StatusNetworkAuthenticationRequired:
		return ErrStatusNetworkAuthenticationRequired
	}
	return nil
}
