# Load Tests

This directory contains performance and load testing configurations for the CloudZero Agent. These tests validate the agent's behavior under various load conditions and help identify performance bottlenecks.

## Overview

Load tests ensure the CloudZero Agent can handle production workloads by:

- Testing performance under realistic traffic volumes
- Identifying resource consumption patterns
- Validating scalability characteristics
- Testing error handling under stress conditions

## Test Structure

### Webhook Load Tests (`webhook/`)

Contains configurations and manifests for testing webhook performance:

**What it tests:**

- Webhook admission processing throughput
- Response times under load
- Memory and CPU usage patterns
- Error rates during high-volume operations

### Manifest Generation (`manifests/`)

Contains Kubernetes manifests used for load generation:

**Purpose:**

- Generate realistic Kubernetes resource creation patterns
- Provide test data for webhook processing
- Simulate various resource types and configurations
- Test different admission scenarios

## Load Testing Scenarios

### Webhook Performance Testing

Tests the admission webhook's ability to handle high-volume resource operations:

1. **Burst testing** - High number of simultaneous resource creations
2. **Sustained load** - Continuous resource operations over time
3. **Mixed workloads** - Different resource types and operations
4. **Error scenarios** - Invalid resources and edge cases

### Resource Processing Testing

Tests the agent's resource collection and processing performance:

1. **Large cluster simulation** - Many resources across namespaces
2. **Rapid change scenarios** - Frequent resource updates and deletions
3. **Resource diversity** - Various Kubernetes resource types
4. **Metadata complexity** - Resources with extensive labels and annotations

## Prerequisites

### Testing Infrastructure

- **Kubernetes cluster** - Sufficient resources for load generation
- **Monitoring tools** - Prometheus, Grafana for performance measurement
- **Load generation tools** - Tools for creating Kubernetes resource load
- **CloudZero Agent** - Deployed and configured for testing

### Resource Requirements

```yaml
# Recommended cluster resources for load testing
nodes: 3+
cpu: 8+ cores total
memory: 16GB+ total
storage: 50GB+ available
```

### Monitoring Setup

```bash
# Ensure monitoring is available
kubectl get pods -n monitoring  # or appropriate namespace

# Verify Prometheus is scraping agent metrics
# Check Grafana dashboards are configured
```

## How to Run Load Tests

**IMPORTANT**: Load tests are NOT automatically run by standard make targets. They require manual execution and cluster setup.

### Preparation

```bash
# Ensure agent is deployed
kubectl get pods -n cz-agent

# Verify monitoring is working
kubectl port-forward -n monitoring svc/prometheus 9090:9090

# Check baseline metrics
curl http://localhost:9090/metrics | grep czo_
```

### Webhook Load Testing

```bash
# Navigate to test directory
cd tests/load

# Apply test manifests (creates load)
kubectl apply -f manifests/

# Monitor webhook performance
kubectl top pods -n cz-agent
kubectl logs -n cz-agent -l app=cloudzero-agent-webhook-server -f

# Check webhook metrics
kubectl port-forward -n cz-agent svc/webhook-service 8080:443
curl -k https://localhost:8080/metrics
```

### Custom Load Generation

```bash
# Generate sustained load
for i in {1..100}; do
  kubectl apply -f manifests/test-deployment-$i.yaml
  sleep 1
done

# Monitor resource usage
watch "kubectl top pods -n cz-agent"

# Clean up after testing
kubectl delete -f manifests/
```

## Performance Metrics

### Key Metrics to Monitor

#### Webhook Performance

- **Request rate** - `czo_webhook_requests_total`
- **Response time** - `czo_webhook_duration_seconds`
- **Error rate** - `czo_webhook_errors_total`
- **Queue depth** - `czo_webhook_queue_depth`

#### Resource Usage

- **CPU utilization** - Pod and container CPU metrics
- **Memory consumption** - Memory usage and garbage collection
- **Network traffic** - Request/response volumes
- **Storage I/O** - Disk read/write patterns

#### Kubernetes Metrics

- **API server load** - Request latency and volume
- **ETCD performance** - Storage backend performance
- **Node resource usage** - Overall cluster resource consumption

### Performance Baselines

Establish baseline performance characteristics:

```bash
# Normal operation metrics
webhook_requests_per_second: 10-50
average_response_time_ms: 5-20
memory_usage_mb: 50-200
cpu_usage_cores: 0.1-0.5

# Under load metrics (acceptable)
webhook_requests_per_second: 100-500
average_response_time_ms: 20-100
memory_usage_mb: 200-1000
cpu_usage_cores: 0.5-2.0
```

## Load Test Configurations

### Webhook Load Profiles

#### Light Load

```yaml
resources_per_minute: 60
concurrent_operations: 10
test_duration: 10m
resource_types: [pods, services, deployments]
```

#### Medium Load

```yaml
resources_per_minute: 300
concurrent_operations: 50
test_duration: 30m
resource_types: [all_supported_types]
```

#### Heavy Load

```yaml
resources_per_minute: 1000
concurrent_operations: 200
test_duration: 60m
resource_types: [all_supported_types]
error_injection: 10%
```

### Resource Patterns

Different resource patterns to test various scenarios:

- **Simple resources** - Basic pods and services
- **Complex resources** - Deployments with multiple containers
- **Resource chains** - Resources with dependencies
- **Large resources** - Resources with extensive metadata

## Troubleshooting Load Tests

### Common Performance Issues

1. **High response times**:
   - Check CPU and memory usage
   - Look for garbage collection pressure
   - Verify network connectivity is stable
   - Review webhook logic for bottlenecks

2. **Memory leaks**:
   - Monitor memory usage over time
   - Check for goroutine leaks
   - Review object lifecycle management
   - Analyze garbage collection patterns

3. **Error rate increases**:
   - Check API server rate limiting
   - Verify webhook certificates are valid
   - Review network policies and connectivity
   - Check for resource conflicts

4. **Cluster instability**:
   - Monitor node resource usage
   - Check for API server overload
   - Verify ETCD performance
   - Review overall cluster health

### Debug Commands

```bash
# Check webhook pod resource usage
kubectl top pod -n cz-agent --containers

# Get detailed pod metrics
kubectl describe pod -n cz-agent <webhook-pod-name>

# Check webhook logs for errors
kubectl logs -n cz-agent -l app=webhook --tail=100

# Monitor API server performance
kubectl top nodes
kubectl get events --sort-by=.metadata.creationTimestamp

# Check for resource conflicts
kubectl get events --field-selector type=Warning
```

## Performance Optimization

### Tuning Parameters

Based on load test results, adjust these parameters:

```yaml
# Webhook configuration
replicas: 3 # Scale webhook pods
resources:
  requests:
    cpu: 200m
    memory: 256Mi
  limits:
    cpu: 1000m
    memory: 1Gi

# Performance settings
max_concurrent_requests: 100
request_timeout: 30s
keep_alive_connections: true
```

### Scaling Strategies

- **Horizontal scaling** - Increase webhook pod replicas
- **Vertical scaling** - Increase CPU and memory limits
- **Load balancing** - Distribute load across multiple pods
- **Caching** - Cache frequently accessed data
- **Batching** - Process multiple operations together

## Best Practices

### Load Test Design

- Start with baseline measurements
- Gradually increase load to find limits
- Test realistic scenarios and data patterns
- Include error and edge case scenarios
- Monitor both agent and cluster metrics

### Performance Analysis

- Establish clear performance criteria
- Use consistent test environments
- Measure multiple metrics simultaneously
- Analyze trends over time, not just peak values
- Document findings and optimization actions

### Continuous Testing

- Include performance tests in CI/CD pipelines
- Set up automated performance monitoring
- Create alerts for performance degradation
- Regular performance regression testing
- Performance budgets and SLA tracking

Load testing ensures the CloudZero Agent maintains reliable performance under production workloads and helps identify optimization opportunities.
