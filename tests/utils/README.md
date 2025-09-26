# Test Utilities

This directory contains shared utilities and helper functions used across different test suites in the `tests/` directory. These utilities provide common functionality for test setup, mocking, and validation.

## Overview

The test utilities provide reusable components that help create consistent and maintainable tests across the entire test suite. They handle common patterns like HTTP mocking, container management, logging, and test conditions.

## Utility Components

### HTTP Testing (`roundtripper.go`)

Provides HTTP client mocking and testing utilities:

**What it provides:**

- HTTP request/response mocking
- Custom transport implementations for testing
- Request validation and response simulation
- Integration with Go's `http.Client` for seamless mocking

**Usage patterns:**

```go
// Used in tests that need to mock HTTP calls
mock := utils.NewHTTPMock()
mock.Expect(http.MethodGet, expectedResponse, 200, nil)
client := mock.HTTPClient()
```

### Container Testing (`container.go`)

Utilities for container-based testing scenarios:

**What it provides:**

- Container lifecycle management helpers
- Docker integration utilities
- Container networking and configuration helpers
- Cleanup and resource management patterns

### Logging Testing (`logging.go`, `logging_test.go`)

Logging utilities and validation for test scenarios:

**What it provides:**

- Test-specific logging configuration
- Log capture and validation utilities
- Debug output management for tests
- Integration with the agent's logging system

**Tests include:**

- Log level handling
- Log format validation
- Output capture and analysis

### Test Conditions (`condition.go`)

Provides condition checking and wait utilities:

**What it provides:**

- Wait conditions for asynchronous operations
- Retry logic for flaky operations
- Timeout handling patterns
- Polling utilities for eventual consistency

## How to Use These Utilities

### In Unit Tests

```go
import "github.com/cloudzero/cloudzero-agent/tests/utils"

func TestMyFeature(t *testing.T) {
    // Use HTTP mocking
    mock := utils.NewHTTPMock()
    mock.Expect(http.MethodPost, `{"result": "success"}`, 200, nil)

    // Use logging utilities
    logger := utils.NewTestLogger(t)

    // Test your code with mocked dependencies
}
```

### In Integration Tests

```go
import "github.com/cloudzero/cloudzero-agent/tests/utils"

func TestIntegration(t *testing.T) {
    // Use wait conditions
    err := utils.WaitForCondition(func() bool {
        return checkServiceReady()
    }, time.Second*30, time.Second)

    require.NoError(t, err, "Service should be ready")
}
```

### In Container Tests

```go
import "github.com/cloudzero/cloudzero-agent/tests/utils"

func TestWithContainer(t *testing.T) {
    // Use container utilities
    container := utils.SetupTestContainer(t, containerConfig)
    defer utils.CleanupContainer(t, container)

    // Run tests against container
}
```

## Prerequisites

Since these are utility functions, they have minimal prerequisites:

- Go 1.21+ installed
- Dependencies used by the specific utilities you're importing
- For container utilities: Docker access

## Running Utility Tests

**IMPORTANT**: These tests follow the same pattern as other `tests/` directory tests - they are NOT run by the main project make targets.

```bash
# Run utility tests manually
cd tests
go test ./utils/...

# Run with verbose output
cd tests
go test -v ./utils/...

# Run specific utility tests
cd tests
go test ./utils/ -run TestLogging
```

## Common Patterns

### HTTP Mocking Pattern

```go
// Create mock
mock := utils.NewHTTPMock()

// Set expectations
mock.Expect(method, responseBody, statusCode, headers)

// Get HTTP client with mock transport
client := mock.HTTPClient()

// Use client in code under test
result, err := yourService.CallAPI(client)

// Verify expectations were met
mock.VerifyExpectations(t)
```

### Condition Waiting Pattern

```go
// Wait for condition with timeout
err := utils.WaitForCondition(func() bool {
    return someAsyncCondition()
}, timeout, pollInterval)

require.NoError(t, err, "Condition should eventually be true")
```

### Container Testing Pattern

```go
// Setup container for test
container := utils.SetupTestContainer(t, config)
defer utils.CleanupContainer(t, container)

// Get container endpoint
endpoint := utils.GetContainerEndpoint(container, port)

// Run tests against container
testAgainstContainer(endpoint)
```

## Design Principles

### Reusability

Utilities are designed to be reusable across different test types:

- Unit tests can use HTTP mocking
- Integration tests can use wait conditions
- Container tests can use lifecycle management

### Consistency

Utilities provide consistent patterns across the test suite:

- Standard error handling
- Common timeout and retry behavior
- Uniform logging and debugging

### Isolation

Each utility function is independent and doesn't have hidden dependencies:

- Clear input/output interfaces
- No global state dependencies
- Explicit configuration parameters

## Adding New Utilities

When adding new test utilities:

1. **Keep them focused** - Each utility should have a single responsibility
2. **Make them reusable** - Design for use across multiple test types
3. **Include tests** - Utility functions should have their own tests
4. **Document usage** - Provide clear examples and patterns
5. **Handle cleanup** - Include proper resource cleanup patterns

Example utility structure:

```go
// NewTestHelper creates a new test helper with given configuration
func NewTestHelper(config Config) *TestHelper {
    return &TestHelper{config: config}
}

// SetupResource sets up a test resource and returns cleanup function
func (h *TestHelper) SetupResource(t *testing.T) (*Resource, func()) {
    resource := createResource(h.config)

    cleanup := func() {
        cleanupResource(resource)
    }

    return resource, cleanup
}
```

## Best Practices

### Error Handling

- Always propagate errors up to the test
- Use descriptive error messages
- Include context about what was being attempted

### Resource Management

- Always provide cleanup functions
- Use `t.Cleanup()` for automatic cleanup
- Handle partial setup failures gracefully

### Configuration

- Make utilities configurable rather than hardcoded
- Provide sensible defaults
- Allow timeout and retry configuration

### Testing

- Test your utility functions themselves
- Include edge cases and error scenarios
- Mock external dependencies in utility tests

This utilities package ensures consistent, maintainable testing patterns across the entire CloudZero Agent test suite.
