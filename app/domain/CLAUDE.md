# Domain Package - Business Logic

## Purpose

Contains core business logic for CloudZero Agent cost allocation, metric processing, and Kubernetes resource management. Most services use dependency injection through types/ interfaces, though some services (shipper, backfiller) have direct HTTP client dependencies.

## Core Services

### Metric Processing

- **`metric_collector.go`** - Prometheus remote_write ingestion
- **`filter/`** - Cost vs observability metric classification
- **`shipper/`** - CloudZero platform data upload

### Kubernetes Integration

- **`webhook/`** - Admission control for resource metadata collection
- **`k8s/`** - Kubernetes client abstractions

### Operational Services

- **`monitor/`** - Secret rotation and certificate management
- **`healthz/`** - Service health monitoring
- **`diagnostic/`** - System diagnostics and troubleshooting

## Subdirectories

- **[webhook/](./webhook/)** - Kubernetes admission webhook logic
- **[filter/](./filter/)** - Metric filtering and classification
- **[shipper/](./shipper/)** - Data shipping to CloudZero platform
- **[monitor/](./monitor/)** - Certificate and secret management
- **[healthz/](./healthz/)** - Health checking services
- **[k8s/](./k8s/)** - Kubernetes client utilities
- **[diagnostic/](./diagnostic/)** - Diagnostic tools and checks

## Testing

```bash
# Test domain layer
make test GO_TEST_TARGET=./app/domain

# Test with mocks
make generate && make test GO_TEST_TARGET=./app/domain
```

## Architecture Role

**Application Core** - Business logic layer using dependency injection. Most external dependencies injected through interfaces defined in `types/`, with exceptions in shipper and backfiller services that create HTTP clients directly.
