# Helm Subchart Tests

This directory contains tests that verify the CloudZero Agent Helm chart functions correctly when used as a subchart dependency in parent charts. These tests ensure proper value inheritance and configuration when the agent is deployed as part of larger Helm umbrella charts.

## Overview

Subchart tests validate:

- **Global value inheritance** - Global values from parent charts are properly passed down
- **Subchart configuration** - Agent-specific values work correctly when nested
- **Dependency management** - Chart dependencies resolve and install correctly
- **Value overrides** - Parent chart values can override subchart defaults
- **Isolation** - Subcharts don't interfere with parent chart resources

## Test Structure

### Parent Chart Setup

Each test uses a parent chart that includes the CloudZero Agent as a dependency:

```yaml
# tests/helm/subchart/basic/chart/Chart.yaml
dependencies:
  - name: cloudzero-agent
    version: ">=0.0.0-0"
    repository: "file://../../../../../helm"
```

### Test Files

Tests follow the same naming convention as schema tests:

- `global.subchart.pass.yaml` - Valid global configuration that should pass
- `global.empty.subchart.pass.yaml` - Empty global values that should pass
- `global.string.subchart.fail.yaml` - Invalid global configuration that should fail

## Available Subchart Tests

### Basic Subchart Tests (`basic/`)

Tests fundamental subchart functionality:

#### Global Value Inheritance (`global.subchart.pass.yaml`)

Tests that global values are properly inherited:

```yaml
# Global values available to all subcharts
global:
  environment: "test"
  region: "us-east-1"
  team: "platform"

# Agent-specific configuration
cloudzero-agent:
  cloudAccountId: "123456789"
  clusterName: "test-cluster"
  region: "us-east-1"
  host: "api.cloudzero.com"
  apiKey: "test-key"
```

**What it tests:**

- Global values are accessible to the agent subchart
- Agent-specific values override global defaults
- Both global and subchart values coexist correctly

#### Empty Global Values (`global.empty.subchart.pass.yaml`)

Tests behavior with minimal global configuration:

**What it tests:**

- Subchart functions without global values
- Default values work correctly in subchart context
- No required global dependencies

#### Invalid Global Configuration (`global.string.subchart.fail.yaml`)

Tests error handling with invalid global values:

**What it tests:**

- Invalid global value types are rejected
- Error messages are appropriate for subchart context
- Validation works correctly in nested scenarios

## Running Subchart Tests

### All Subchart Tests

```bash
# Run all subchart validation tests
make helm-test-subchart
```

### Individual Subchart Tests

```bash
# Run specific subchart test
make tests/helm/subchart/basic/global.subchart.pass

# Run failure test
make tests/helm/subchart/basic/global.string.subchart.fail
```

### Manual Testing

```bash
# Test subchart dependency resolution
cd tests/helm/subchart/basic/chart
helm dependency update
helm template test-release . --values ../global.subchart.pass.yaml
```

## Subchart Configuration Patterns

### Global Value Usage

Global values provide shared configuration across multiple subcharts:

```yaml
global:
  # Environment metadata
  environment: "production"
  region: "us-west-2"
  team: "platform"

  # Common configuration
  monitoring:
    enabled: true
  security:
    networkPolicies: true

# Subchart-specific overrides
cloudzero-agent:
  cloudAccountId: "123456789"
  # Global region is inherited unless overridden
  region: "us-east-1" # Overrides global.region
```

### Value Precedence

Values are applied in this order (highest precedence first):

1. **Subchart-specific values** - `cloudzero-agent.key`
2. **Global values** - `global.key`
3. **Chart defaults** - Default values from agent chart

### Dependency Management

Parent charts declare the agent as a dependency:

```yaml
# Parent Chart.yaml
dependencies:
  - name: cloudzero-agent
    version: "^1.0.0"
    repository: "https://charts.cloudzero.com"
    condition: cloudzero-agent.enabled
    tags:
      - monitoring
      - cloudzero
```

## Common Subchart Scenarios

### Multi-Environment Deployment

```yaml
# staging-values.yaml
global:
  environment: "staging"
  region: "us-west-2"

cloudzero-agent:
  enabled: true
  cloudAccountId: "staging-account"
  clusterName: "staging-cluster"

# production-values.yaml
global:
  environment: "production"
  region: "us-east-1"

cloudzero-agent:
  enabled: true
  cloudAccountId: "prod-account"
  clusterName: "prod-cluster"
  replicas: 3  # Production scaling
```

### Platform Integration

```yaml
# platform-chart values
global:
  platform:
    version: "2.1.0"
    monitoring: true
    security: "strict"

# Multiple monitoring subcharts
prometheus:
  enabled: true
  global: "{{ .Values.global }}"

grafana:
  enabled: true
  global: "{{ .Values.global }}"

cloudzero-agent:
  enabled: true
  # Inherits global.platform.monitoring setting
```

### Conditional Deployment

```yaml
# Conditional subchart activation
global:
  features:
    costAnalysis: true
    securityMonitoring: false

cloudzero-agent:
  enabled: "{{ .Values.global.features.costAnalysis }}"
  components:
    webhookServer:
      enabled: "{{ .Values.global.features.securityMonitoring }}"
```

## Subchart Development Workflow

### Adding New Subchart Tests

1. **Create test values** - Add new YAML file with test configuration
2. **Test locally** - Run `helm template` to verify rendering
3. **Validate output** - Ensure generated manifests are correct
4. **Run test suite** - Execute `make helm-test-subchart`
5. **Document scenarios** - Update README with test purpose

### Testing Parent Chart Integration

```bash
# Create test parent chart
mkdir -p test-parent/templates
cat > test-parent/Chart.yaml << EOF
apiVersion: v2
name: test-parent
version: 0.1.0
dependencies:
  - name: cloudzero-agent
    version: ">=0.0.0-0"
    repository: "file://../../helm"
EOF

# Update dependencies
cd test-parent && helm dependency update

# Test with values
helm template test-release . --values ../subchart-test-values.yaml
```
