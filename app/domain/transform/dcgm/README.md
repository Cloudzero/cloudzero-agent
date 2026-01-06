# DCGM Transformer

## Overview

The DCGM transformer converts NVIDIA DCGM (Data Center GPU Manager) exporter metrics into standardized container-level GPU resource metrics for cost allocation analysis. This transformer is part of the metric transformation pipeline and handles GPU-specific metric normalization.

## Purpose

NVIDIA DCGM Exporter provides raw GPU telemetry in NVIDIA's native format. For accurate cost allocation, CloudZero needs:

1. **Standardized metric names** - Consistent naming across GPU vendors (NVIDIA, AMD, Intel)
2. **Percentage-based metrics** - Normalized values for comparison and analysis
3. **Container attribution** - Per-container GPU usage for accurate cost allocation

This transformer bridges the gap between NVIDIA's native DCGM format and CloudZero's standardized GPU metrics.

## Architecture

````text
MetricCollector
  └── catalog.Transformer (metric routing)
        └── dcgm.Transformer (NVIDIA GPU metrics)
```text

The DCGM transformer is invoked by the catalog transformer, which routes metrics to specialized transformers based on metric characteristics. Future GPU vendors (AMD ROCm, Intel XPU) would be implemented as sibling transformers.

## Metric Transformations

### Input Metrics (DCGM Format)

The transformer processes three DCGM metrics from NVIDIA DCGM Exporter:

| DCGM Metric            | Description                 | Unit               | Labels                                                                                 |
| ---------------------- | --------------------------- | ------------------ | -------------------------------------------------------------------------------------- |
| `DCGM_FI_DEV_GPU_UTIL` | GPU compute utilization     | Percentage (0-100) | namespace, pod, container, gpu, Hostname, UUID, modelName, pci_bus_id, device, pod_uid |
| `DCGM_FI_DEV_FB_USED`  | GPU framebuffer memory used | MiB                | namespace, pod, container, gpu, Hostname, UUID, modelName, pci_bus_id, device, pod_uid |
| `DCGM_FI_DEV_FB_FREE`  | GPU framebuffer memory free | MiB                | namespace, pod, container, gpu, Hostname, UUID, modelName, pci_bus_id, device, pod_uid |

### Output Metrics (Standardized Format)

The transformer produces two standardized container-level GPU metrics:

| Standardized Metric                            | Description             | Unit               | Calculation                              |
| ---------------------------------------------- | ----------------------- | ------------------ | ---------------------------------------- |
| `container_resources_gpu_usage_percent`        | GPU compute utilization | Percentage (0-100) | Pass-through from `DCGM_FI_DEV_GPU_UTIL` |
| `container_resources_gpu_memory_usage_percent` | GPU memory utilization  | Percentage (0-100) | `(USED / (USED + FREE)) * 100`           |

### Transformation Rules

1. **GPU Utilization**: Direct rename from `DCGM_FI_DEV_GPU_UTIL` to `container_resources_gpu_usage_percent`

   - Already in percentage format (0-100)
   - No calculation required
   - Immediate transformation

2. **GPU Memory Utilization**: Calculated from `DCGM_FI_DEV_FB_USED` and `DCGM_FI_DEV_FB_FREE`
   - Formula: `(used / (used + free)) * 100`
   - Requires buffering to ensure paired metrics
   - Calculated during flush phase

## Processing Strategy

The transformer uses a **buffered processing model** for memory metrics to handle the asynchronous arrival of USED and FREE metrics:

### Phase 1: Transform (Per Metric)

```text
For each metric in batch:
  ├─ If DCGM_FI_DEV_GPU_UTIL
  │   └─ Transform immediately → container_resources_gpu_usage_percent
  │
  ├─ If DCGM_FI_DEV_FB_USED
  │   └─ Buffer in memoryBuffer (key: namespace/pod/container/gpu)
  │
  ├─ If DCGM_FI_DEV_FB_FREE
  │   └─ Buffer in memoryBuffer (key: namespace/pod/container/gpu)
  │
  └─ If not DCGM metric
      └─ Pass through unchanged
```text

### Phase 2: Flush (End of Batch)

```text
For each buffered memory pair:
  ├─ If both USED and FREE present
  │   └─ Calculate percentage → container_resources_gpu_memory_usage_percent
  │
  └─ If incomplete pair (missing USED or FREE)
      └─ Drop (logged as incomplete pair)
```text

This two-phase approach ensures:

- Memory metrics are always calculated from complete USED+FREE pairs
- GPU utilization metrics are transformed immediately (no buffering overhead)
- Non-DCGM metrics pass through without interference

## Required Labels

For accurate container attribution, DCGM metrics must include these labels:

- `namespace` - Kubernetes namespace
- `pod` - Pod name
- `container` - Container name

Metrics missing any required label are dropped with a warning log.

## Label Handling

### DCGM Labels Preserved and Transformed

The transformer preserves all DCGM-specific labels for operational correlation:

- `gpu` - GPU index (0, 1, 2, etc.) - **preserved as-is**
- `Hostname` - Node hostname where GPU is located (e.g., "ip-10-30-23-129.ec2.internal") - **preserved as-is**
- `UUID` → `gpu_uuid` - NVIDIA GPU UUID (e.g., "GPU-4980eea4-963e-7b82-ecb9-36ee26fdceb8") - **renamed for standardization**
- `modelName` - GPU model name (e.g., "Tesla T4", "NVIDIA A100-SXM4-40GB") - **preserved as-is**
- `pci_bus_id` - PCIe bus identifier (e.g., "00000000:00:1E.0") - **preserved as-is**
- `device` - NVIDIA device name (e.g., "nvidia0") - **preserved as-is**
- `pod_uid` - Kubernetes pod UID (may be empty) - **preserved as-is**

**Label Renaming**: The `UUID` label from DCGM is renamed to `gpu_uuid` in the output metrics for consistency with standardized GPU metric conventions.

### Label Aliasing

For node attribution, the transformer provides fallback logic:

- Primary: Use `node` label if present
- Fallback: Use `Hostname` label from DCGM

This ensures node attribution works regardless of label source.

## Configuration

### Helm Chart Integration

The DCGM transformer is automatically enabled when GPU metrics are configured:

```yaml
# clusters/brahms-overrides.yaml
prometheusConfig:
  scrapeJobs:
    gpu:
      enabled: true
      scrapeInterval: 30s
```text

### Metric Filtering

Transformed metrics are classified as **cost metrics** in the Helm chart:

```yaml
# helm/templates/_defaults.tpl
metricFilters:
  cost:
    name:
      exact:
        - container_resources_gpu_usage_percent
        - container_resources_gpu_memory_usage_percent
```text

Note: Raw DCGM metrics are **not** included in cost filters - only the transformed percentage-based metrics.

## Data Flow

```text
DCGM Exporter (GPU node)
  │
  │ DCGM_FI_DEV_GPU_UTIL
  │ DCGM_FI_DEV_FB_USED
  │ DCGM_FI_DEV_FB_FREE
  │
  ▼
Prometheus (scrapes every 30s)
  │
  │ Prometheus Remote Write
  │
  ▼
CloudZero Agent Aggregator
  │
  ▼
MetricCollector.PutMetrics()
  │
  ├─ Decode Prometheus format
  │
  ├─ Transform (catalog → dcgm)
  │   │
  │   ├─ GPU_UTIL → container_resources_gpu_usage_percent
  │   │
  │   └─ FB_USED + FB_FREE → container_resources_gpu_memory_usage_percent
  │
  ├─ Filter (cost vs observability)
  │   │
  │   └─ Transformed metrics → COST
  │
  └─ Store in database → Ship to CloudZero
```text

## Error Handling

### Incomplete Memory Pairs

If a memory USED metric arrives without a corresponding FREE metric (or vice versa), the incomplete pair is dropped:

```go
log.Ctx(ctx).Warn().
    Str("key", key).
    Bool("has_used", pair.used != nil).
    Bool("has_free", pair.free != nil).
    Msg("dropping incomplete DCGM memory metric pair")
```text

This can occur when:

- Prometheus scrape fails for one metric but not the other
- DCGM Exporter temporarily stops reporting one metric
- Network issues cause partial metric delivery

### Missing Required Labels

Metrics without required container attribution labels are dropped:

```go
log.Ctx(ctx).Warn().
    Str("metric", metric.MetricName).
    Interface("labels", metric.Labels).
    Msg("dropping DCGM metric missing required labels")
```text

### Memory Buffer Management

The memory buffer is cleared after each flush to prevent unbounded growth:

```go
// Clear buffer after flush
t.memoryBuffer = make(map[string]*memoryPair)
```text

This ensures the buffer doesn't accumulate stale metrics across batches.

## Testing

### Unit Tests

Comprehensive unit tests cover all transformation scenarios:

```bash
# Run DCGM transformer tests
GO_TEST_TARGET=./app/domain/transform/dcgm make test

# Run with verbose output
GO_TEST_FLAGS="-v" GO_TEST_TARGET=./app/domain/transform/dcgm make test
```text

Test coverage includes:

- GPU utilization pass-through transformation
- Memory percentage calculation from USED+FREE pairs
- Label preservation and aliasing
- Error cases (missing labels, incomplete pairs)
- Buffer management and flush behavior

### Integration Testing

End-to-end testing with actual DCGM metrics:

1. Deploy DCGM Exporter to a GPU-enabled cluster
2. Configure CloudZero Agent with GPU scraping enabled
3. Verify transformed metrics appear in CloudZero platform
4. Validate cost allocation accuracy for GPU workloads

## Performance Considerations

### Memory Buffer Size

The memory buffer is bounded by the number of unique GPU containers in a single batch:

- **Typical size**: 10-100 entries (10 pods × 1-10 GPUs each)
- **Maximum size**: ~1000 entries (pathological case)
- **Memory overhead**: ~1 KB per entry (metric metadata + pointers)

The buffer is cleared after each batch, preventing unbounded growth.

### Transformation Overhead

- **GPU utilization**: O(1) - simple field rename
- **Memory percentage**: O(1) - buffered calculation during flush
- **Overall complexity**: O(n) where n = number of metrics in batch

Transformation adds negligible latency (<1ms per batch) compared to network and database I/O.

## Future Extensions

### Multi-Vendor Support

The transformer architecture supports future GPU vendor extensions:

```text
app/domain/transform/
  ├── catalog/        # Metric routing
  ├── dcgm/          # NVIDIA GPUs (current)
  ├── rocm/          # AMD GPUs (future)
  └── xpu/           # Intel GPUs (future)
```text

Each vendor-specific transformer would:

1. Convert vendor metrics to standardized format
2. Handle vendor-specific label schemas
3. Implement vendor-specific calculation logic

### Additional GPU Metrics

Future enhancements may include:

- **GPU temperature** - For thermal-aware cost optimization
- **GPU power consumption** - For energy cost attribution
- **GPU error rates** - For reliability tracking
- **Multi-instance GPU (MIG)** - For GPU partitioning support

## Troubleshooting

### No GPU Metrics Appearing

**Symptom**: No `container_resources_gpu_*` metrics in CloudZero

**Diagnosis**:

1. Check DCGM Exporter is running:

   ```bash
   kubectl get pods -A | grep dcgm
````

2. Check Prometheus is scraping DCGM:

   ```bash
   # Port-forward to Prometheus
   kubectl -n cloudzero-agent port-forward svc/prometheus 9090:9090

   # Check targets: http://localhost:9090/targets
   # Look for "cloudzero-dcgm-exporter" job
   ```

3. Check aggregator logs for DCGM metrics:
   ```bash
   kubectl -n cloudzero-agent logs -l app.kubernetes.io/part-of=cloudzero-agent,app.kubernetes.io/name=aggregator \
     | grep "DCGM_FI_DEV"
   ```

### Metrics Being Dropped

**Symptom**: DCGM metrics received but not transformed

**Diagnosis**: Check for missing required labels:

````bash
kubectl -n cloudzero-agent logs -l app.kubernetes.io/part-of=cloudzero-agent,app.kubernetes.io/name=aggregator \
  | grep "dropping DCGM metric missing required labels"
```text

**Resolution**: Ensure DCGM Exporter is configured to include Kubernetes labels (namespace, pod, container).

### Missing Memory Metrics

**Symptom**: GPU utilization metrics present but memory metrics missing

**Diagnosis**: Check for incomplete pairs:

```bash
kubectl -n cloudzero-agent logs -l app.kubernetes.io/part-of=cloudzero-agent,app.kubernetes.io/name=aggregator \
  | grep "incomplete DCGM memory metric pair"
```text

**Resolution**:

- Check DCGM Exporter health
- Verify Prometheus scrape success rate
- Check for network issues between Prometheus and DCGM Exporter

## References

### NVIDIA DCGM

- [DCGM Documentation](https://docs.nvidia.com/datacenter/dcgm/)
- [DCGM Exporter GitHub](https://github.com/NVIDIA/dcgm-exporter)
- [DCGM Field Identifiers](https://docs.nvidia.com/datacenter/dcgm/latest/dcgm-api/dcgm-api-field-ids.html)

### Related Documentation

- [app/domain/transform/README.md](../README.md) - Transformation architecture
- [app/domain/transform/catalog/README.md](../catalog/README.md) - Metric routing
- [helm/docs/troubleshooting-guide.md](../../../../helm/docs/troubleshooting-guide.md) - Operational troubleshooting
````
