# Structured Logging Infrastructure

## Purpose

Provides structured logging infrastructure built on Zerolog. Enables consistent structured logging across all components with configurable output destinations and filtering.

## Core Components

- `logger.go` - Primary logging infrastructure with Zerolog integration
- `filtered_writer.go` - Configurable log filtering and routing
- `store_sink.go` - Storage sink for log persistence

## Specialized Packages

**Instrumentation (`instr/`):**

- `context.go` - Context-aware logging with trace correlation
- `metrics.go` - Logging metrics and monitoring
- `middleware.go` - HTTP request/response logging
- `span.go` - Distributed tracing span logging

**Validator-Specific (`validator/`):**

- `logging.go` - Validator component logging configuration
- `text_format.go` - Custom text formatting
- `sequence.go` - Sequential log processing

## Testing

```bash
make test GO_TEST_TARGET=./app/logging/...
```
