# Backfiller Integration Tests

This directory contains minimal integration tests for the CloudZero Agent backfiller functionality. The tests verify that the backfiller can correctly discover Kubernetes namespace resources and send them to the collector in the proper Prometheus RemoteWrite format.

## Test Components

### 1. **Kind Cluster Setup** (`kind-config.yaml`)

- Creates a minimal single-node Kind cluster named `cloudzero-backfiller-test`
- Includes volume mounts for test data exchange

### 2. **Test Resources** (`test-namespaces.yaml`)

- Creates 4 test namespaces with various labels and annotations:
  - `production` - with `environment=production`, `team=backend`, `cost-center=engineering`
  - `staging` - with `environment=staging`, `team=frontend`, `cost-center=engineering`
  - `development` - with `environment=development`, `team=devops`, `cost-center=operations`
  - `test-exclude` - with `exclude-from-monitoring=true` (should be filtered out)

### 3. **Mock Collector** (`mock_collector.go`)

- HTTP server that mimics the CloudZero collector API
- Captures Prometheus RemoteWrite requests (protobuf + snappy compression)
- Validates received metrics and saves them to files for inspection
- Provides methods to verify namespace-specific metrics

### 4. **Integration Test** (`backfiller_integration_test.go`)

- End-to-end test that:
  - Sets up Kind cluster
  - Applies test namespaces
  - Starts mock collector
  - Runs backfiller against the cluster
  - Validates results

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

Kind (Kubernetes in Docker) uses your Docker Desktop to create Kubernetes clusters inside Docker containers. The `kind-config.yaml` file is just a configuration file that tells Kind how to set up the cluster.

**Setup Steps:**

1. **Ensure Docker Desktop is Running**

   - Make sure Docker Desktop is running on your machine
   - Kind will use Docker Desktop's Docker daemon

2. **Verify Docker Desktop Setup**

   ```bash
   # Check Docker is running
   docker version

   # Check if Docker Desktop has Kubernetes enabled (optional)
   docker info | grep -i kubernetes
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

**Note:** You don't need to manually create a Kind cluster - the test handles that automatically!

## Running the Tests

### Quick Test

```bash
cd tests/backfiller
make test-backfiller
```

### Debug Mode (keeps cluster running)

```bash
cd tests/backfiller
make test-backfiller-debug
```

**Note:** The first run may take 5-10 minutes to download the Kind node image. Subsequent runs will be much faster.

### Check Test Status

```bash
cd tests/backfiller
make test-backfiller-status
```

### Manual Cleanup

```bash
cd tests/backfiller
make test-backfiller-cleanup
```

## Test Configuration

The test uses a configuration (`test-config.yaml`) that:

- **Enables label filtering** for: `environment`, `team`, `cost-center`
- **Enables annotation filtering** for: `deployment.kubernetes.io/managed-by`, `description`
- **Focuses on namespaces only** (pods, deployments, etc. are disabled)
- **Uses localhost:8080** as the collector endpoint

## Expected Results

The test validates:

1. **Resource Discovery**: Backfiller discovers all 4 test namespaces
2. **Label Filtering**: Only configured labels are captured
3. **Annotation Filtering**: Only configured annotations are captured
4. **Protobuf Format**: Metrics are sent in proper Prometheus RemoteWrite format
5. **Collector Integration**: Mock collector receives and parses the data correctly

## Test Output

The test generates several outputs:

1. **Console logs** showing test progress and results
2. **Metric files** in `/tmp/cloudzero-backfiller-test-*` directories
3. **Detailed validation** of expected vs actual namespaces found

## How the Test Works

The test creates everything automatically:

1. **Creates the Kind cluster** using the `kind-config.yaml` file:

   ```bash
   kind create cluster --config kind-config.yaml --wait 60s
   ```

2. **Applies the test namespaces** to the cluster:

   ```bash
   kubectl apply -f test-namespaces.yaml
   ```

3. **Runs the backfiller** against the cluster

4. **Validates the results**

5. **Cleans up** (unless you use debug mode)

## Manual Kind Cluster Operations

If you want to manually work with Kind clusters:

```bash
# Create a cluster manually
kind create cluster --name cloudzero-backfiller-test --config tests/backfiller/kind-config.yaml

# List clusters
kind get clusters

# Get kubeconfig
kind get kubeconfig --name cloudzero-backfiller-test

# Use kubectl with the cluster
kubectl cluster-info --context kind-cloudzero-backfiller-test

# Delete the cluster
kind delete cluster --name cloudzero-backfiller-test
```

## Troubleshooting

### Common Issues

1. **Kind cluster creation fails**:

   - Check if Docker Desktop is running
   - Ensure Kind is properly installed
   - Check if port 8080 is available

2. **Test timeout**:

   - Increase timeout in test configuration
   - Check if namespaces are created successfully: `kubectl get namespaces`

3. **No metrics received**:
   - Check mock collector logs
   - Verify backfiller configuration
   - Ensure Kind cluster is accessible

### Debugging

1. **Keep cluster running**:

   ```bash
   make test-backfiller-debug
   ```

2. **Check cluster resources**:

   ```bash
   kubectl --kubeconfig /tmp/kubeconfig get namespaces
   kubectl --kubeconfig /tmp/kubeconfig describe namespace production
   ```

3. **View mock collector output**:
   ```bash
   find /tmp -name "cloudzero-backfiller-test-*" -type d
   ls -la /tmp/cloudzero-backfiller-test-*/received_metrics_*.json
   ```

## What This Test Validates

✅ **Namespace Discovery**: Confirms backfiller finds all Kubernetes namespaces  
✅ **Label Filtering**: Verifies only configured labels are captured  
✅ **Annotation Filtering**: Verifies only configured annotations are captured  
✅ **Protobuf Format**: Validates proper Prometheus RemoteWrite format  
✅ **End-to-End Pipeline**: Tests complete flow from K8s API → Backfiller → Collector

## Extending the Tests

To extend these tests:

1. **Add more resource types**: Modify `test-namespaces.yaml` to include pods, deployments, etc.
2. **Test different filters**: Update `test-config.yaml` with different label/annotation patterns
3. **Add negative tests**: Create resources that should be filtered out
4. **Performance testing**: Add many namespaces to test pagination and worker pools

## Files Structure

```
tests/backfiller/
├── README.md                          # This file
├── Makefile                           # Test automation
├── kind-config.yaml                   # Kind cluster configuration
├── test-namespaces.yaml               # Test Kubernetes resources
├── test-config.yaml                   # Backfiller configuration
├── mock_collector.go                  # Mock collector implementation
├── backfiller_integration_test.go     # Main integration test
└── [generated files]                  # Test outputs and temporary files
```
