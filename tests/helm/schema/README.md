# Helm Schema Validation Tests

This directory contains comprehensive schema validation tests for the CloudZero Agent Helm chart. These tests ensure that chart values are properly validated and that templates render correctly with various configurations.

## Overview

Schema validation tests verify:

- **Chart values validation** - Values conform to the chart's schema
- **Template rendering** - Charts generate valid Kubernetes manifests
- **Kubeconform validation** - Generated manifests comply with Kubernetes API specifications
- **Edge cases** - Invalid configurations fail appropriately

## Test File Naming Convention

Test files follow a structured naming pattern:

```text
<component>.<field>.<condition>.<expected-result>.yaml
```

**Examples:**

- `aggregator.database.compressionLevel.valid.pass.yaml` - Valid compression level should pass
- `cloudAccountId.invalid.fail.yaml` - Invalid cloud account ID should fail
- `components.webhookServer.backfill.schedule.valid3.pass.yaml` - Valid webhook backfill schedule should pass

**Components:**

- `aggregator.*` - Tests for the data aggregator component
- `cloudAccountId.*` - Tests for cloud account configuration
- `components.*` - Tests for various chart components (webhook, insights controller, etc.)
- `image.*` - Tests for container image configuration
- `insightsController.*` - Tests for insights controller configuration

**Expected Results:**

- `.pass` - Test should succeed (valid configuration)
- `.fail` - Test should fail (invalid configuration)

## Test Types

### Template Validation Tests

These tests verify that Helm templates render successfully:

```bash
# Run template validation for a specific test
make tests/helm/schema/aggregator.database.compressionLevel.valid.pass-template

# Run template validation for all tests
make helm-test-schema-template
```

**What is tested:**

- Chart templates render without errors
- Values are properly interpolated
- Conditional logic works correctly
- Required fields are present

### Kubeconform Validation Tests

These tests verify that rendered templates produce valid Kubernetes manifests:

```bash
# Run kubeconform validation for a specific test
make tests/helm/schema/components.webhookServer.backfill.schedule.valid3.pass-kubeconform

# Run kubeconform validation for all tests
make helm-test-schema-kubeconform
```

**What is tested:**

- Generated manifests conform to Kubernetes API schema
- Resource types are valid
- Field types and formats are correct
- Required Kubernetes fields are present

### Combined Validation Tests

These tests run both template and kubeconform validation:

```bash
# Run both validations for a specific test
make tests/helm/schema/aggregator.database.compressionLevel.valid.pass

# Run all schema validation tests
make helm-test-schema
```

## Test Configuration Examples

### Valid Configuration Tests

**Database Compression Level:**

```yaml
# aggregator.database.compressionLevel.valid.pass.yaml
aggregator:
  database:
    compressionLevel: 6
```

**Webhook Backfill Schedule:**

```yaml
# components.webhookServer.backfill.schedule.valid3.pass.yaml
components:
  webhookServer:
    backfill:
      schedule: "0 2 * * *"
```

### Invalid Configuration Tests

**Invalid Image Format:**

```yaml
# image.digest.invalid.fail.yaml
image:
  digest: "not-a-valid-digest"
```

**Empty Cloud Account ID:**

```yaml
# cloudAccountId.empty-quotes.fail.yaml
cloudAccountId: ""
```

## Running Schema Tests

### Individual Test Execution

```bash
# Run specific test (both template and kubeconform)
make tests/helm/schema/aggregator.image.valid.pass

# Run only template validation
make tests/helm/schema/aggregator.image.valid.pass-template

# Run only kubeconform validation
make tests/helm/schema/aggregator.image.valid.pass-kubeconform

# Run a failure test (template validation only)
make tests/helm/schema/image.digest.invalid.fail
```

### Bulk Test Execution

```bash
# Run all schema tests
make helm-test-schema

# Run only template validation tests
make helm-test-schema-template

# Run only kubeconform validation tests
make helm-test-schema-kubeconform
```

### Test Filtering

```bash
# Run tests for specific component
make helm-test-schema | grep aggregator

# Run only pass/fail tests
find tests/helm/schema -name "*.pass.yaml" -exec basename {} \; | while read test; do
  make "tests/helm/schema/${test%.yaml}"
done
```

## Test Data Structure

Each test file contains Helm values that override the chart defaults:

```yaml
# Example test file structure
global:
  # Global overrides

aggregator:
  # Component-specific configuration
  database:
    compressionLevel: 6

components:
  # Component enablement and configuration
  webhookServer:
    enabled: true
    backfill:
      schedule: "0 2 * * *"
```

## Expected Test Outcomes

### Passing Tests (.pass files)

- Template rendering succeeds without errors
- Generated manifests pass kubeconform validation
- All Kubernetes resources are valid
- Values are properly applied to templates

### Failing Tests (.fail files)

- Template rendering fails with appropriate error messages
- Invalid configurations are rejected
- Schema validation catches configuration errors
- Error messages are informative and actionable

## Common Test Scenarios

### Configuration Validation

- **Required fields** - Tests that required values are present
- **Field types** - Tests that values are correct types (string, int, bool)
- **Value ranges** - Tests that numeric values are within valid ranges
- **Format validation** - Tests that strings match expected formats (URLs, cron expressions)

### Component Integration

- **Component dependencies** - Tests that dependent components are properly configured
- **Resource references** - Tests that components reference each other correctly
- **Conditional rendering** - Tests that components render only when enabled

### Edge Cases

- **Empty values** - Tests behavior with empty or null values
- **Extreme values** - Tests with very large or very small values
- **Special characters** - Tests with special characters in strings
- **Unicode handling** - Tests with international characters

## Adding New Schema Tests

### Creating Test Files

1. **Choose descriptive name** following the naming convention
2. **Create YAML file** with test values
3. **Include only relevant overrides** - don't repeat chart defaults
4. **Test both positive and negative cases**

Example new test:

```yaml
# components.insightsController.replicas.high.pass.yaml
components:
  insightsController:
    enabled: true
    replicas: 10
```

### Test Development Process

1. **Write test file** with specific configuration
2. **Run template validation** to check rendering
3. **Run kubeconform validation** to verify Kubernetes compliance
4. **Verify expected outcome** (pass or fail)
5. **Document any special requirements**

### Best Practices

- **One concept per test** - Each test should focus on a single configuration aspect
- **Clear naming** - Test names should clearly indicate what is being tested
- **Comprehensive coverage** - Include tests for all major configuration options
- **Error scenarios** - Include tests that verify error handling
- **Documentation** - Comment complex test configurations

## Troubleshooting Schema Tests

### Template Rendering Failures

```bash
# Debug template rendering
helm template test-release ../../helm/chart --values tests/helm/schema/your-test.yaml --debug

# Check for syntax errors
helm lint ../../helm/chart --values tests/helm/schema/your-test.yaml
```

### Kubeconform Failures

```bash
# Generate manifest and validate manually
helm template test-release ../../helm/chart --values tests/helm/schema/your-test.yaml > output.yaml
kubeconform output.yaml
```

### Common Issues

- **Missing required fields** - Ensure all required Helm values are provided
- **Invalid value types** - Check that string/int/bool types match schema
- **Template syntax errors** - Verify Go template syntax in chart templates
- **API version mismatches** - Ensure Kubernetes API versions are correct

Schema validation tests ensure the CloudZero Agent Helm chart is robust and handles various configuration scenarios correctly.
