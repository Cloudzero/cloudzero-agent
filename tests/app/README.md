# App Unit Tests

This directory contains unit tests for the core application components of the CloudZero Agent. These tests validate individual components in isolation without external dependencies.

## Test Structure

### HTTP Client Tests (`http/client/`)

Tests the HTTP client functionality used for external API communications.

**What is tested:**

- HTTP request/response handling
- Error handling and retries
- Authentication mechanisms
- Connection timeouts and failures

**How to run:**

```bash
# These tests are NOT run by the standard make targets
# To run them, you must run from the tests/ directory:
cd tests && go test ./app/http/client/...
```

### Domain Logic Tests (`domain/`)

#### Diagnostic Tests (`domain/diagnostic/`)

Tests the health check and diagnostic systems that validate external service connectivity.

**Subcomponents:**

- **`kms/`** - AWS KMS connectivity and permissions checks
- **`k8s/version/`** - Kubernetes API server version compatibility checks
- **`cz/`** - CloudZero API connectivity and authentication checks

**What is tested:**

- Service availability checks
- Authentication validation
- API compatibility verification
- Error handling for service failures

**How to run:**

```bash
# These tests are NOT run by the standard make targets
# To run them, you must run from the tests/ directory:

# Run all diagnostic tests
cd tests && go test ./app/domain/diagnostic/...

# Run specific diagnostic tests
cd tests && go test ./app/domain/diagnostic/kms/...
cd tests && go test ./app/domain/diagnostic/k8s/version/...
cd tests && go test ./app/domain/diagnostic/cz/...
```

### Logging Tests (`logging/`)

#### Validator Tests (`logging/validator/`)

Tests the log validation and sequencing functionality.

**What is tested:**

- Log message formatting and validation
- Sequence numbering and ordering
- Error log handling
- Log level filtering

**How to run:**

```bash
# These tests are NOT run by the standard make targets
# To run them, you must run from the tests/ directory:
cd tests && go test ./app/logging/validator/...
```

## Prerequisites

- Go 1.21+ installed
- No external dependencies required (unit tests use mocks)

## Test Patterns

These unit tests follow standard Go testing patterns:

- Use `testing.T` for test execution
- Employ table-driven tests where appropriate
- Mock external dependencies
- Focus on individual component behavior

## Adding New Tests

When adding new unit tests to this directory:

1. **Follow the existing directory structure** - Place tests alongside the code they test
2. **Use descriptive test names** - Follow the pattern `TestFunctionName_Scenario`
3. **Mock external dependencies** - Unit tests should not make real network calls
4. **Test both success and failure cases** - Include error handling validation
5. **Use testify/require** - Follow existing assertion patterns

Example test structure:

```go
func TestHTTPClient_SendRequest_Success(t *testing.T) {
    // Arrange - set up test data and mocks
    client := NewHTTPClient()

    // Act - perform the operation being tested
    result, err := client.SendRequest(testRequest)

    // Assert - verify expected outcomes
    require.NoError(t, err)
    require.Equal(t, expectedResult, result)
}
```

## Common Issues

- **Import path errors** - Ensure test files use correct relative imports
- **Mock setup** - Verify mocks are properly configured for the component under test
- **Test isolation** - Each test should be independent and not rely on shared state
