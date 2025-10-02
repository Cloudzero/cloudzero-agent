# KUTTL (Kubernetes Test Tool) Tests

This directory contains declarative end-to-end tests using KUTTL (Kubernetes Test Tool). KUTTL provides a way to write Kubernetes tests as YAML manifests, testing real resource behavior and interactions in a Kubernetes cluster.

## Overview

KUTTL tests validate the CloudZero Agent's behavior in actual Kubernetes environments by:

- Creating real Kubernetes resources
- Waiting for expected states and conditions
- Validating resource properties and relationships
- Testing multi-step workflows and scenarios

## Test Structure

Each test directory contains:

- **`kuttl-test.yaml`** - Test configuration and settings
- **`steps/`** directory - Sequential test steps as YAML files
- **Supporting files** - Additional manifests and configurations

### Test Suites

#### Collector Tests

- **`collector-test/`** - Basic collector functionality testing
- **`collector-comprehensive-test/`** - Extended collector validation

#### Webhook Tests

- **`webhook-test/`** - Basic webhook functionality testing
- **`webhook-comprehensive-test/`** - Extended webhook validation

## What KUTTL Tests Validate

### Resource Lifecycle

- **Creation** - Resources are created correctly
- **Updates** - Resources respond to configuration changes
- **Deletion** - Cleanup happens properly
- **Status conditions** - Resources reach expected states

### Component Integration

- **Collector behavior** - Metric collection and storage
- **Webhook functionality** - Admission webhook processing
- **Service interactions** - Components communicate correctly
- **Configuration handling** - Settings are applied properly

### Kubernetes Integration

- **RBAC permissions** - Service accounts have correct permissions
- **Networking** - Services and endpoints work correctly
- **Storage** - Persistent volumes and claims function
- **Monitoring** - Metrics and health checks work

## Prerequisites

### Required Infrastructure

- **Kubernetes cluster** - KIND cluster or real cluster
- **KUTTL CLI** - Installed via `make install-tools`
- **kubectl** - Configured for the target cluster
- **Deployed agent** - CloudZero Agent installed via Helm

### Tool Installation

```bash
# Install KUTTL and other required tools
make install-tools

# Verify KUTTL installation
kubectl kuttl version
```

## How to Run KUTTL Tests

**IMPORTANT**: KUTTL tests ARE integrated with the main project Makefile.

### Complete Test Workflow

```bash
# Full test cycle: create cluster, install chart, run tests, cleanup
make kind-test
```

This runs:

1. Creates KIND cluster (`kind-up`)
2. Installs Helm chart (`helm-install-current`)
3. Runs KUTTL tests (`helm-test-kuttl`)
4. Uninstalls chart (`helm-uninstall`)
5. Deletes cluster (`kind-down`)

### Run Tests on Existing Cluster

#### All KUTTL Tests

```bash
# Run all KUTTL test suites (assumes cluster exists with agent deployed)
make helm-test-kuttl
```

#### Individual Test Suites via Makefile

```bash
# Run specific test suite via Makefile pattern
make tests/kuttl/collector-test/run
make tests/kuttl/collector-comprehensive-test/run
make tests/kuttl/webhook-test/run
make tests/kuttl/webhook-comprehensive-test/run
```

#### Individual Test Suites via kubectl kuttl

```bash
# Run individual test (from project root)
kubectl kuttl test tests/kuttl/collector-test/

# Run with specific cluster name (affects kubeconfig and namespace)
CLUSTER_NAME=my-cluster kubectl kuttl test tests/kuttl/webhook-test/

# Run with specific kubeconfig
KUBECONFIG=tests/kuttl/kubeconfig kubectl kuttl test tests/kuttl/webhook-test/

# Run with verbose output
kubectl kuttl test tests/kuttl/collector-comprehensive-test/ --config tests/kuttl/collector-comprehensive-test/kuttl-test.yaml -v 1
```

## Test Configuration

### KUTTL Test Configuration (`kuttl-test.yaml`)

Each test suite has its own configuration:

```yaml
apiVersion: kuttl.dev/v1beta1
kind: TestSuite
metadata:
  name: collector-test
spec:
  startControllers: false
  skipDelete: false
  timeout: 300 # 5 minutes
  # Additional test-specific settings
```

### Cluster Configuration

Tests use the cluster configuration system for consistent setup:

- **Kubeconfig path**: `tests/kuttl/kubeconfig`
- **Namespace**: Determined by cluster config
- **Timeout**: Configurable per test suite

#### Cluster Selection with CLUSTER_NAME

The `CLUSTER_NAME` variable allows you to specify which cluster to run KUTTL tests against:

```bash
# Run tests against specific cluster
CLUSTER_NAME=my-test-cluster make helm-test-kuttl

# Run individual test suite against specific cluster
CLUSTER_NAME=staging-cluster make tests/kuttl/collector-test/run

# Run complete KIND workflow with specific cluster name
CLUSTER_NAME=custom-kind-cluster make kind-test
```

**Default behavior:**

- If `CLUSTER_NAME` is not set, tests use the default cluster configuration
- The variable is used for both cluster creation (KIND) and test execution
- Cluster name affects namespace selection and resource naming

**Common cluster patterns:**

```bash
# Development testing
CLUSTER_NAME=dev-cluster make helm-test-kuttl

# Staging environment
CLUSTER_NAME=staging make tests/kuttl/webhook-comprehensive-test/run

# Feature branch testing
CLUSTER_NAME=feature-$(git branch --show-current) make kind-test

# CI/CD pipeline usage
CLUSTER_NAME=${CI_PIPELINE_ID}-test make helm-test-kuttl
```

## Test Examples

### Collector Test Structure

```text
tests/kuttl/collector-test/
├── kuttl-test.yaml              # Test configuration
└── steps/
    ├── 01-create-metric-resources.yaml    # Create test resources
    ├── 02-verify-collector-logs.yaml      # Check collector behavior
    └── 03-test-collector-metrics.yaml     # Validate metrics collection
```

### Webhook Test Structure

```text
tests/kuttl/webhook-test/
├── kuttl-test.yaml              # Test configuration
└── steps/
    ├── 01-create-test-resources.yaml      # Create resources that trigger webhook
    ├── 02-verify-webhook-response.yaml    # Check webhook processing
    └── 03-validate-metrics.yaml           # Verify webhook metrics
```

## Test Step Patterns

### Resource Creation Step

```yaml
# 01-create-resources.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
  namespace: default
spec:
  replicas: 1
  # ... deployment spec
---
# Expected state after step
apiVersion: kuttl.dev/v1beta1
kind: TestStep
metadata:
  name: create-resources
```

### Validation Step

```yaml
# 02-verify-behavior.yaml
apiVersion: kuttl.dev/v1beta1
kind: TestAssert
metadata:
  name: verify-deployment-ready
spec:
  timeout: 60
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
  namespace: default
status:
  readyReplicas: 1
  # Additional assertions
```

### Cleanup Step

```yaml
# 03-cleanup.yaml
apiVersion: kuttl.dev/v1beta1
kind: TestStep
metadata:
  name: cleanup-resources
---
# Delete resources
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
  namespace: default
$delete: true
```

## Common Test Scenarios

### Collector Testing

1. **Resource Creation** - Create pods, services, deployments
2. **Metric Collection** - Verify collector gathers resource metadata
3. **Storage Validation** - Check metrics are stored correctly
4. **Performance Testing** - Test with multiple resources

### Webhook Testing

1. **Admission Processing** - Create resources that trigger webhook
2. **Response Validation** - Verify webhook allows/denies correctly
3. **Metrics Generation** - Check webhook metrics are recorded
4. **Error Handling** - Test webhook with invalid resources

### Integration Testing

1. **Component Interaction** - Test collector and webhook together
2. **Configuration Changes** - Test reconfiguration scenarios
3. **Failure Scenarios** - Test behavior during failures
4. **Upgrade Testing** - Test during chart upgrades

## Troubleshooting

### Common Issues

1. **Test timeouts**:

   ```bash
   # Increase timeout in kuttl-test.yaml
   spec:
     timeout: 600  # 10 minutes
   ```

2. **Resource not found**:

   ```bash
   # Check if resources exist
   kubectl get all -n <namespace>

   # Check RBAC permissions
   kubectl auth can-i create pods --as=system:serviceaccount:<namespace>:<service-account>
   ```

3. **Assertions fail**:

   ```bash
   # Check actual vs expected state
   kubectl describe <resource> <name>

   # Review test logs
   kubectl kuttl test --config kuttl-test.yaml -v 2
   ```

4. **KUTTL not found**:

   ```bash
   # Reinstall tools
   make install-tools

   # Check PATH
   which kubectl-kuttl
   ```

### Debug Tips

```bash
# Run tests with verbose output
kubectl kuttl test tests/kuttl/collector-test/ -v 2

# Keep resources after test failure
kubectl kuttl test tests/kuttl/webhook-test/ --skip-delete

# Use specific kubeconfig
KUBECONFIG=./my-kubeconfig kubectl kuttl test tests/kuttl/

# Run single test step
kubectl apply -f tests/kuttl/collector-test/steps/01-create-resources.yaml
```

## Test Development

### Adding New Test Suites

1. **Create directory structure**:

   ```text
   tests/kuttl/my-new-test/
   ├── kuttl-test.yaml
   └── steps/
       ├── 01-setup.yaml
       ├── 02-test-action.yaml
       └── 03-verify-result.yaml
   ```

2. **Write test configuration**:

   ```yaml
   # kuttl-test.yaml
   apiVersion: kuttl.dev/v1beta1
   kind: TestSuite
   metadata:
     name: my-new-test
   spec:
     timeout: 300
     skipDelete: false
   ```

3. **Create test steps** following KUTTL patterns

4. **Add to Makefile** (automatic discovery via pattern matching)

### Best Practices

- **Sequential steps** - Use numbered prefixes (01-, 02-, 03-)
- **Clear names** - Descriptive step file names
- **Appropriate timeouts** - Set realistic timeouts for each step
- **Cleanup** - Always clean up test resources
- **Assertions** - Verify expected states explicitly
- **Documentation** - Comment complex test logic

### Test Patterns

```yaml
# Create and wait pattern
---
apiVersion: v1
kind: Pod
# ... pod spec
---
apiVersion: kuttl.dev/v1beta1
kind: TestAssert
timeout: 60
---
apiVersion: v1
kind: Pod
status:
  phase: Running
```

```yaml
# Delete and verify pattern
---
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
$delete: true
---
apiVersion: kuttl.dev/v1beta1
kind: TestAssert
timeout: 30
---
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
$expect: NotFound
```

KUTTL tests provide comprehensive validation of the CloudZero Agent in real Kubernetes environments, ensuring reliable operation across different scenarios and configurations.
