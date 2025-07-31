# Webhook Load Test Tool

A Go tool for **pre-deployment webhook testing** and **load testing** the always-allow webhook behavior. This tool creates identifiable test resources that can be verified in backfill operations before cleaning up automatically.

## Purpose

**Run this tool BEFORE deploying the CloudZero agent Helm chart** to:

1. **Test webhook validation behavior** - Verify always-allow functionality works correctly
2. **Generate identifiable resources** - Create namespaces and pods that can be tracked in backfill operations
3. **Load test webhook performance** - Stress test with configurable concurrency and resource counts
4. **Validate cleanup** - Ensure resources are properly cleaned up after testing
5. **Pre-deployment verification** - Confirm webhook is working before installing the agent

## Features

- Creates N namespaces with M pods each in parallel using efficient concurrent operations
- All resources tagged with consistent labels including `nsgroup` for easy identification and management
- Automatic cleanup after configurable live duration (default: 30 seconds)
- Built-in validation to ensure all expected resources are created
- Real-time progress reporting across all test phases
- Client-side throttling protection to avoid overwhelming the API server

## Pre-Deployment Workflow

### Step 1: Build the Tool

```bash
cd tests/webhook/load
go build -o webhook-load-test main.go
```

### Step 2: Test Webhook Before Agent Deployment

```bash
# Quick validation test (recommended before deployment)
./webhook-load-test -namespaces 5 -pods-per-ns 2 -live-duration 30s -nsgroup pre-deploy-test

# Medium load test (for performance validation)
./webhook-load-test -namespaces 100 -pods-per-ns 2 -live-duration 60s -batch-size 20 -concurrency 5

# Large-scale load test (comprehensive webhook stress testing)
./webhook-load-test -namespaces 1000 -pods-per-ns 1 -live-duration 300s -batch-size 50 -concurrency 3
```

### Step 3: Monitor Test Resources (Optional)

```bash
# In another terminal, watch the test resources being created/destroyed
kubectl get pods -l nsgroup=pre-deploy-test --all-namespaces -w
```

### Step 4: Deploy CloudZero Agent

```bash
# After successful webhook testing, deploy the agent
helm install cloudzero-agent ./helm/chart/
```

## Usage Examples

```bash
# Default test (100 namespaces, 1 pod each, 30s lifetime)
./webhook-load-test

# Pre-deployment validation test
./webhook-load-test \
  -namespaces 10 \
  -pods-per-ns 2 \
  -live-duration 45s \
  -nsgroup webhook-validation \
  -concurrency 3

# Heavy load test (use lifecycle batching for large scale)
./webhook-load-test \
  -namespaces 500 \
  -pods-per-ns 2 \
  -live-duration 120s \
  -batch-size 25 \
  -concurrency 5 \
  -resource-profile medium \
  -nsgroup load-test

# Custom resource naming
./webhook-load-test \
  -ns-prefix test-namespace \
  -pod-prefix test-pod \
  -nsgroup custom-test

# Use specific kubeconfig
./webhook-load-test -kubeconfig ~/.kube/config
```

## Parameters

- `-namespaces`: Number of namespaces to create (default: 100)
- `-pods-per-ns`: Number of pods per namespace (default: 1)
- `-live-duration`: How long resources should live before cleanup (default: 30s)
- `-concurrency`: Maximum concurrent operations (default: 10, **reduce to 2-5 to avoid throttling**)
- `-batch-size`: Process in waves of this size (default: 0 = all at once, **recommended for large tests**)
- `-resource-profile`: Resource profile for pods: light, medium, heavy, or custom (default: light)
- `-cpu-request`: CPU request (only used with custom profile)
- `-cpu-limit`: CPU limit (only used with custom profile)
- `-memory-request`: Memory request (only used with custom profile)
- `-memory-limit`: Memory limit (only used with custom profile)
- `-nsgroup`: NSGroup label value for resource grouping (default: auto-generated timestamp)
- `-ns-prefix`: Prefix for namespace names (default: webhook-test-ns)
- `-pod-prefix`: Prefix for pod names (default: webhook-test-pod)
- `-kubeconfig`: Path to kubeconfig file (default: uses default config)

### Lifecycle Batch Processing

For large-scale tests (50+ namespaces), use **lifecycle batch processing** for optimal resource management:

```bash
# Large scale test with lifecycle batching
./webhook-load-test -namespaces 1000 -batch-size 50 -live-duration 300s

# Ultra-large test with smaller batches for stability
./webhook-load-test -namespaces 5000 -batch-size 25 -concurrency 3
```

**Lifecycle batch processing benefits:**

- **Constant resource usage** - only batch-size namespaces exist at any time (not all 1000!)
- **Memory efficiency** - dramatically reduces cluster memory usage (~90% reduction)
- **True load simulation** - mimics real-world workload patterns
- **Better failure isolation** - if one batch fails, others continue unaffected
- **Easier monitoring** - clear progress tracking per batch with resource counts
- **Safer testing** - no risk of overwhelming cluster with massive resource spikes

### Resource Profiles

Choose pod resource allocation to simulate different workload types:

| Profile    | CPU Request/Limit | Memory Request/Limit | Use Case                |
| ---------- | ----------------- | -------------------- | ----------------------- |
| **light**  | 10m/20m           | 16Mi/32Mi            | Microservices, sidecars |
| **medium** | 100m/200m         | 128Mi/256Mi          | Standard applications   |
| **heavy**  | 500m/1000m        | 512Mi/1Gi            | Resource-intensive apps |
| **custom** | User-defined      | User-defined         | Specific requirements   |

**Examples:**

```bash
# Light workload simulation
./webhook-load-test -namespaces 100 -resource-profile light

# Heavy workload with batching
./webhook-load-test -namespaces 200 -batch-size 25 -resource-profile heavy

# Custom resource requirements
./webhook-load-test -resource-profile custom -cpu-request 250m -cpu-limit 500m -memory-request 256Mi -memory-limit 512Mi
```

### Performance Guidance

Choose optimal parameters based on your test scale and cluster capacity:

| Test Scale | Namespaces | Pods/NS | Batch Size      | Concurrency | Resource Profile | Example Use Case          |
| ---------- | ---------- | ------- | --------------- | ----------- | ---------------- | ------------------------- |
| **Small**  | 5-25       | 1-2     | 0 (all at once) | 10          | light            | Pre-deployment validation |
| **Medium** | 50-200     | 1-3     | 20-25           | 5-8         | medium           | Performance testing       |
| **Large**  | 500-1000   | 1-2     | 25-50           | 3-5         | medium           | Load testing              |
| **Ultra**  | 1000+      | 1       | 25-50           | 2-3         | light            | Stress testing            |

**Memory considerations:**

- Lifecycle batching reduces cluster memory usage by ~90% during large tests
- Each batch completes its full lifecycle (create → live → delete) before the next batch starts
- With batching: only batch-size resources exist at any time
- Without batching: ALL resources exist simultaneously (potentially thousands)
- Larger batch sizes = faster completion but higher peak memory usage
- Smaller batch sizes = slower completion but more predictable resource usage

### Throttling Recommendations

If you see throttling messages in the output like:

```
I0731 00:59:17.077231   86610 request.go:752] "Waited before sending request" delay="1.048016916s" reason="client-side throttling, not priority and fairness" verb="POST" URL="https://..."
I0731 00:59:27.155697   86610 request.go:752] "Waited before sending request" delay="1.947236667s" reason="client-side throttling, not priority and fairness" verb="DELETE" URL="https://..."
```

**This is normal and expected** with high resource creation rates. These messages indicate the Kubernetes client is automatically slowing down requests to prevent overwhelming the API server. To reduce throttling:

- Use lifecycle batch processing: `-batch-size 25` or `-batch-size 50`
- Use lower concurrency: `-concurrency 2` or `-concurrency 3`
- The tool includes built-in delays between operations to reduce server pressure
- Throttling doesn't affect test success, just increases duration

## Labels

All created resources include these labels:

- `test: always-allow`
- `team: cirrus`
- `purpose: testing`
- `nsgroup: <specified-group>`
- `ns-index: <namespace-number>`
- `pod-index: <pod-number>`
- `resource-profile: <profile-name>` (pods only)

## Test Phases

### Traditional Mode (batch-size = 0)

1. **Phase 1**: Create all namespaces at once
2. **Phase 2**: Create all pods across all namespaces
3. **Phase 2.5**: Validate all expected resources were created successfully
4. **Phase 3**: Wait for specified live duration (ALL resources live together)
5. **Phase 4**: Clean up all resources (delete all namespaces)

### Lifecycle Batch Mode (batch-size > 0)

**Repeats for each batch until all namespaces are processed:**

1. **Phase 1**: Create batch-size namespaces (e.g., namespaces 1-25)
2. **Phase 2**: Create pods for this batch only (e.g., pods in namespaces 1-25)
3. **Phase 3**: Wait for live duration (only this batch lives)
4. **Phase 4**: Clean up this batch (delete these namespaces)
5. **Move to next batch**: Repeat for namespaces 26-50, then 51-75, etc.

### Lifecycle Batch Example

For 100 namespaces with batch-size=25:

```
🔄 Batch 1/4: Processing namespaces 1-25 (25 namespaces)
  Phase 1: Creating 25 namespaces...
  Phase 2: Creating 50 pods... (25 namespaces × 2 pods each)
  Phase 3: Waiting 30s for resources to live...
  Phase 4: Cleaning up batch...
✅ Batch 1/4 completed

🔄 Batch 2/4: Processing namespaces 26-50 (25 namespaces)
  Phase 1: Creating 25 namespaces...
  Phase 2: Creating 50 pods...
  Phase 3: Waiting 30s for resources to live...
  Phase 4: Cleaning up batch...
✅ Batch 2/4 completed

... and so on
```

Only 25 namespaces exist at any given time, not all 100!

## Monitoring

### Real-time Watching

The tool outputs the nsgroup at startup (e.g., `webhook-load-test-1753936486`). Use this for monitoring:

```bash
# Option 1: Watch namespaces and pods separately (recommended)
# Terminal 1: Watch namespaces
kubectl get namespaces -l nsgroup=<nsgroup-from-output> -w

# Terminal 2: Watch pods across all test namespaces
kubectl get pods -l nsgroup=<nsgroup-from-output> --all-namespaces -w

# Option 2: Use custom nsgroup for easier monitoring
./webhook-load-test -nsgroup my-test -namespaces 10 -live-duration 60s

# Then watch with known label:
kubectl get pods -l nsgroup=my-test --all-namespaces -w
```

### Periodic Status Checks

```bash
# Check both resource types at once (no real-time watching)
kubectl get namespaces,pods -l nsgroup=<nsgroup> --all-namespaces

# Use watch command to refresh every 2 seconds
watch -n 2 "kubectl get namespaces,pods -l nsgroup=<nsgroup> --all-namespaces"
```

### Webhook Metrics Monitoring

The webhook exposes Prometheus metrics at `/metrics` that track all admission requests. Use these to validate webhook activity:

```bash
# Get webhook service name (adjust release name and namespace as needed)
RELEASE_NAME="your-release-name"
WEBHOOK_NAMESPACE="your-webhook-namespace"
SERVICE_NAME="${RELEASE_NAME}-cloudzero-agent-webhook-server-svc"

# Method 1: Use kubectl run with curl to access metrics endpoint
kubectl run webhook-metrics-check \
  --image=curlimages/curl --rm -it --restart=Never \
  -- curl -s -k "https://${SERVICE_NAME}.${WEBHOOK_NAMESPACE}.svc:443/metrics"

# Method 2: Port-forward and access locally
kubectl port-forward -n ${WEBHOOK_NAMESPACE} service/${SERVICE_NAME} 8443:443 &
curl -k https://localhost:8443/metrics

# Look for webhook metrics in the output
grep "czo_webhook_types_total" metrics_output.txt
```

### Expected Metrics

During load testing, you should see metrics like:

```prometheus
# HELP czo_webhook_types_total Total number of webhook events filterable by kind_group, kind_version, kind_resource, and operation
# TYPE czo_webhook_types_total counter

# Namespace creations
czo_webhook_types_total{kind_group="",kind_resource="namespace",kind_version="v1",operation="CREATE"} 100

# Pod creations
czo_webhook_types_total{kind_group="",kind_resource="pod",kind_version="v1",operation="CREATE"} 500
```

### Validate Load Test Results

After running the load test, check metrics to confirm all webhook invocations:

```bash
# Run your load test with wave processing for better monitoring
./webhook-load-test -namespaces 50 -pods-per-ns 2 -batch-size 10 -nsgroup metrics-test

# Immediately check metrics while resources exist
kubectl run metrics-validator \
  --image=curlimages/curl --rm -it --restart=Never \
  -- curl -s -k "https://${SERVICE_NAME}.${WEBHOOK_NAMESPACE}.svc:443/metrics" | \
  grep -E "czo_webhook_types_total.*namespace.*CREATE|czo_webhook_types_total.*pod.*CREATE"

# Expected: 50 namespace CREATE operations + 100 pod CREATE operations
# Wave processing makes it easier to correlate metrics with test progress
```

### Webhook Logs

```bash
# Check webhook logs for admission processing details
kubectl logs -n <webhook-namespace> <webhook-pod> -f
```

## Integration with CloudZero Agent Backfill

### Resource Identification for Backfill

The test creates resources with predictable naming and labels that can be identified in backfill operations:

```bash
# Namespaces: {ns-prefix}-1, {ns-prefix}-2, ..., {ns-prefix}-N
# Pods: {pod-prefix}-1, {pod-prefix}-2, ..., {pod-prefix}-M (in each namespace)
# All tagged with: nsgroup=<group-name>, team=cirrus, purpose=testing
```

### Pre-Deployment Testing Benefits

1. **Webhook validation** - Confirms always-allow behavior works correctly
2. **Backfill verification** - Generated resources appear in backfill data before cleanup
3. **Performance baseline** - Establishes webhook performance characteristics
4. **Resource tracking** - Validates that created resources are properly identified and processed

## Expected Behavior

With the always-allow webhook configured correctly:

- ✅ **All resources should be created successfully** regardless of any webhook validation issues
- ✅ **Resources appear in backfill operations** during their lifetime
- ✅ **Automatic cleanup** removes all test resources after the specified duration
- ✅ **No webhook failures** should block resource creation
- ⚠️ **Client-side throttling is normal** and doesn't indicate webhook problems

## Troubleshooting

### Test Failures

- **Resource creation fails**: Check webhook deployment and always-allow configuration
- **Validation fails**: Ensure webhook is processing requests and allowing all operations
- **Cleanup fails**: Verify RBAC permissions for namespace deletion

### Performance Issues

- **High throttling**: Reduce `-concurrency` parameter (try 2-3)
- **Slow execution**: Check cluster load and webhook response times
- **Timeouts**: Increase `-live-duration` for larger tests
