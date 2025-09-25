# Smoke Tests

This directory contains high-level smoke tests that validate the core functionality of the CloudZero Agent using Docker containers and testcontainers. These tests provide end-to-end validation of data collection, processing, and shipping workflows.

## Overview

Smoke tests use testcontainers to orchestrate multi-container environments that simulate real-world deployments. They test the complete data flow from collection through shipping to remote endpoints.

## Test Components

### Core Test Files

- **`smoke.go`** - Test framework and utilities for container orchestration
- **`collector_test.go`** - Tests for the metric collection component
- **`shipper_test.go`** - Tests for the data shipping component
- **`client_test.go`** - Tests for HTTP client functionality
- **`remotewrite_test.go`** - Tests for remote write endpoint functionality
- **`load_test.go`** - Load and performance testing scenarios

### Supporting Files

- **`collector.go`** - Collector container setup and configuration
- **`shipper.go`** - Shipper container setup and configuration
- **`remotewrite.go`** - Mock remote write server implementation
- **`mock_controller.go`** - Mock controller for test coordination

## What is Tested

### Data Collection Flow

- Metric collection from Kubernetes resources
- Data persistence to disk storage
- File compression and formatting (Brotli compression)
- Database storage and retrieval

### Data Shipping Flow

- Reading metrics from disk storage
- Batching and compression of data
- HTTP transmission to remote endpoints
- Error handling and retry logic
- Authentication with CloudZero API

### Container Orchestration

- Multi-container networking
- Service discovery between containers
- Container lifecycle management
- Resource cleanup and teardown

### Performance Characteristics

- Load handling capabilities
- Memory and CPU usage patterns
- Network throughput and latency
- Concurrent processing scenarios

## Prerequisites

### Required Software

- Docker and Docker Compose
- Go 1.21+ installed
- Network access for container image pulling

### Environment Variables

- `CLOUDZERO_DEV_API_KEY` - API key for CloudZero services (optional, defaults to "ak-test")
- `CLOUDZERO_HOST` - CloudZero API endpoint (optional, defaults to mock server)

## How to Run Tests

**IMPORTANT**: Smoke tests ARE integrated with the main project Makefile.

### Makefile Integration (Recommended)

```bash
# Run smoke tests via Makefile
make test-smoke
```

This runs: `go -C tests test -run Smoke -v -timeout 10m ./smoke/...`

### Manual Execution

You can also run smoke tests manually:

```bash
# From project root
cd tests && go test ./smoke/

# From tests/ directory
go test ./smoke/
```

### Run Specific Test Suites

```bash
# Test collector functionality
cd tests && go test ./smoke/ -run TestCollector

# Test shipper functionality
cd tests && go test ./smoke/ -run TestShipper

# Test remote write functionality
cd tests && go test ./smoke/ -run TestRemoteWrite

# Run load tests
cd tests && go test ./smoke/ -run TestLoad
```

### Run with Verbose Output

```bash
go test -v ./tests/smoke/
```

### Run with Custom Timeouts

```bash
go test -timeout 10m ./tests/smoke/
```

## Test Configuration

### Default Configuration

Tests use a default configuration that includes:

- Test cluster name: "smoke-test-cluster"
- Test account ID: "test-account-id"
- Debug logging enabled
- 1-minute intervals for cost and observability metrics
- 90-day data retention

### Customizing Tests

Use test context options to customize behavior:

```go
// Example: Test with custom upload delay
runTest(t, func(tx *testContext) {
    // Test implementation
}, withUploadDelayMs("5000"))

// Example: Test with config override
runTest(t, func(tx *testContext) {
    // Test implementation
}, withConfigOverride(func(cfg *config.Settings) {
    cfg.Database.CostMaxInterval = time.Second * 30
}))
```

## Container Architecture

### Network Setup

- Tests create isolated Docker networks
- Containers communicate using internal DNS names
- Network cleanup handled automatically

### Container Types

- **Collector Container** - Runs the metrics collector
- **Shipper Container** - Runs the data shipper
- **Mock Remote Write** - Simulates CloudZero API endpoint
- **S3 Mock** - Simulates S3 storage (when needed)
- **Controller** - Coordinates test scenarios

### Data Flow

1. Collector gathers metrics and writes to shared storage
2. Shipper reads from storage and sends to remote endpoint
3. Mock servers validate received data
4. Test assertions verify expected behavior

## Test Data Generation

### Metric Generation

The `WriteTestMetrics()` function creates realistic test data:

- Alternates between cost and observability metrics
- Uses compressed JSON format with Brotli
- Supports custom file paths and quantities
- Includes realistic timestamps and metadata

### File Naming Convention

Generated files follow the pattern:

- `cost_<start_timestamp>_<end_timestamp>.json.br`
- `observability_<start_timestamp>_<end_timestamp>.json.br`

## Troubleshooting

### Common Issues

- **Docker permission errors** - Ensure user is in docker group
- **Port conflicts** - Tests use random ports, but conflicts can occur
- **Network timeouts** - Check firewall settings and Docker networking
- **Container startup failures** - Verify Docker daemon is running

### Debug Tips

- Use `-v` flag for verbose test output
- Check container logs in test output
- Verify environment variables are set correctly
- Ensure adequate disk space for test data

### Cleanup

Tests automatically clean up resources, but manual cleanup may be needed:

```bash
# Remove test containers
docker container prune

# Remove test networks
docker network prune

# Remove test volumes
docker volume prune
```

## Adding New Smoke Tests

When adding new smoke tests:

1. **Use the `runTest()` wrapper** - Provides consistent test context setup
2. **Follow container patterns** - Use existing container setup methods
3. **Include cleanup** - Ensure proper resource cleanup
4. **Test both success and failure** - Include error scenarios
5. **Use realistic data** - Generate meaningful test metrics
6. **Document configuration** - Explain any special setup requirements

Example test structure:

```go
func TestNewFeature(t *testing.T) {
    runTest(t, func(tx *testContext) {
        // Setup containers and test data
        tx.CreateNetwork()
        tx.WriteTestMetrics(5, 100)

        // Start containers and run test
        // ... test implementation

        // Verify results
        // ... assertions
    })
}
```
