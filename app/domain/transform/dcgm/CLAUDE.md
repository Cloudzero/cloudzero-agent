# DCGM Transformer - AI Development Guide

## Quick Reference

**Purpose**: Transform NVIDIA DCGM exporter metrics into standardized GPU metrics for cost allocation

**Location**: `app/domain/transform/dcgm/`

**Key Files**:

- [transformer.go](transformer.go) - Core transformation logic
- [transformer_test.go](transformer_test.go) - Comprehensive unit tests
- [README.md](README.md) - User-facing documentation

**Testing**: `GO_TEST_TARGET=./app/domain/transform/dcgm make test`

## Architecture Context

### Hexagonal Architecture Position

````text
app/
├── types/                     # Interfaces (ports)
│   └── metric_transformer.go # MetricTransformer interface
│
├── domain/                    # Business logic (core)
│   ├── metric_collector.go   # Invokes transformation pipeline
│   └── transform/
│       ├── catalog/          # Routes to specialized transformers
│       └── dcgm/             # ← THIS PACKAGE
│           └── transformer.go # Implements MetricTransformer
```text

### Integration Points

**Upstream** (calls this package):

- [app/domain/transform/catalog/catalog.go](../catalog/catalog.go:55) - Routes DCGM metrics here
- [app/domain/metric_collector.go](../../metric_collector.go:216) - Invokes catalog transformer

**Downstream** (this package calls):

- [app/types/metric.go](../../../types/metric.go) - Metric data structure
- `github.com/rs/zerolog/log` - Structured logging

**Configuration**:

- [helm/templates/\_defaults.tpl](../../../../helm/templates/_defaults.tpl:69-75) - Prometheus scraping
- [helm/templates/\_defaults.tpl](../../../../helm/templates/_defaults.tpl:234-235) - Cost metric filters

## Implementation Guide

### Transformation Logic Flow

```go
// Phase 1: Per-Metric Processing
Transform(ctx, metrics) {
    for each metric {
        if DCGM_FI_DEV_GPU_UTIL:
            → immediate transform → container_resources_gpu_usage_percent

        if DCGM_FI_DEV_FB_USED:
            → buffer in memoryBuffer[namespace/pod/container/gpu].used

        if DCGM_FI_DEV_FB_FREE:
            → buffer in memoryBuffer[namespace/pod/container/gpu].free

        else:
            → pass through unchanged
    }
}

// Phase 2: Batch Completion
flushMemory(ctx) {
    for each buffered pair {
        if has_both(used, free):
            percentage = (used / (used + free)) * 100
            → create container_resources_gpu_memory_usage_percent
        else:
            → drop incomplete pair (log warning)
    }
    clear buffer
}
```text

### Key Data Structures

**Transformer**:

```go
type Transformer struct {
    memoryBuffer map[string]*memoryPair  // Key: "namespace/pod/container/gpu"
}
```text

**Memory Pair**:

```go
type memoryPair struct {
    used *types.Metric  // DCGM_FI_DEV_FB_USED
    free *types.Metric  // DCGM_FI_DEV_FB_FREE
}
```text

**Buffer Key Format**:

```text
"{namespace}/{pod}/{container}/{gpu}"
Example: "default/gpu-pod-123/cuda-app/0"
```text

### Critical Implementation Details

**1. Why Buffering?**

Memory metrics arrive as separate USED and FREE metrics. We need both to calculate percentage:

```go
percentage = (used / (used + free)) * 100
```text

Buffering ensures we have complete pairs before calculation.

**2. Why Flush at End?**

The buffer accumulates metrics throughout the batch. Flushing at the end ensures:

- All metrics in the batch are considered
- Pairs are only calculated when complete
- Buffer doesn't grow unbounded across batches

**3. Required Labels Check**

Container attribution requires these labels:

```go
var requiredLabels = []string{"namespace", "pod", "container"}
```text

Metrics missing any are dropped - we can't attribute cost without knowing which container used the GPU.

**4. Label Transformations**

The transformer standardizes DCGM labels for consistency:

**UUID → gpu_uuid Renaming**:

```go
// copyLabels renames UUID to gpu_uuid for standardization
if k == "UUID" {
    result["gpu_uuid"] = v  // Standardized name
}
```text

**Node Name Aliasing**:

```go
nodeName := metric.NodeName
if nodeName == "" {
    nodeName = metric.Labels["Hostname"]  // Fallback to DCGM label
}
```text

These transformations ensure compatibility with standardized GPU metric conventions.

## Development Workflow

### Adding New DCGM Metrics

**Step 1**: Add constant for DCGM metric name:

```go
const (
    dcgmGPUTemperature = "DCGM_FI_DEV_GPU_TEMP"  // Example new metric
)
```text

**Step 2**: Add constant for standardized name:

```go
const (
    standardGPUTemperature = "container_resources_gpu_temperature_celsius"
)
```text

**Step 3**: Add case to `transformSingle()`:

```go
case dcgmGPUTemperature:
    return transformGPUTemperature(metric), nil
```text

**Step 4**: Implement transformation function:

```go
func transformGPUTemperature(metric types.Metric) []types.Metric {
    transformed := metric
    transformed.MetricName = standardGPUTemperature
    transformed.ID = uuid.New()
    return []types.Metric{transformed}
}
```text

**Step 5**: Add test cases to [transformer_test.go](transformer_test.go).

**Step 6**: Update Helm chart to include metric in filters:

```yaml
# helm/templates/_defaults.tpl
containerMetrics:
  - container_resources_gpu_temperature_celsius
```text

### Adding Tests

Follow the table-driven test pattern:

```go
func TestTransform_NewMetric(t *testing.T) {
    tests := []struct {
        name     string
        input    []types.Metric
        want     []types.Metric
        wantErr  bool
    }{
        {
            name: "transforms new metric",
            input: []types.Metric{{
                MetricName: "DCGM_FI_DEV_NEW_METRIC",
                // ... test data
            }},
            want: []types.Metric{{
                MetricName: "container_resources_new_metric",
                // ... expected output
            }},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            transformer := NewTransformer()
            got, err := transformer.Transform(context.Background(), tt.input)

            if tt.wantErr {
                assert.Error(t, err)
                return
            }

            assert.NoError(t, err)
            assert.Equal(t, tt.want, got)
        })
    }
}
```text

### Debugging Tips

**1. Enable Debug Logging**

Set aggregator log level to debug in cluster overrides:

```yaml
# clusters/brahms-overrides.yaml
aggregator:
  logging:
    level: debug
```text

**2. Check Metric Flow**

```bash
# Check if DCGM metrics are being received
kubectl -n cloudzero-agent logs -l app.kubernetes.io/component=aggregator \
  | grep "DCGM_FI_DEV"

# Check if transformed metrics are produced
kubectl -n cloudzero-agent logs -l app.kubernetes.io/component=aggregator \
  | grep "container_resources_gpu"

# Check for dropped metrics
kubectl -n cloudzero-agent logs -l app.kubernetes.io/component=aggregator \
  | grep "dropping"
```text

**3. Add Temporary Logging**

If debugging specific issues, add temporary logs (remember to remove):

```go
log.Ctx(ctx).Debug().
    Str("metric", metric.MetricName).
    Interface("labels", metric.Labels).
    Msg("DEBUG: processing metric")
```text

**Mark with `// TODO: TEMPORARY` and remove after debugging.**

## Common Development Tasks

### Task: Add New GPU Vendor Support

**Example**: Adding AMD ROCm support

**Step 1**: Create new package `app/domain/transform/rocm/`

**Step 2**: Implement `types.MetricTransformer`:

```go
package rocm

type Transformer struct {
    // ROCm-specific state
}

func (t *Transformer) Transform(ctx context.Context, metrics []types.Metric) ([]types.Metric, error) {
    // Transform ROCm metrics to standard format
}
```text

**Step 3**: Register in catalog transformer:

```go
// app/domain/transform/catalog/catalog.go
func NewMetricTransformer() types.MetricTransformer {
    return &Transformer{
        transformers: []types.MetricTransformer{
            dcgm.NewTransformer(),
            rocm.NewTransformer(),  // Add AMD support
        },
    }
}
```text

**Step 4**: Add Prometheus scrape job in Helm chart for ROCm exporter.

### Task: Change Transformation Logic

**Example**: Use total memory instead of percentage

**Current**:

```go
percentage = (used / (used + free)) * 100
```text

**New**:

```go
totalBytes = used  // Just report used bytes directly
```text

**Changes Required**:

1. Update `calculateMemoryPercentage()` function
2. Update metric name to reflect bytes vs percentage
3. Update unit tests for new calculation
4. Update Helm chart metric name in filters
5. Update this documentation

### Task: Fix Memory Leak in Buffer

**Symptom**: Memory buffer grows unbounded

**Diagnosis**:

```go
// Check if buffer is being cleared
log.Ctx(ctx).Debug().
    Int("buffer_size_before_flush", len(t.memoryBuffer)).
    Int("buffer_size_after_flush", len(t.memoryBuffer)).
    Msg("buffer flush")
```text

**Fix**: Ensure `flushMemory()` clears buffer at end:

```go
// Clear buffer after flush
t.memoryBuffer = make(map[string]*memoryPair)
```text

## Testing Checklist

Before submitting changes:

- [ ] Unit tests pass: `GO_TEST_TARGET=./app/domain/transform/dcgm make test`
- [ ] Test coverage >90%: `go test -cover ./app/domain/transform/dcgm`
- [ ] Integration test with DCGM exporter in test cluster
- [ ] Helm chart updated if metric names changed
- [ ] Documentation updated (README.md, CLAUDE.md)
- [ ] No temporary debug logging remains
- [ ] Code follows existing patterns (see similar functions)

## Performance Guidelines

### Memory Buffer Size

**Expected**: 10-100 entries per batch
**Maximum**: ~1000 entries (acceptable)
**Alert**: >10,000 entries (indicates buffer not being cleared)

**Monitoring**:

```go
if len(t.memoryBuffer) > 1000 {
    log.Ctx(ctx).Warn().
        Int("buffer_size", len(t.memoryBuffer)).
        Msg("DCGM memory buffer unusually large")
}
```text

### Transformation Latency

**Expected**: <1ms per batch
**Maximum**: <10ms per batch
**Alert**: >100ms per batch

The transformation should add negligible overhead compared to network and database I/O.

## Related Packages

**Must Read First**:

- [app/domain/transform/README.md](../README.md) - Transformation architecture overview
- [app/domain/transform/catalog/CLAUDE.md](../catalog/CLAUDE.md) - Routing logic

**Related Transformers** (future):

- `app/domain/transform/rocm/` - AMD GPU support
- `app/domain/transform/xpu/` - Intel GPU support

**Consumers**:

- [app/domain/metric_collector.go](../../metric_collector.go) - Invokes transformation
- [app/domain/metric_filter.go](../../metric_filter.go) - Filters transformed metrics

## Important Constraints

### DO NOT Change

These are part of the public contract and changing them breaks cost allocation:

1. **Standardized metric names**:

   - `container_resources_gpu_usage_percent`
   - `container_resources_gpu_memory_usage_percent`

2. **Required labels**: `namespace`, `pod`, `container` (needed for cost attribution)

3. **Percentage range**: 0-100 (CloudZero expects this range)

### Safe to Change

These are internal implementation details:

1. Buffer key format (as long as it's unique per GPU)
2. Logging messages
3. Internal function names
4. Performance optimizations
5. Error messages

### Requires Coordination

These require updates in multiple places:

1. **Adding new metrics**: Update Helm chart filters
2. **Changing label requirements**: Update DCGM Exporter config
3. **Changing calculation logic**: Update documentation and tests

## Edge Cases to Consider

### Multi-Instance GPU (MIG)

NVIDIA MIG partitions a single GPU into multiple instances. Each instance appears as a separate GPU:

```text
gpu="0"  → MIG instance 0 of physical GPU 0
gpu="1"  → MIG instance 1 of physical GPU 0
```text

The transformer handles this correctly - each MIG instance is treated as a separate GPU in the buffer key.

### GPU Time-Slicing

When Kubernetes time-slices GPUs across containers, DCGM reports per-container metrics:

```text
namespace="default", pod="pod-a", container="app-1", gpu="0"
namespace="default", pod="pod-b", container="app-2", gpu="0"
```text

The transformer handles this correctly - each container gets its own buffer key.

### Missing Metrics

If DCGM Exporter stops reporting metrics (crash, network issue), incomplete pairs accumulate in buffer:

**Current behavior**: Dropped with warning on next flush
**Alternative**: Could implement TTL to drop stale pairs

## Maintenance Notes

### When to Update This Package

1. **NVIDIA releases new DCGM metrics**: Add to transformation logic
2. **CloudZero adds GPU cost models**: May need new derived metrics
3. **Performance issues**: Optimize buffer management
4. **New GPU vendors**: Create sibling packages (don't modify this one)

### When NOT to Update This Package

1. **Changing metric filters**: Update Helm chart instead
2. **Changing Prometheus scraping**: Update Prometheus config
3. **DCGM Exporter deployment**: Update DCGM Exporter Helm chart
4. **Cost calculation changes**: Update CloudZero backend

### Code Review Checklist

When reviewing changes to this package:

- [ ] Unit tests cover new/changed code paths
- [ ] Existing tests still pass
- [ ] No breaking changes to metric names or output format
- [ ] Buffer management remains bounded (cleared after flush)
- [ ] Required labels are validated
- [ ] Error cases have appropriate logging
- [ ] Performance impact is acceptable (<10ms per batch)
- [ ] Documentation updated to reflect changes

## Quick Command Reference

```bash
# Run tests
GO_TEST_TARGET=./app/domain/transform/dcgm make test

# Run tests with coverage
go test -cover ./app/domain/transform/dcgm

# Run specific test
GO_TEST_FLAGS="-run TestTransform_GPUUtilization" \
  GO_TEST_TARGET=./app/domain/transform/dcgm make test

# Check for race conditions
GO_TEST_FLAGS="-race" GO_TEST_TARGET=./app/domain/transform/dcgm make test

# Format code
make format

# Lint code
make lint

# Build entire project (includes this package)
make build

# Deploy to test cluster
CLUSTER_NAME=brahms make helm-install helm-wait

# Check logs for GPU metrics
kubectl -n cloudzero-agent logs -l app.kubernetes.io/component=aggregator \
  --tail=100 | grep -E "(DCGM|container_resources_gpu)"
```text

## Remember

1. **Read [README.md](README.md) first** - It has the user-facing documentation
2. **Check [app/domain/transform/README.md](../README.md)** - Understand transformation architecture
3. **Follow hexagonal architecture** - Keep business logic in domain layer
4. **Use table-driven tests** - See existing tests for patterns
5. **Document why, not what** - Code shows what, comments explain why
6. **Test with real DCGM metrics** - Unit tests alone aren't sufficient
7. **Update Helm chart when adding metrics** - Filter configuration matters

## Support

If you're stuck:

1. Read the README.md in this directory
2. Check similar transformers (catalog transformer as example)
3. Look at existing test cases for patterns
4. Review the MetricTransformer interface definition
5. Check DCGM Exporter documentation for metric definitions
````
