# Mock Services and Test Data

Mock services and test data generators for testing CloudZero Agent functionality without external dependencies.

## Components

**Controller (`controller/`):**

- Mock Kubernetes controller components for admission control testing

**Metrics (`metrics/`):**

- `metrics.go` - Test metrics data generation utilities
- `memory.go`, `cpu.go` - Resource usage metric generators
- `pod_history.go`, `node_history.go` - Historical metric data
- `summary_record.go` - Metric summary generation

**Remote Write (`remotewrite/`):**

- Mock Prometheus remote write endpoints for metric ingestion testing

## Usage

```bash
make test               # Unit tests using mocks
make test-integration   # Integration tests with mock services
```
