# Common Utilities

## Purpose

Provides reusable utility functions and types used across all architectural layers. Includes time management, data processing, and operational support utilities.

## Core Utilities

- `clock.go` - Clock abstraction with UTC standardization
- `chunk.go` - Data chunking for batching and streaming

## Specialized Packages

**Kubernetes (`k8s/`):**

- `services.go` - Service discovery and management

**Concurrency (`lock/`, `parallel/`):**

- `lock/` - File locking mechanisms
- `parallel/` - Parallel processing utilities

**Cloud Discovery (`scout/`):**

- Cloud provider detection and configuration discovery
- Supports `auto/`, `aws/`, `azure/` patterns

**Observability (`telemetry/`):**

- Telemetry collection and reporting

## Testing

```bash
make test GO_TEST_TARGET=./app/utils/...
```
