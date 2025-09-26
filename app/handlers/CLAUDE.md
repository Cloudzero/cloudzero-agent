# Handlers Package - HTTP Endpoints

## Purpose

Implements Primary Adapters that translate HTTP requests into domain service operations. Handles Prometheus remote_write, Kubernetes webhooks, and operational endpoints.

## HTTP Endpoints

### Core APIs

- **`webhook.go`** - Kubernetes admission webhooks (fail-open design)
- **`remote_write.go`** - Prometheus remote_write endpoint (v1 & v2)
- **`prom_metrics.go`** - Prometheus metrics exposition (/metrics)
- **`shipper.go`** - Operational monitoring endpoints
- **`profiling.go`** - Go pprof profiling endpoints (debug)

## Integration Patterns

### Kubernetes Webhooks

- **Fail-open design** - Never blocks cluster operations
- **TLS required** - HTTPS-only admission control
- **Load balancing** - Connection management for HA

### Prometheus Integration

- **High-throughput** - Handles large metric payloads
- **Compression support** - Snappy decompression
- **Protocol compliance** - Full remote_write specification

## Testing

```bash
# Test handlers
make test GO_TEST_TARGET=./app/handlers

# Integration tests with real HTTP
make test-integration GO_TEST_TARGET=./app/handlers
```

## Architecture Role

**Primary Adapter** - HTTP layer that validates requests and delegates to domain services. No business logic in handlers.
