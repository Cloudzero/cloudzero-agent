// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package http provides HTTP client utilities for CloudZero Agent external API integrations.
package http

import "github.com/cloudzero/cloudzero-agent/app/build"

const (
	// HeaderAccept specifies the media types acceptable for HTTP response content.
	// CloudZero Agent uses this header to negotiate response formats with external APIs,
	// enabling support for JSON, protobuf, and other content types based on integration requirements.
	HeaderAccept = "Accept"

	// HeaderUserAgent identifies CloudZero Agent in HTTP requests to external services.
	// This header enables external services to identify CloudZero Agent requests for logging,
	// rate limiting, and compatibility purposes while providing version information for debugging.
	HeaderUserAgent = "User-Agent"

	// HeaderAuthorization carries authentication credentials for CloudZero platform and API integrations.
	// This header transmits API keys, bearer tokens, and other authentication data required
	// for secure communication with external services including the CloudZero platform.
	HeaderAuthorization = "Authorization"

	// HeaderContentEncoding specifies the encoding applied to HTTP request/response bodies.
	// CloudZero Agent uses this header to indicate compression formats (gzip, snappy)
	// for efficient data transmission with external services, particularly for metric uploads.
	HeaderContentEncoding = "Content-Encoding"

	// HeaderContentType indicates the media type of HTTP request/response body content.
	// This header enables proper serialization/deserialization of data exchanged with
	// external APIs, supporting JSON, protobuf, and other formats used by CloudZero integrations.
	HeaderContentType = "Content-Type"

	// HeaderAcceptEncoding specifies the content encodings acceptable for HTTP responses.
	// CloudZero Agent uses this header to indicate compression support (gzip, deflate)
	// to external services, enabling bandwidth optimization for large data transfers.
	HeaderAcceptEncoding = "Accept-Encoding"
)

const (
	// ContentTypeGzip identifies gzip-compressed content in HTTP communications.
	// CloudZero Agent uses this encoding type for compressing metric data uploads
	// to the CloudZero platform, reducing bandwidth usage and improving transfer efficiency.
	ContentTypeGzip = "gzip"

	// ContentTypeProtobuf specifies Protocol Buffer serialized content for binary data exchange.
	// This content type is used for Prometheus remote_write protocol communications
	// and other high-efficiency binary data transfers between CloudZero Agent and external services.
	ContentTypeProtobuf = "application/x-protobuf"

	// ContentTypeJSON indicates JSON-formatted content for structured data exchange.
	// CloudZero Agent uses JSON for CloudZero platform API communications, configuration
	// management, and human-readable data formats across various integrations.
	ContentTypeJSON = "application/json"

	// ContentTypeValueBin specifies binary octet-stream content for raw data transfers.
	// This content type supports file uploads, binary metric data, and other unstructured
	// binary content exchanged with external services.
	ContentTypeValueBin = "application/octet-stream"

	// ContentTypeValueTxt indicates plain text content for human-readable communications.
	// CloudZero Agent uses text content types for logging, debugging outputs,
	// and simple string-based API interactions.
	ContentTypeValueTxt = "text/plain"

	// ContentTypeValueYAML specifies YAML-formatted content for configuration and structured data.
	// This content type supports Kubernetes YAML resources, configuration files,
	// and human-readable structured data exchange.
	ContentTypeValueYAML = "text/yaml"

	// ContentTypeValueCSV indicates comma-separated values format for tabular data.
	// CloudZero Agent may use CSV format for data exports, reporting,
	// and structured data exchange with external analytics systems.
	ContentTypeValueCSV = "text/csv"
)

// setUserAgent configures the User-Agent header for CloudZero Agent HTTP requests.
// This function ensures consistent agent identification across all external service communications,
// providing version information and enabling external services to track CloudZero Agent usage patterns.
//
// User-Agent format: "cloudzero/<version>"
//   - Identifies requests as originating from CloudZero Agent
//   - Includes build version for compatibility tracking and debugging
//   - Follows standard User-Agent conventions for service identification
//
// The User-Agent header enables:
//   - External service logging and analytics
//   - Rate limiting and access control based on client identification
//   - Compatibility tracking across CloudZero Agent versions
//   - Debugging support for integration issues
//
// This function modifies the headers map in-place, creating it if nil,
// ensuring that all CloudZero Agent HTTP requests include proper identification.
func setUserAgent(headers map[string]string) {
	if headers == nil {
		headers = make(map[string]string)
	}
	headers["User-Agent"] = "cloudzero/" + build.GetVersion()
}
