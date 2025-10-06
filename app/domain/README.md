# Domain Layer

The domain layer implements the Application Core in CloudZero Agent's hexagonal architecture, containing core business logic for cost allocation, metric processing, and Kubernetes resource management.

## Key Components

### Metric Processing

- **`metric_collector.go`** - Processes all Prometheus metrics (both cost and observability)
- **`filter/`** - Classifies metrics by type (cost vs observability)
- **`shipper/`** - Uploads processed data to CloudZero platform

### Kubernetes Integration

- **`webhook/`** - Admission control for resource metadata collection
- **`k8s/`** - Kubernetes client abstractions and utilities

### Operational Services

- **`monitor/`** - Certificate management and secret rotation
- **`healthz/`** - Health checking and service monitoring
- **`diagnostic/`** - System diagnostics and troubleshooting

## Architecture

Uses hexagonal (ports and adapters) architecture with dependency injection. Business logic depends on interfaces defined in `../types/`, with concrete implementations in adapters.

## Testing

```sh
# Test domain layer
make test GO_TEST_TARGET=./app/domain

# Test with mocks
make generate && make test GO_TEST_TARGET=./app/domain
```

## Development

See [CLAUDE.md](./CLAUDE.md) for detailed architecture documentation and AI-assistant guidance.
