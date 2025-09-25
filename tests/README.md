# CloudZero Agent Test Suite

This directory contains the comprehensive test suite for the CloudZero Agent. The tests are organized by type and purpose to ensure thorough validation of the agent's functionality across different environments and use cases.

## Test Structure Overview

- **[app/](app/)** - Unit tests for application components (HTTP clients, domain logic, logging)
- **[backfiller/](backfiller/)** - Tests for Kubernetes resource metadata collection job
- **[docker/](docker/)** - Docker-based containerization tests
- **[helm/](helm/)** - Helm chart validation and Kubernetes deployment tests
- **[integration/](integration/)** - End-to-end integration tests with external services
- **[kind/](kind/)** - Kubernetes-in-Docker local cluster tests
- **[kuttl/](kuttl/)** - Kubernetes Test Tool (KUTTL) declarative tests
- **[load/](load/)** - Performance and load testing
- **[smoke/](smoke/)** - High-level smoke tests using Docker containers
- **[testkube/](testkube/)** - TestKube workflow tests for CI/CD pipelines
- **[utils/](utils/)** - Shared testing utilities and helpers
- **[webhook/](webhook/)** - Kubernetes admission webhook tests

## Test Categories

### Unit Tests

Found in `app/` - Test individual components in isolation:

- HTTP client functionality
- Domain logic (diagnostics for KMS, K8s, CloudZero)
- Logging and validation systems

### Integration Tests

Found in `integration/`, `webhook/`, `helm/` - Test component interactions:

- Full system behavior with real services
- Kubernetes admission webhook functionality
- Helm chart deployment and configuration

### Container Tests

Found in `smoke/`, `docker/` - Test containerized environments:

- Multi-container orchestration with testcontainers
- Docker networking and service discovery
- End-to-end data flow validation

### Kubernetes Tests

Found in `kind/`, `kuttl/`, `testkube/`, `helm/` - Test Kubernetes deployments:

- Local cluster validation with KIND
- Declarative test scenarios with KUTTL
- CI/CD pipeline tests with TestKube
- Helm chart installation and configuration

### Performance Tests

Found in `load/` - Test system performance and scalability

## Prerequisites

### General Requirements

- Go 1.21+ installed
- Docker and Docker Compose
- Access to test API keys (set `CLOUDZERO_DEV_API_KEY` environment variable)

### Kubernetes Testing

- `kubectl` configured
- `helm` CLI tool
- Kind or access to a Kubernetes cluster

### Integration Testing

- Network access to CloudZero services
- Valid API credentials

## Running Tests

**IMPORTANT**: Tests in this directory have their own `go.mod` and are separate from the main application tests.

### Makefile Targets

#### Core Test Targets

- **`make test`** - Runs unit tests for the main application code (in `app/`, `mock/`, etc.) but NOT tests in this `tests/` directory
- **`make test-integration`** - Runs tests with "Integration" in the function name across ALL directories (including `tests/integration/`) using `-run Integration` filter
- **`make test-smoke`** - Runs smoke tests from `tests/smoke/` directory using `go -C tests test ./smoke/...`

#### Helm Test Targets (Fully Integrated)

- **`make helm-test`** - Runs all Helm validation tests (schema, subchart, unittest, template)
- **`make helm-test-schema`** - Runs all schema validation tests
- **`make helm-test-schema-template`** - Runs only template rendering validation
- **`make helm-test-schema-kubeconform`** - Runs only kubeconform validation
- **`make helm-test-subchart`** - Runs subchart validation tests
- **`make helm-test-unittest`** - Runs helm-unittest tests (from `helm/tests/`)
- **`make helm-test-template`** - Generates templates from override files
- **`make helm-test-template-diff`** - Compares templates with git version

#### Kubernetes/KIND Test Targets

- **`make kind-test`** - Complete workflow: create cluster, install chart, run KUTTL tests, cleanup
- **`make kind-up`** - Create KIND cluster for testing
- **`make kind-down`** - Delete KIND cluster and cleanup
- **`make helm-test-kuttl`** - Run KUTTL tests (assumes cluster exists)

#### Individual Test Targets

- **`make tests/helm/schema/<test-name>`** - Run specific schema test
- **`make tests/helm/schema/<test-name>-template`** - Run template validation for specific test
- **`make tests/helm/schema/<test-name>-kubeconform`** - Run kubeconform validation for specific test
- **`make tests/helm/template/<name>.yaml`** - Generate specific template
- **`make tests/kuttl/<suite>/run`** - Run specific KUTTL test suite

#### CI Test Targets

- **`make test-ci`** - Run all CI test suites
- **`make test-ci-chart-kuttl`** - Run chart tests via ACT (GitHub Actions)
- **`make test-ci-docker-build`** - Test docker build workflow

### Example Usage of Makefile Targets

#### Running Specific Helm Schema Tests

```bash
# Run a specific schema test (both template and kubeconform validation)
make tests/helm/schema/components.webhookServer.backfill.schedule.valid3.pass

# Run only template validation for a specific test
make tests/helm/schema/insightsController.certificate.defaults.pass-template

# Run only kubeconform validation for a specific test
make tests/helm/schema/components.webhookServer.backfill.schedule.valid3.pass-kubeconform

# Run a failure test (only template validation)
make tests/helm/schema/image.digest.invalid.fail
```

#### Running Template Generation

```bash
# Generate all templates
make helm-test-template

# Generate specific templates
make tests/helm/template/manifest.yaml
make tests/helm/template/federated.yaml

# Compare templates with git version
make helm-test-template-diff
make tests/helm/template/manifest.yaml-semdiff
```

#### Running KUTTL Tests

```bash
# Run all KUTTL tests
make helm-test-kuttl

# Run specific KUTTL test suite
make tests/kuttl/collector-test/run
make tests/kuttl/webhook-comprehensive-test/run
```

### Manual Test Execution

Tests NOT integrated with Makefile targets must be run manually:

#### Unit Tests (tests/app/)

```bash
# From the tests/ directory:
cd tests

# Run all unit tests
go test ./app/...

# Run specific component tests
go test ./app/http/client/...
go test ./app/domain/diagnostic/...
```

#### Integration Tests (tests/integration/, tests/webhook/)

```bash
# From the tests/ directory:
cd tests

# Run integration tests
go test -tags=integration ./integration/...
go test -tags=integration ./webhook/...
```

#### Smoke Tests (tests/smoke/)

```bash
# Use the Makefile target (recommended)
make test-smoke

# Or run manually from tests/ directory:
cd tests && go test ./smoke/...
```

### Kubernetes Tests

```bash
# Run KIND-based tests
go test ./kind/...

# Run KUTTL tests
kubectl kuttl test ./kuttl/

# Run TestKube workflows
testkube run workflow agent-basic-test
```

### Load Tests

```bash
# Run performance tests
go test ./load/...
```

## Test Configuration

### Environment Variables

- `CLOUDZERO_DEV_API_KEY` - API key for CloudZero services
- `CLOUDZERO_HOST` - Override default CloudZero host endpoint
- `TEST_TIMEOUT` - Override default test timeout values

### Test Data

- Test data files are located in `*/testdata/` directories
- Configuration templates are in `helm/testdata/`
- Mock services and fixtures are defined per test type

## Continuous Integration

Tests are organized for different CI environments:

- **Unit tests** - Run on every commit
- **Integration tests** - Run on pull requests
- **Smoke tests** - Run nightly or before releases
- **Kubernetes tests** - Run in dedicated cluster environments

## Test Integration Summary

### ✅ Fully Integrated with Main Makefile

- **Helm tests** (`tests/helm/`) - Complete integration with granular control
- **KUTTL tests** (`tests/kuttl/`) - Integrated via `helm-test-kuttl` and individual targets
- **KIND cluster management** (`tests/kind/`) - Integrated cluster lifecycle
- **Smoke tests** (`tests/smoke/`) - Integrated via `test-smoke` target

### ⚠️ Partially Integrated

- **CI tests** - Special targets for GitHub Actions testing

### ❌ NOT Integrated (Manual Execution Required)

- **Unit tests** (`tests/app/`) - Must run with `cd tests && go test ./app/...`
- **Integration tests** (`tests/integration/`) - Partially integrated: tests with "Integration" in function name run via `make test-integration`, others require `cd tests && go test -tags=integration ./integration/...`
- **Webhook tests** (`tests/webhook/`) - Must run with `cd tests && go test -tags=integration ./webhook/...`
- **Backfiller tests** (`tests/backfiller/`) - Has own Makefile, run with `cd tests/backfiller && make test-backfiller`
- **TestKube tests** (`tests/testkube/`) - Run via TestKube framework, not Make
- **Load tests** (`tests/load/`) - Manual execution required
- **Utility tests** (`tests/utils/`) - Must run with `cd tests && go test ./utils/...`

### 🔧 Docker tests (`tests/docker/`)

- Not run directly - containers built automatically by smoke tests

## Adding New Tests

When adding new functionality:

1. **Unit tests** - Add to `app/` following existing patterns
2. **Integration tests** - Add to appropriate directory with `integration` build tag
3. **Container tests** - Use testcontainers pattern from `smoke/` directory
4. **Kubernetes tests** - Add YAML manifests to appropriate test framework directory
5. **Helm tests** - Add to `tests/helm/schema/` or `tests/helm/template/` for automatic integration

See individual directory README files for specific testing patterns and examples.

## Troubleshooting

### Common Issues

- **Docker permission errors** - Ensure user is in docker group
- **Network timeouts** - Check firewall and proxy settings
- **API authentication failures** - Verify `CLOUDZERO_DEV_API_KEY` is set
- **Kubernetes connection issues** - Verify `kubectl` context and permissions

### Debug Mode

Enable verbose testing output:

```bash
go test -v -tags=integration ./...
```

Enable debug logging in tests:

```bash
LOGLEVEL=debug go test ./...
```
