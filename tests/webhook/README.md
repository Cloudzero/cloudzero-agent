# Webhook Integration Tests

This directory contains integration tests for the CloudZero Agent webhook functionality. The tests verify that the webhook resource name configuration correctly supports the expected Kubernetes resource types and that the webhook controller properly recognizes plural resource names.

## Test Components

### 1. **Kind Cluster Setup** (`kind-config.yaml`)

- Creates a minimal single-node Kind cluster named `cloudzero-webhook-test`
- Includes volume mounts for test data exchange

### 2. **Basic Integration Test** (`webhook_integration_test.go`)

- Resource validation test that:
  - Sets up Kind cluster
  - Creates webhook controller with test configuration
  - Validates supported resource types match webhook configuration
  - Tests that plural resource names are properly recognized

### 3. **Helm Chart Integration Test** (`webhook_chart_integration_test.go`)

- End-to-end webhook deployment test that:
  - Deploys actual CloudZero Helm chart to Kind cluster
  - Enables webhook with real TLS certificates and ValidatingWebhookConfiguration
  - Creates test Kubernetes resources to trigger webhook invocations
  - Validates webhook receives admission reviews via Prometheus `/metrics` endpoint
  - **Uses kubectl port-forward to access webhook metrics endpoint for validation**
  - **Proves the webhook resource name fix works in practice**

## What This Test Validates

This test specifically validates the **webhook resource name fix** from [PR #371](https://github.com/Cloudzero/cloudzero-agent/pull/371):

✅ **Resource Name Validation**: Confirms webhook controller supports all 18+ resource types from the configuration  
✅ **Plural Form Recognition**: Verifies webhook recognizes correct plural resource names  
✅ **API Group Support**: Tests support across different Kubernetes API groups  
✅ **Version Support**: Validates support for multiple API versions

### Supported Resources Tested

The test validates webhook support for these resource types (with correct plural forms):

**Core API Group (`/v1`)**:

- `pods`, `namespaces`, `nodes`, `services`
- `persistentvolumes`, `persistentvolumeclaims`

**Apps API Group (`apps/v1`)**:

- `deployments`, `statefulsets`, `daemonsets`, `replicasets`

**Batch API Group (`batch/v1`)**:

- `jobs`, `cronjobs`

**Storage API Group (`storage.k8s.io/v1`)**:

- `storageclasses`

**Networking API Group (`networking.k8s.io/v1`)**:

- `ingresses`, `ingressclasses`

**API Extensions Group (`apiextensions.k8s.io/v1`)**:

- `customresourcedefinitions`

**Gateway API Group (`gateway.networking.k8s.io/v1`)**:

- `gateways`, `gatewayclasses`

## Prerequisites

### Install Required Tools

```bash
# Install Kind on macOS with Homebrew
brew install kind

# Or install Kind using Go
go install sigs.k8s.io/kind@latest

# Install kubectl
# Follow instructions at https://kubernetes.io/docs/tasks/tools/install-kubectl/

# Verify installation
kind version
kubectl version --client
```

### Docker Desktop Setup

Kind (Kubernetes in Docker) uses your Docker Desktop to create Kubernetes clusters inside Docker containers.

**Setup Steps:**

1. **Ensure Docker Desktop is Running**

   - Make sure Docker Desktop is running on your machine
   - Kind will use Docker Desktop's Docker daemon

2. **Verify Docker Desktop Setup**

   ```bash
   # Check Docker is running
   docker version
   ```

3. **Test Kind Installation**

   ```bash
   # Create a test cluster
   kind create cluster --name test-cluster

   # Verify it's working
   kubectl cluster-info --context kind-test-cluster

   # Clean up
   kind delete cluster --name test-cluster
   ```

## Running the Tests

### Webhook Integration Test (Requires API Key)

This test deploys the actual Helm chart and validates real webhook invocations:

```bash
# Set API key (required)
export CLOUDZERO_DEV_API_KEY="your-api-key"
# or
export CZ_DEV_API_TOKEN="your-api-key"

# Run webhook test
cd tests/webhook
make test-webhook
```

### Debug Mode (keeps cluster running)

```bash
cd tests/webhook
make test-webhook-debug
```

This keeps the cluster and chart deployment running for debugging. You can then:

```bash
# Access webhook metrics
kubectl port-forward -n cz-webhook-test svc/webhook-chart-test-cloudzero-agent 8080:8080

# View metrics in browser
open http://localhost:8080/metrics

# Look for webhook metrics like:
# webhook_types_total{kind_resource="deployments",operation="CREATE"} 1
# webhook_types_total{kind_resource="services",operation="CREATE"} 1
# webhook_types_total{kind_resource="namespaces",operation="CREATE"} 1

# Clean up when done
make test-webhook-cleanup
```

**Note:** The test automatically uses kubectl port-forward to access the webhook metrics endpoint and validate that webhook invocations are recorded correctly.

**Note:** The first run may take 5-10 minutes to download the Kind node image. Subsequent runs will be much faster.

### Check Test Status

```bash
cd tests/webhook
make test-webhook-status
```

### Manual Cleanup

```bash
cd tests/webhook
make test-webhook-cleanup
```

## Test Configuration

The test creates a minimal webhook configuration that:

- **Tests resource recognition** for all 18+ resource types from the webhook configuration
- **Validates API group support** across core, apps, batch, storage, networking, and gateway APIs
- **Verifies plural form handling** to ensure the webhook resource name fix works correctly
- **Uses debug logging** to provide detailed output about supported resources

## Expected Results

The test validates:

1. **Resource Recognition**: Webhook controller recognizes all expected resource types
2. **Plural Form Support**: Resources are identified using correct plural names (not singular)
3. **API Group Coverage**: Support across all required Kubernetes API groups
4. **Multi-Version Support**: Resources supported across different API versions

## Test Output

The test generates detailed console output showing:

1. **Supported Resources Map**: Complete list of all resources the webhook controller supports
2. **Resource Validation**: Per-resource validation results
3. **API Group Analysis**: Breakdown by API group and version
4. **Debug Information**: Detailed logging for troubleshooting

Example output:

```
=== COMPLETE SUPPORTED RESOURCES MAP ===
Group: apps
  Version: v1
    Kind: deployment
    Kind: statefulset
    Kind: daemonset
    Kind: replicaset
Group:
  Version: v1
    Kind: pod
    Kind: namespace
    Kind: node
    Kind: service
    Kind: persistentvolume
    Kind: persistentvolumeclaim
...
```

## How the Test Works

The test creates everything automatically:

1. **Creates the Kind cluster** using the `kind-config.yaml` file
2. **Initializes webhook controller** with test configuration
3. **Queries supported resources** from the webhook controller
4. **Validates resource support** against expected configuration
5. **Cleans up** (unless you use debug mode)

## Background: The Resource Name Fix

This test was created to validate the fix for a critical webhook configuration issue:

**Problem**: The webhook configuration in `helm/templates/webhook-validating-config.yaml` was using singular resource names (`deployment`, `pod`, `namespace`, etc.) instead of the required plural forms.

**Solution**: Updated all resource names to use correct plural forms (`deployments`, `pods`, `namespaces`, etc.) to match Kubernetes API resource naming.

**Validation**: This integration test confirms that:

- The webhook controller properly supports all the corrected plural resource names
- The resource name fix enables proper webhook interception of Kubernetes operations
- All 18+ resource types from the configuration are working correctly

## Troubleshooting

### Common Issues

1. **Kind cluster creation fails**:

   - Check if Docker Desktop is running
   - Ensure Kind is properly installed
   - Check available disk space

2. **Test timeout**:

   - Increase timeout in test configuration
   - Check if Kind cluster starts successfully

3. **Resource support validation fails**:
   - Review webhook controller implementation
   - Check if API groups are properly registered
   - Verify webhook configuration matches expectations

### Debugging

1. **Keep cluster running**:

   ```bash
   make test-webhook-debug
   ```

2. **Check cluster status**:

   ```bash
   kubectl --context kind-cloudzero-webhook-test cluster-info
   kubectl --context kind-cloudzero-webhook-test get nodes
   ```

3. **Manual testing**:
   ```bash
   # After test creates cluster, you can interact with it
   kubectl --context kind-cloudzero-webhook-test get all --all-namespaces
   ```

## Extending the Tests

To extend these tests:

1. **Add webhook deployment**: Deploy actual webhook admission controller to test end-to-end
2. **Test resource operations**: Create/update/delete resources to trigger webhook calls
3. **Add negative tests**: Test resources that should not be supported
4. **Performance testing**: Test webhook with many simultaneous operations

## Files Structure

```
tests/webhook/
├── README.md                     # This file
├── Makefile                      # Test automation
├── kind-config.yaml              # Kind cluster configuration
├── webhook_integration_test.go   # Main integration test
└── [generated files]             # Test outputs and temporary files
```
