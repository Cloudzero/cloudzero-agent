# Autoscaling the CloudZero Agent

## Introduction

The CloudZero Agent's autoscaling feature automatically adjusts the number of aggregator pods based on the volume of cost metrics being processed. This ensures optimal resource utilization while maintaining performance on clusters of any size.

### What is Autoscaling?

Autoscaling automatically increases or decreases the number of replicas based on observed metrics. Traditional autoscaling relies on CPU or memory usage, but the CloudZero Agent uses a more sophisticated approach based on business logic—specifically, how close the system is to reaching its data shipping capacity.

### Why Custom Metrics?

The CloudZero Agent processes cost metrics from Kubernetes clusters and ships them to the CloudZero platform. The volume of this data varies significantly based on:

- **Cluster size**: Larger clusters generate more metrics
- **Resource churn**: Frequent pod creation/deletion increases metric volume
- **Time of day**: Peak business hours often correlate with higher metric volume
- **Deployment patterns**: CI/CD activity and autoscaling events generate metric spikes

### How It Works

The aggregator component maintains an in-memory buffer of cost metrics before flushing them to disk and shipping to CloudZero. The custom metric `czo_cost_metrics_shipping_progress` tracks how full this buffer is relative to the configured thresholds.

When the buffer fills up (indicating high metric volume), the HPA automatically scales up additional aggregator pods to handle the increased load. When the buffer is consistently low (indicating normal or low volume), it scales down to conserve resources.

This approach provides:

- **Proactive scaling**: Scale before performance degrades
- **Cost efficiency**: Use only the resources needed for current load
- **High availability**: Prevent data loss during metric volume spikes
- **Predictable performance**: Maintain consistent shipping rates regardless of load

## Implementation Details

### Custom Metrics API

The chart automatically registers the custom metrics API with Kubernetes by creating an `APIService` resource. This allows the HPA controller to discover and query the custom metrics endpoint exposed by the aggregator pods.

### The `czo_cost_metrics_shipping_progress` Metric

The autoscaling system is built around a single custom metric that represents how close the system is to its in-memory buffer capacity. This metric is calculated based on two flush triggers that occur during normal operation:

1. **Record Count**: Flush when `currentPending >= maxRecords`
2. **Time Interval**: Flush when `elapsedTime >= costMaxInterval`

#### Calculation Formula

The metric is calculated as:

$$\frac{\text{Records Pending}}{\frac{\text{Elapsed Time}}{\text{Max Interval}} \times \text{Max Records}}$$

This formula normalizes the current buffer state against the expected capacity over time, providing a percentage-based value where:

- **0.0**: Buffer is empty
- **1.0**: Buffer is at expected capacity for the elapsed time
- **> 1.0**: Buffer is over capacity and needs scaling

#### Example Values

Based on the default `maxRecords = 1,500,000` and `costMaxInterval = 30m`:

- **5 minutes elapsed, 250,000 records**: `progress = 250,000 / (5/30 * 1,500,000) = 1.0` (100% of expected rate)
- **10 minutes elapsed, 300,000 records**: `progress = 300,000 / (10/30 * 1,500,000) = 0.6` (60% of expected rate - could scale down)
- **15 minutes elapsed, 900,000 records**: `progress = 900,000 / (15/30 * 1,500,000) = 1.2` (120% of expected rate - should scale up)
- **30 minutes elapsed, 1,500,000 records**: `progress = 1,500,000 / (30/30 * 1,500,000) = 1.0` (100% at time limit)

#### Relationship to System Behavior

The metric directly correlates with the system's disk flushing behavior:

1. **Normal Operation**: Metrics accumulate in memory buffer
2. **Flush Trigger**: When `currentPending >= maxRecords` or time interval exceeded, system flushes to disk
3. **Post-Flush**: `currentPending` resets to 0, metric drops to 0.0
4. **Cycle Repeats**: Buffer fills again as new metrics arrive

#### HPA Integration

By default, the HPA targets `"900m"` (0.9 or 90% of capacity). The target value of 90% allows the HPA to proactively scale before the buffer becomes full, preventing data loss or performance degradation:

- **Below 0.9**: May scale down to as few as `minReplicas`
- **Above 0.9**: May scale up to as many as `maxReplicas`

## Configuration

### Enabling Autoscaling

To enable HPA for the aggregator component:

```yaml
components:
  aggregator:
    autoscale: true
```

This automatically creates:

- **HPA Resource**: Configured to scale based on `czo_cost_metrics_shipping_progress`
- **APIService Registration**: Registers `v1beta1.custom.metrics.k8s.io` with Kubernetes
- **RBAC Permissions**: Allows the HPA controller to access custom metrics

### Scaling Parameters

Detailed configuration is available in the (non-API-stable) `aggregator.scaling` section:

```yaml
aggregator:
  scaling:
    minReplicas: 1
    maxReplicas: 10
    targetValue: "900m" # Target 90% of capacity
    behavior:
      scaleUp:
        stabilizationWindowSeconds: 300
        policies:
          - type: Percent
            value: 100
            periodSeconds: 60
          - type: Pods
            value: 2
            periodSeconds: 60
        selectPolicy: Max
      scaleDown:
        stabilizationWindowSeconds: 300
        policies:
          - type: Percent
            value: 50
            periodSeconds: 60
          - type: Pods
            value: 1
            periodSeconds: 60
        selectPolicy: Min
```

### Target Value Format

The `targetValue` supports both percentage and resource quantity formats:

- **Percentage**: `"90%"` (90% of maximum records threshold)
- **Resource Quantity**: `"900m"` (0.9 as a decimal)

Both formats are equivalent and represent the same scaling threshold.

## Troubleshooting

### HPA Shows "Unknown" Metrics

**Symptoms**: HPA status shows metrics as "unknown" or "0"

**Causes**:

- Collector pods not running
- Custom metrics API not accessible
- Network policies blocking access API server on collector container

**Solutions**:

1. Verify collector pods are running:

   ```bash
   kubectl get pods -l app.kubernetes.io/name=cloudzero-collector
   ```

2. Test custom metrics API directly:

   ```bash
   kubectl get --raw "/apis/custom.metrics.k8s.io/v1beta1/"
   ```

3. Check collector logs for API errors:
   ```bash
   kubectl logs -l app.kubernetes.io/name=cloudzero-collector
   ```

### HPA Not Scaling

**Symptoms**: Metric values are high but HPA doesn't scale

**Causes**:

- Target value too high
- MaxReplicas already reached
- Resource constraints

**Solutions**:

1. Check HPA configuration:

   ```bash
   kubectl describe hpa cloudzero-aggregator
   ```

2. Verify resource quotas:

   ```bash
   kubectl describe resourcequota
   ```

3. Review HPA events:
   ```bash
   kubectl get events --field-selector involvedObject.name=cloudzero-aggregator
   ```

### Metric Values Seem Wrong

**Symptoms**: Metric values don't match expected shipping progress

**Causes**:

- Configuration mismatch between aggregator and collector
- Metric calculation errors
- Time synchronization issues

**Solutions**:

1. Compare aggregator configuration:

   ```bash
   kubectl get configmap cloudzero-aggregator-config -o yaml
   ```

2. Check Prometheus metrics directly:
   ```bash
   kubectl port-forward svc/cloudzero-collector 8080:8080
   curl localhost:8080/metrics | grep czo_cost_metrics_shipping_progress
   ```

## Security

### RBAC Requirements

The HPA requires specific permissions to access custom metrics:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: hpa-custom-metrics-reader
rules:
  - apiGroups: ["custom.metrics.k8s.io"]
    resources: ["*"]
    verbs: ["get", "list"]
```

### Network Policies

Ensure network policies allow HPA to communicate with collector pods on the custom metrics API port.
