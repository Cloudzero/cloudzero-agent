# KUTTL Testing Infrastructure

This directory contains comprehensive Kubernetes testing infrastructure using KUTTL (Kubernetes Test Tool) for the CloudZero Agent.

## Overview

KUTTL provides a lightweight, reliable testing framework for Kubernetes applications that:

- Supports both kind and k3s clusters
- Provides step-by-step test execution
- Handles resource cleanup automatically
- Generates detailed test reports
- Avoids Docker Hub rate limiting issues

## Test Structure

### Webhook Tests

#### Basic Webhook Test (`webhook-test/`)

- **Purpose**: Basic webhook functionality validation
- **Resources**: Namespace, Deployment, Service
- **Validation**: Webhook events, metrics endpoint
- **Duration**: ~5-6 seconds

#### Comprehensive Webhook Test (`webhook-comprehensive-test/`)

- **Purpose**: Comprehensive webhook functionality validation
- **Resources**: Namespace, Deployment, Service, StatefulSet, Job
- **Validation**: Health endpoints, webhook metrics, comprehensive resource testing
- **Duration**: ~5-6 seconds

### Collector Tests

#### Basic Collector Test (`collector-test/`)

- **Purpose**: Basic collector functionality validation
- **Resources**: Namespace, Deployment, Service, StatefulSet, Job
- **Validation**: Collector logs, metrics endpoints
- **Duration**: ~5-6 seconds

#### Comprehensive Collector Test (`collector-comprehensive-test/`)

- **Purpose**: Comprehensive collector functionality validation
- **Resources**: Namespace, Deployment, Service, StatefulSet, Job, ConfigMap, Secret
- **Validation**: Collector logs, metric validation, comprehensive resource testing
- **Duration**: ~5-6 seconds

## Test Execution

### Local Testing

```bash
# Run all KUTTL tests
make test-local-start

# Run specific test suites
kubectl-kuttl test --config tests/kuttl/webhook-test/kuttl-test.yaml tests/kuttl/webhook-test/
kubectl-kuttl test --config tests/kuttl/collector-test/kuttl-test.yaml tests/kuttl/collector-test/

# Clean up test resources
make test-local-stop
```

### CI/CD Testing

The GitHub Actions workflows use the same KUTTL infrastructure:

```yaml
# In GitHub Actions
- name: Run KUTTL Tests
  run: |
    export KUBECONFIG=$HOME/.kube/config
    make test-kuttl
```

## Test Configuration

### KUTTL Test Suite Configuration

Each test suite has a `kuttl-test.yaml` configuration:

```yaml
apiVersion: kuttl.dev/v1beta1
kind: TestSuite
metadata:
  name: webhook-test
spec:
  testDirs:
    - ./steps
  timeouts:
    test: 300s
    step: 60s
  parallel: 1
  reportFormat: JSON
  deleteNamespace: true
```

### Test Steps

Each test suite contains step files in the `steps/` directory:

1. **Resource Creation**: Create test resources (namespaces, deployments, services, etc.)
2. **Validation**: Verify functionality (logs, metrics, health checks)
3. **Cleanup**: Automatic cleanup handled by KUTTL

## Migration from TestKube

### What Was Migrated

- **Webhook Tests**: From `tests/webhook/` and `tests/testkube/webhook-comprehensive.yaml`
- **Collector Tests**: From `tests/testkube/collector-validation.yaml`
- **GitHub Action Integration**: Updated to use KUTTL instead of TestKube

### Benefits of Migration

1. **Reliability**: No Docker Hub rate limiting issues
2. **Speed**: Faster test execution (~5-6 seconds per test suite)
3. **Simplicity**: Lighter weight than TestKube
4. **Consistency**: Same testing framework for local and CI/CD
5. **Maintainability**: Easier to understand and modify

## Test Coverage

### Webhook Functionality

- ✅ Resource creation triggers webhook events
- ✅ Webhook metrics collection
- ✅ Health endpoint validation
- ✅ Metrics endpoint accessibility
- ✅ Comprehensive resource types (Deployment, Service, StatefulSet, Job)

### Collector Functionality

- ✅ Metric collection from Kubernetes resources
- ✅ Remote write functionality
- ✅ Specific metric validation (kube_pod_info, kube_pod_labels, etc.)
- ✅ Webhook metric integration
- ✅ Cost metrics shipping progress
- ✅ Comprehensive resource coverage

## Troubleshooting

### Common Issues

1. **YAML Syntax Errors**: Ensure proper escaping in shell commands
2. **Timeout Issues**: Increase timeouts in kuttl-test.yaml if needed
3. **Resource Cleanup**: KUTTL handles cleanup automatically
4. **Cluster Issues**: Use `make test-local-stop` to clean up before retesting

### Debugging

```bash
# Check cluster status
kubectl get pods -A

# Check specific namespace
kubectl get pods -n cz-agent

# View logs
kubectl logs deployment/cz-agent-aggregator -n cz-agent -c cz-agent-aggregator-collector

# Run individual test with verbose output
kubectl-kuttl test --config tests/kuttl/webhook-test/kuttl-test.yaml tests/kuttl/webhook-test/ -v 2
```

## Integration with Makefile

The Makefile provides convenient targets for KUTTL testing:

- `make test-local-start`: Run all KUTTL tests
- `make test-local-stop`: Clean up test resources
- `make test-kuttl`: Run KUTTL tests only (requires existing cluster)

## Future Enhancements

1. **Additional Test Suites**: Add more specialized test scenarios
2. **Performance Testing**: Add load testing capabilities
3. **Security Testing**: Add security-focused test scenarios
4. **Integration Testing**: Add tests for external service integration
5. **Custom Metrics**: Add tests for custom metric collection

## References

- [KUTTL Documentation](https://kuttl.dev/)
- [KUTTL GitHub Repository](https://github.com/kudobuilder/kuttl)
- [Kubernetes Testing Best Practices](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/)
