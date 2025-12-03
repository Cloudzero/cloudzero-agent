# Integration Tests

This directory contains integration tests that validate component interactions and system behavior in more realistic environments. These tests require external dependencies and may make network calls to test real integrations.

## Overview

Integration tests verify that multiple components work together correctly and that the system behaves properly when interacting with external services. They bridge the gap between unit tests and full end-to-end tests.

## Test Files

### Core Integration Tests

- **`integration_test.go`** - Main integration test suite (currently commented out)
  - Tests HTTP API endpoints and routing
  - Validates request/response handling
  - Tests authentication and authorization flows

- **`shutdown_coordination_test.go`** - Tests shutdown coordination between collector and shipper
  - File-based coordination mechanism testing
  - Collector shutdown marker creation
  - Shipper coordination with collector shutdown
  - Timeout handling scenarios
  - Configurable settings validation

### Supporting Files

- **`config.go`** - Integration test configuration utilities
- **`helpers.go`** - Common helper functions for integration tests
- **`test-config.yaml`** - Test configuration file
- **`fake-api-key`** - Mock API key for testing (not a real key)
- **`test_server/`** - Test server implementation for integration scenarios

## What is Tested

### Shutdown Coordination

- **File-based coordination** - Tests the mechanism used to coordinate graceful shutdowns
- **Collector shutdown markers** - Validates creation and cleanup of shutdown marker files
- **Shipper coordination** - Tests how shipper responds to collector shutdown signals
- **Timeout handling** - Ensures proper behavior when coordination timeouts occur
- **Configuration integration** - Tests shutdown coordination with various config settings

### HTTP API Integration (Commented Out)

The main integration tests are currently disabled but would test:

- API endpoint routing and responses
- Authentication with real or mock services
- Request validation and error handling
- Integration with external services

## Prerequisites

### Required Environment

- Go 1.21+ installed
- **Integration build tag required** - Tests use `//go:build integration`
- Network access (for tests that aren't mocked)

### Environment Variables

- `CLOUDZERO_DEV_API_KEY` - Required for authentication tests
- `CLOUDZERO_HOST` - CloudZero API endpoint
- Additional variables may be needed for specific test scenarios

## How to Run Tests

**IMPORTANT**: These tests are NOT automatically run by standard make targets.

### Manual Execution

```bash
# From the tests/ directory (required due to separate go.mod):
cd tests

# Run all integration tests
go test -tags=integration ./integration/...

# Run with verbose output
go test -v -tags=integration ./integration/...

# Run specific test functions
go test -tags=integration -run TestShutdownCoordination ./integration/...

# Skip integration tests in short mode
go test -short ./integration/... # (will skip integration tests)
```

### With Environment Setup

```bash
cd tests

# Set required environment variables
export CLOUDZERO_DEV_API_KEY="your-dev-key"
export CLOUDZERO_HOST="dev-api.cloudzero.com"

# Run tests
go test -tags=integration -v ./integration/...
```

## Test Configuration

### Build Tags

All integration tests use the `integration` build tag:

```go
//go:build integration
// +build integration
```

This ensures they only run when explicitly requested and don't interfere with unit test runs.

### Test Data Location

- Temporary directories created with `t.TempDir()`
- Test configuration files in same directory
- Mock API keys and configuration provided

## Common Integration Patterns

### File-Based Testing

Many integration tests use file system operations:

```go
tmpDir := t.TempDir()
shutdownFile := filepath.Join(tmpDir, config.ShutdownMarkerFilename)
// Test file operations
```

### Configuration Testing

Tests often create custom configurations:

```go
cfg := &config.Settings{
    // Custom test configuration
}
```

### Timeout Testing

Integration tests include timeout scenarios:

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

## Troubleshooting

### Common Issues

- **Missing build tag** - Ensure you use `-tags=integration` flag
- **Working directory** - Run from `tests/` directory due to separate go.mod
- **Environment variables** - Set required API keys and endpoints
- **Network access** - Some tests may require internet connectivity
- **File permissions** - Ensure write access for temporary file operations

### Debug Tips

```bash
# Run with verbose output to see detailed test execution
go test -v -tags=integration ./integration/...

# Run specific tests to isolate issues
go test -tags=integration -run TestShutdownCoordination_Integration ./integration/...

# Check test configuration
cat test-config.yaml
```

## Adding New Integration Tests

When adding new integration tests:

1. **Use the integration build tag**:

   ```go
   //go:build integration
   // +build integration
   ```

2. **Follow existing patterns** - Use helpers and configuration utilities
3. **Handle cleanup** - Use `t.TempDir()` and proper cleanup functions
4. **Test realistic scenarios** - Focus on component interactions
5. **Include error cases** - Test failure modes and error handling
6. **Document dependencies** - Clearly state external requirements

Example integration test structure:

```go
//go:build integration
// +build integration

func TestNewFeatureIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }

    // Setup
    tmpDir := t.TempDir()
    // ... test implementation

    // Cleanup handled automatically by t.TempDir()
}
```
