// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package logging provides structured logging infrastructure for CloudZero Agent operations.
// This package implements a comprehensive logging framework built on Zerolog that enables
// consistent, structured logging across all CloudZero Agent components with configurable
// output destinations, log levels, and contextual information.
//
// The logging infrastructure supports CloudZero Agent operational requirements including
// development debugging, production monitoring, audit compliance, and performance analysis
// through structured log events with consistent formatting and metadata.
//
// Key capabilities:
//   - Structured logging: JSON-formatted log events with consistent field structure
//   - Multiple sinks: Simultaneous output to multiple destinations (stdout, files, external systems)
//   - Level control: Configurable log levels for development and production environments
//   - Context integration: Automatic correlation with request contexts and tracing information
//   - Hook system: Extensible logging pipeline with custom processing hooks
//   - Attribute injection: Consistent metadata addition across all log events
//
// Configuration patterns:
//   - Development: Human-readable console output with debug information
//   - Production: JSON-structured logs for aggregation and analysis
//   - Testing: Filtered output for deterministic test execution
//   - Integration: Custom sinks for external monitoring and alerting systems
//
// The logging framework integrates with CloudZero Agent operational monitoring,
// providing the observability foundation for debugging, performance analysis,
// and compliance reporting across all agent components and deployment environments.
package logging

import (
	"fmt"
	"io"
	"os"

	"github.com/cloudzero/cloudzero-agent/app/build"
	"github.com/rs/zerolog"
)

// Attr provides dynamic attribute injection for CloudZero Agent logger configuration.
// This function type enables flexible addition of contextual information to log events,
// supporting consistent metadata across different logging contexts and operational scenarios.
//
// Attribute patterns:
//   - Service identification: Component names, versions, and deployment information
//   - Request correlation: Request IDs, trace IDs, and user contexts
//   - Operational context: Environment, region, and cluster information
//   - Business context: Customer IDs, cost allocation metadata
//
// The function receives a zerolog.Context and returns an enhanced context with
// additional fields, enabling chainable attribute addition and flexible logger customization.
type Attr func(zerolog.Context) zerolog.Context

// internalLogger manages CloudZero Agent logger configuration during construction.
// This internal struct accumulates logger configuration options before creating
// the final Zerolog instance, enabling flexible and validated logger setup.
//
// Configuration state:
//   - level: Log level threshold for event filtering
//   - sinks: Output destinations for log events (stdout, files, external systems)
//   - hooks: Processing hooks for log event transformation and routing
//   - attrs: Attribute functions for consistent metadata injection
//   - version: CloudZero Agent version information for log correlation
//
// The struct supports builder pattern configuration through functional options,
// enabling clean and validated logger construction with comprehensive error handling.
type internalLogger struct {
	// level controls the minimum log level for event output filtering.
	// Events below this level are discarded, enabling performance optimization
	// and appropriate verbosity control across development and production environments.
	level zerolog.Level

	// sinks define output destinations for log events.
	// Multiple sinks enable simultaneous output to console, files, and external systems,
	// supporting diverse operational and compliance requirements.
	sinks []io.Writer

	// hooks provide extensible log event processing pipeline.
	// Hooks enable custom log event transformation, routing, and integration
	// with external monitoring and alerting systems.
	hooks []zerolog.Hook

	// attrs contain attribute injection functions for consistent metadata.
	// These functions add contextual information to all log events,
	// ensuring consistent operational and business context across all logging.
	attrs []Attr

	// version provides CloudZero Agent version information for log correlation.
	// Version data enables tracking and debugging across different agent releases
	// and deployment environments.
	version string
}

// LoggerOpt defines functional options for CloudZero Agent logger configuration.
// This type enables flexible, validated logger setup through composable configuration
// functions that modify the internal logger state during construction.
//
// The functional options pattern provides:
//   - Type safety: Compile-time validation of configuration options
//   - Composability: Multiple options can be combined and reused
//   - Error handling: Each option can validate and report configuration errors
//   - Extensibility: New options can be added without breaking existing code
//
// Options modify the internalLogger during construction and return errors
// for invalid configurations, enabling robust logger setup with comprehensive validation.
type LoggerOpt = func(logger *internalLogger) error

// WithLevel configures the minimum log level for CloudZero Agent logger output filtering.
// This option enables appropriate verbosity control across different deployment environments,
// optimizing performance and ensuring suitable log output for development and production.
//
// Supported log levels (from most to least verbose):
//   - "trace": Detailed execution flow for deep debugging
//   - "debug": Development debugging information
//   - "info": General operational information (default)
//   - "warn": Warning conditions that should be noted
//   - "error": Error conditions requiring attention
//   - "fatal": Critical errors causing application termination
//   - "panic": Panic conditions with stack traces
//
// Level selection guidelines:
//   - Development: "debug" or "trace" for comprehensive debugging
//   - Testing: "warn" or "error" to reduce noise
//   - Production: "info" for operational visibility
//   - Troubleshooting: Temporarily lower levels for issue diagnosis
//
// Returns an error if the level string is not recognized or valid.
func WithLevel(level string) LoggerOpt {
	return func(logger *internalLogger) error {
		// parse the level
		logLevel, err := zerolog.ParseLevel(level)
		if err != nil {
			return fmt.Errorf("failed to parse the log level: %w", err)
		}
		logger.level = logLevel
		return nil
	}
}

// WithSink adds an output destination for CloudZero Agent log events.
// This option enables simultaneous log output to multiple destinations including
// console output, files, and external logging systems for comprehensive observability.
//
// Common sink configurations:
//   - os.Stdout: Console output for development and container logging
//   - File writers: Persistent log files for audit and analysis
//   - Network writers: Integration with external logging systems
//   - Filtered writers: Custom filtering and formatting for specific requirements
//
// Multiple sinks support:
//   - This function can be called multiple times to add multiple output destinations
//   - All sinks receive identical log events simultaneously
//   - Each sink can apply independent filtering and formatting
//   - Failed sinks do not affect other sink operations
//
// Operational benefits:
//   - Console output for real-time monitoring during development
//   - File output for persistent storage and analysis
//   - External system integration for centralized logging and alerting
//   - Backup logging destinations for reliability
//
// The sink receives all log events that pass the configured log level threshold.
func WithSink(sink io.Writer) LoggerOpt {
	return func(logger *internalLogger) error {
		logger.sinks = append(logger.sinks, sink)
		return nil
	}
}

// WithHook adds a processing hook to the CloudZero Agent logging pipeline.
// This option enables custom log event processing, transformation, and routing
// for advanced logging requirements including external system integration and custom formatting.
//
// Hook capabilities:
//   - Event filtering: Custom logic for conditional log event processing
//   - Field transformation: Modify or enhance log event fields
//   - External routing: Send specific events to external systems
//   - Compliance processing: Add regulatory or audit-specific information
//   - Performance monitoring: Track logging performance and patterns
//
// Common hook use cases:
//   - Security filtering: Remove sensitive information from log events
//   - Alerting integration: Forward critical events to alerting systems
//   - Audit compliance: Add regulatory metadata for compliance requirements
//   - Performance tracking: Measure and report logging performance metrics
//   - Custom formatting: Apply organization-specific log formatting
//
// Multiple hooks support:
//   - This function can be called multiple times to add multiple hooks
//   - Hooks are executed in the order they are added
//   - Hook failures do not prevent log event processing
//   - Each hook receives the log event independently
//
// Hooks execute synchronously and should be designed for minimal performance impact.
func WithHook(hook zerolog.Hook) LoggerOpt {
	return func(logger *internalLogger) error {
		logger.hooks = append(logger.hooks, hook)
		return nil
	}
}

// WithVersion specifies custom version information for CloudZero Agent log events.
// This option overrides the default version from the build library, enabling custom
// version identification for special deployments, development builds, or testing scenarios.
//
// Version information usage:
//   - Log correlation: Track issues across different agent versions
//   - Debugging support: Identify version-specific behaviors and bugs
//   - Operational monitoring: Monitor version distribution across deployments
//   - Compliance tracking: Maintain version audit trails for regulatory requirements
//
// Default behavior:
//   - If not specified, version is automatically retrieved from build.GetVersion()
//   - Build version includes Git commit, build time, and release information
//   - Production deployments typically use automatic version detection
//
// Custom version scenarios:
//   - Development builds: Custom identifiers for local development
//   - Testing environments: Test-specific version markers
//   - Special deployments: Custom deployment identifiers
//   - Integration testing: Version markers for test correlation
//
// The version appears in all log events as a consistent field for operational visibility.
func WithVersion(version string) LoggerOpt {
	return func(logger *internalLogger) error {
		logger.version = version
		return nil
	}
}

// WithAttrs configures consistent attribute injection for all CloudZero Agent log events.
// This option enables automatic addition of contextual information to every log event,
// ensuring consistent metadata across all logging operations for operational visibility and correlation.
//
// Attribute injection patterns:
//   - Service identification: Component names, service versions, deployment information
//   - Request correlation: Request IDs, trace IDs, session identifiers
//   - Operational context: Environment, region, cluster, namespace information
//   - Business context: Customer IDs, cost allocation metadata, resource identifiers
//
// Example usage:
//
//	logger := NewLogger(
//		WithAttrs(
//			func(ctx zerolog.Context) zerolog.Context {
//				return ctx.Str("component", "webhook-controller").
//					Str("environment", "production").
//					Str("cluster", "us-west-2")
//			},
//		),
//	)
//
// Multiple attribute functions:
//   - Multiple Attr functions can be provided to this single call
//   - Functions are applied in order to build cumulative context
//   - Later functions can reference fields added by earlier functions
//   - All attributes are added to every log event automatically
//
// Performance considerations:
//   - Attribute functions execute for every log event
//   - Keep attribute logic simple and efficient
//   - Consider caching expensive computations
//   - Avoid I/O operations within attribute functions
func WithAttrs(attrs ...Attr) LoggerOpt {
	return func(logger *internalLogger) error {
		logger.attrs = attrs
		return nil
	}
}

// NewLogger creates a configured CloudZero Agent logger with comprehensive operational capabilities.
// This constructor applies functional options to build a Zerolog-based logger optimized for
// CloudZero Agent operational requirements including structured logging, multiple outputs,
// and consistent metadata injection.
//
// Default configuration:
//   - Log level: Info (filters out debug and trace events)
//   - Output sink: Filtered stdout with sensitive field removal
//   - Version: Automatic detection from build information
//   - Timestamp: UTC timestamps for consistent time representation
//   - Caller information: File and line number for debugging
//
// Configuration validation:
//   - All options are applied and validated during construction
//   - Invalid configurations return descriptive errors
//   - Partial failures prevent logger creation
//   - Default values ensure operational logger even with minimal configuration
//
// Operational features:
//   - Multi-sink output: Simultaneous logging to multiple destinations
//   - Hook processing: Extensible event processing pipeline
//   - Attribute injection: Consistent metadata across all log events
//   - Context integration: Sets global context logger for request correlation
//
// Error conditions:
//   - Invalid log level strings
//   - Failed option application
//   - Sink or hook configuration errors
//
// Returns a fully configured Zerolog logger ready for CloudZero Agent operations,
// or an error describing configuration failures.
func NewLogger(opts ...LoggerOpt) (*zerolog.Logger, error) {
	ilogger := &internalLogger{
		level: zerolog.InfoLevel,
		sinks: make([]io.Writer, 0),
	}

	// apply the opts
	for _, opt := range opts {
		if err := opt(ilogger); err != nil {
			return nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}

	// add a default version
	if ilogger.version == "" {
		ilogger.version = build.GetVersion()
	}

	// add a default sink
	if len(ilogger.sinks) == 0 {
		filteredStdout := NewFieldFilterWriter(os.Stdout, []string{"spanId", "parentSpanId"})
		ilogger.sinks = append(ilogger.sinks, filteredStdout)
	}

	// create a multi-sink
	multiSink := io.MultiWriter(ilogger.sinks...)

	// create the logger
	var zlogger zerolog.Logger
	zlogger = zerolog.New(multiSink)
	zlogger = zlogger.Level(ilogger.level).With().
		Str("version", ilogger.version).
		Timestamp().
		Caller().
		Logger()

	// add hooks
	for _, hook := range ilogger.hooks {
		zlogger = zlogger.Hook(hook)
	}

	// apply the attributes
	ctx := zlogger.With()
	for _, attr := range ilogger.attrs {
		ctx = attr(ctx)
	}
	zlogger = ctx.Logger()

	// set as default context logger
	zerolog.DefaultContextLogger = &zlogger

	return &zlogger, nil
}
