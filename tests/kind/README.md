# KIND (Kubernetes in Docker) Tests

This directory contains utilities and configurations for running Kubernetes tests using KIND (Kubernetes in Docker). KIND provides lightweight, local Kubernetes clusters for testing purposes.

## Overview

KIND tests provide a middle ground between unit tests and full cluster integration tests. They create real Kubernetes clusters in Docker containers, allowing testing of Kubernetes-specific functionality without requiring a full cluster setup.

## Components

### Cluster Management (`cluster.go`)

Provides utilities for KIND cluster lifecycle management:

**What it provides:**

- KIND cluster creation and configuration
- Cluster lifecycle management (create, configure, destroy)
- Kubernetes client setup for testing
- Network and storage configuration for test scenarios

### Test Data (`testdata/`)

Contains configuration files and manifests used in KIND-based tests:

- Cluster configuration files
- Test Kubernetes manifests
- Network and storage configurations
- Example deployments for testing

## Integration with Makefile

KIND tests are integrated with the main project Makefile:

### Available Make Targets

```bash
# Create KIND cluster for testing
make kind-up

# Delete KIND cluster and cleanup
make kind-down

# Complete test workflow: create cluster, install chart, run tests, cleanup
make kind-test

# Run KUTTL tests (assumes cluster is already running)
make helm-test-kuttl
```

### Configuration

```makefile
# Test configuration (can be overridden)
TEST_K8S_VERSION          ?= v1.32.3
TEST_KIND_IMAGE_VERSION   ?= v1.33.2
TEST_PLATFORM             ?= linux/amd64
CLUSTER_NAME              ?= kind
```

## What KIND Tests Enable

### Kubernetes API Testing

- **Real Kubernetes API** - Tests against actual Kubernetes API server
- **Resource creation** - Test actual resource creation, updates, deletion
- **Controller behavior** - Test how controllers respond to resource changes
- **RBAC testing** - Validate permissions and service account configurations

### Helm Chart Testing

- **Chart deployment** - Deploy actual Helm charts to test clusters
- **Value validation** - Test different chart configurations in real environments
- **Upgrade/rollback** - Test chart upgrade and rollback scenarios
- **Resource interaction** - Test how chart resources interact with each other

### Network and Storage Testing

- **Service discovery** - Test Kubernetes networking and DNS
- **Persistent volumes** - Test storage configurations and persistence
- **Ingress testing** - Test ingress controllers and routing
- **Cross-namespace communication** - Test network policies and isolation

## Prerequisites

### Required Software

- **Docker** - KIND runs Kubernetes nodes as Docker containers
- **kubectl** - For interacting with the KIND cluster
- **helm** - For chart deployment testing (installed via `make install-tools`)
- **kind** - KIND CLI tool (installed via `make install-tools`)

### Installation

```bash
# Install all required tools
make install-tools

# Verify installation
kind version
kubectl version --client
docker version
```

## How to Run KIND Tests

**IMPORTANT**: KIND tests ARE integrated with the main project Makefile (unlike other tests in `tests/` directory).

### Basic Workflow

```bash
# Complete test cycle
make kind-test
```

This runs the complete workflow:

1. Creates KIND cluster (`kind-up`)
2. Installs Helm chart with current code (`helm-install-current`)
3. Runs KUTTL tests (`helm-test-kuttl`)
4. Uninstalls chart (`helm-uninstall`)
5. Deletes cluster (`kind-down`)

### Manual Cluster Management

```bash
# Create cluster only
make kind-up

# Run tests against existing cluster
make helm-test-kuttl

# Clean up cluster
make kind-down
```

### Helm Chart Operations on KIND

```bash
# Install chart with current development code
make helm-install-current

# Install chart with specified values
make helm-install

# Wait for chart to be ready
make helm-wait

# Uninstall chart
make helm-uninstall
```

### Custom Configuration

```bash
# Use different Kubernetes version
make kind-test TEST_K8S_VERSION=v1.30.0

# Use different cluster name
make kind-test CLUSTER_NAME=my-test-cluster
```

## Test Configuration Files

### Cluster Configuration

KIND clusters are configured through the cluster configuration system. Default configuration includes:

- Single-node cluster (sufficient for most tests)
- Volume mounts for test data exchange
- Network configuration for service testing
- Resource limits appropriate for local testing

### Kubeconfig Management

```bash
# Kubeconfig is automatically created at:
tests/kuttl/kubeconfig

# Use with kubectl:
export KUBECONFIG=tests/kuttl/kubeconfig
kubectl get nodes
```

## Integration with Other Tests

### KUTTL Tests

KIND clusters are primarily used to run KUTTL (Kubernetes Test Tool) tests:

- Declarative test scenarios
- Resource lifecycle testing
- Multi-step test workflows

### Helm Chart Testing

KIND provides the cluster for Helm chart integration testing:

- Real deployment scenarios
- Resource validation
- Configuration testing

### Webhook Testing

KIND clusters can be used for webhook integration testing (see `tests/webhook/` for webhook-specific tests).

## Common Use Cases

### Chart Development

```bash
# Quick test of chart changes
make kind-up
make helm-install-current
# Make changes to chart
make helm-upgrade
make kind-down
```

### Feature Testing

```bash
# Test new Kubernetes features
make kind-up TEST_K8S_VERSION=v1.33.0
# Run feature-specific tests
make kind-down
```

### CI/CD Integration

```bash
# Automated testing pipeline
make kind-test  # Complete cycle in CI
```

## Troubleshooting

### Common Issues

1. **KIND cluster creation fails**:

   ```bash
   # Check Docker is running
   docker info

   # Clean up any existing clusters
   kind delete cluster --name kind

   # Check disk space
   df -h
   ```

2. **Cluster not ready**:

   ```bash
   # Check cluster status
   kubectl --kubeconfig tests/kuttl/kubeconfig get nodes

   # Wait for cluster to be ready
   kubectl --kubeconfig tests/kuttl/kubeconfig wait --for=condition=Ready nodes --all --timeout=4m
   ```

3. **Network issues**:

   ```bash
   # Check Docker networks
   docker network ls

   # Restart Docker if needed
   # (Docker Desktop -> Restart)
   ```

4. **Resource constraints**:

   ```bash
   # Check Docker resource limits
   # Docker Desktop -> Settings -> Resources

   # Increase memory limit to at least 4GB for Kubernetes
   ```

### Debug Tips

```bash
# Check cluster logs
docker logs <kind-container-name>

# Get cluster info
kubectl --kubeconfig tests/kuttl/kubeconfig cluster-info

# Check pod status in all namespaces
kubectl --kubeconfig tests/kuttl/kubeconfig get pods --all-namespaces

# Describe problematic resources
kubectl --kubeconfig tests/kuttl/kubeconfig describe pod <pod-name>
```

## Performance Considerations

### Resource Usage

- KIND clusters use Docker resources
- Single-node clusters are sufficient for most tests
- Monitor Docker memory and CPU usage during tests

### Test Speed

- Cluster creation takes 1-2 minutes (first time longer for image pull)
- Subsequent tests are faster with cached images
- Consider keeping cluster running during development

### Cleanup

- Always clean up clusters after testing
- Use `make kind-down` or manual cleanup:
  ```bash
  kind delete cluster --name <cluster-name>
  ```

## Best Practices

### Test Design

- Keep tests focused and fast
- Use appropriate resource limits
- Clean up test resources properly
- Test realistic scenarios

### Development Workflow

- Create cluster once, run multiple tests
- Use port-forwarding for debugging
- Save logs and configs for troubleshooting

### CI/CD Integration

- Use complete `make kind-test` cycle
- Ensure proper cleanup in all scenarios
- Set appropriate timeouts for cluster operations

KIND provides an excellent balance between test realism and resource efficiency, enabling comprehensive Kubernetes testing without requiring external cluster infrastructure.
