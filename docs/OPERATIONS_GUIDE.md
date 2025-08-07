# CloudZero Agent Operations Guide

## Overview

This guide provides comprehensive operational information for deploying, monitoring, and maintaining the CloudZero Kubernetes Agent in production environments.

## System Architecture

The CloudZero Agent consists of several key components that work together to provide comprehensive cost monitoring:

- **Metric Collector**: Ingests metrics from Prometheus remote write endpoints
- **Metric Shipper**: Processes and uploads metric data to CloudZero APIs  
- **Webhook Server**: Kubernetes admission controller for resource validation
- **Diagnostic System**: Health monitoring and system validation
- **Storage Layer**: High-performance disk and database storage

For detailed architecture information, see [`app/ARCHITECTURE.md`](../app/ARCHITECTURE.md).

## Deployment Architecture

### Production Deployment Pattern
```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Kubernetes    │    │    CloudZero    │    │   Monitoring    │
│    Cluster      │    │     Platform    │    │     Stack       │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          ▼                      ▼                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                    CloudZero Agent Deployment                  │
├─────────────────┬─────────────────┬─────────────────────────────┤
│   DaemonSet     │   Deployment    │      ConfigMaps &           │
│  (Collector)    │  (Webhook)      │       Secrets               │
└─────────────────┴─────────────────┴─────────────────────────────┘
```

### Component Deployment Strategy

#### DaemonSet (Metric Collector)
- **Purpose**: Runs on every node for comprehensive metric collection
- **Resource Requirements**: CPU: 100m-500m, Memory: 128Mi-512Mi
- **Storage**: Persistent volume for metric buffering
- **Network**: Access to Prometheus and CloudZero APIs

#### Deployment (Webhook Server)
- **Purpose**: Kubernetes admission controller
- **Replicas**: 2-3 for high availability
- **Resource Requirements**: CPU: 50m-200m, Memory: 64Mi-256Mi
- **Certificates**: TLS certificates for webhook validation

## Installation and Configuration

### Helm Installation
```bash
# Add CloudZero Helm repository
helm repo add cloudzero https://charts.cloudzero.com
helm repo update

# Install with production values
helm install cloudzero-agent cloudzero/cloudzero-agent \
  --namespace cloudzero-system \
  --create-namespace \
  --values production-values.yaml

# Verify deployment
kubectl get pods -n cloudzero-system
kubectl get daemonset -n cloudzero-system
```

### Production Configuration
```yaml
# production-values.yaml
global:
  cloudAccountId: "123456789012"
  clusterName: "production-cluster"

cloudzero:
  apiKey: "${CLOUDZERO_API_KEY}"
  host: "https://api.cloudzero.com"

storage:
  persistentVolume:
    enabled: true
    size: 10Gi
    storageClass: "gp3"

resources:
  collector:
    requests:
      cpu: 200m
      memory: 256Mi
    limits:
      cpu: 500m
      memory: 512Mi
      
  webhook:
    requests:
      cpu: 100m
      memory: 128Mi
    limits:
      cpu: 200m
      memory: 256Mi

monitoring:
  enabled: true
  serviceMonitor:
    enabled: true
```

## Monitoring and Observability

### Health Check Endpoints

#### Collector Health Check
```bash
# Check collector health
kubectl port-forward daemonset/cloudzero-agent-collector 8080:8080
curl http://localhost:8080/healthz

# Expected response: 200 OK "ok"
```

#### Webhook Health Check  
```bash
# Check webhook server health
kubectl port-forward deployment/cloudzero-agent-webhook 9443:9443
curl -k https://localhost:9443/healthz

# Expected response: 200 OK "ok"
```

#### Diagnostic Information
```bash
# Get comprehensive diagnostic information
curl http://localhost:8080/diagnostic

# Returns JSON with system status, connectivity, and performance metrics
```

### Prometheus Metrics

The agent exposes several metrics for monitoring:

```prometheus
# Metric collection rates
cloudzero_metrics_collected_total{cluster="prod"} 
cloudzero_metrics_processed_total{cluster="prod"}
cloudzero_metrics_uploaded_total{cluster="prod"}

# Storage utilization
cloudzero_storage_disk_usage_bytes{cluster="prod"}
cloudzero_storage_files_pending{cluster="prod"}

# Webhook performance
cloudzero_webhook_requests_total{cluster="prod", operation="validate"}
cloudzero_webhook_duration_seconds{cluster="prod"}

# Error rates
cloudzero_errors_total{cluster="prod", component="collector"}
```

### Grafana Dashboard

Key metrics to monitor:

```json
{
  "dashboard": {
    "title": "CloudZero Agent",
    "panels": [
      {
        "title": "Metric Collection Rate",
        "targets": [
          "rate(cloudzero_metrics_collected_total[5m])"
        ]
      },
      {
        "title": "Storage Usage", 
        "targets": [
          "cloudzero_storage_disk_usage_bytes"
        ]
      },
      {
        "title": "Error Rate",
        "targets": [
          "rate(cloudzero_errors_total[5m])"
        ]
      }
    ]
  }
}
```

## Troubleshooting Guide

### Common Issues

#### 1. Metrics Not Being Collected
**Symptoms**: Low or zero metric collection rates
**Diagnosis**:
```bash
# Check collector logs
kubectl logs daemonset/cloudzero-agent-collector

# Verify Prometheus configuration
kubectl get configmap prometheus-config -o yaml

# Check network connectivity
kubectl exec -it cloudzero-agent-collector-xxxx -- nc -zv prometheus 9090
```

**Solutions**:
- Verify Prometheus remote write configuration
- Check network policies and firewall rules
- Validate API keys and authentication
- Ensure sufficient resources allocated

#### 2. Webhook Admission Failures
**Symptoms**: Kubernetes resource creation failures
**Diagnosis**:
```bash
# Check webhook logs
kubectl logs deployment/cloudzero-agent-webhook

# Verify webhook configuration
kubectl get validatingwebhookconfiguration cloudzero-agent

# Test webhook endpoint
kubectl port-forward deployment/cloudzero-agent-webhook 9443:9443
curl -k https://localhost:9443/validate
```

**Solutions**:
- Verify TLS certificates are valid and not expired
- Check webhook service and endpoint configuration
- Validate RBAC permissions for webhook service account
- Ensure webhook server has sufficient resources

#### 3. Storage Issues  
**Symptoms**: High disk usage, failed uploads
**Diagnosis**:
```bash
# Check storage usage
kubectl exec cloudzero-agent-collector-xxxx -- df -h /var/lib/agent

# Review storage configuration
kubectl get pvc -n cloudzero-system

# Check upload logs
kubectl logs cloudzero-agent-collector-xxxx | grep "upload"
```

**Solutions**:
- Increase persistent volume size
- Adjust metric retention and compression settings
- Verify CloudZero API connectivity and authentication
- Monitor and clean up failed upload files

### Log Analysis

#### Critical Log Patterns
```bash
# Successful metric processing
kubectl logs daemonset/cloudzero-agent-collector | grep "metrics processed successfully"

# Authentication errors
kubectl logs daemonset/cloudzero-agent-collector | grep "authentication failed"

# Storage warnings
kubectl logs daemonset/cloudzero-agent-collector | grep "storage warning"

# Webhook validation errors  
kubectl logs deployment/cloudzero-agent-webhook | grep "validation failed"
```

#### Log Levels and Debugging
```yaml
# Enable debug logging
logging:
  level: debug
  format: json
  
# Structured logging fields
{
  "level": "info",
  "component": "collector", 
  "operation": "process_metrics",
  "duration_ms": 150,
  "metric_count": 1000,
  "timestamp": "2024-01-15T10:30:00Z"
}
```

## Performance Optimization

### Resource Tuning

#### Collector Optimization
```yaml
# High-throughput configuration
collector:
  resources:
    requests:
      cpu: 500m
      memory: 512Mi
    limits:
      cpu: 1000m
      memory: 1Gi
      
  config:
    batchSize: 5000
    flushInterval: 30s
    compressionLevel: 6
    maxFileSize: 100MB
```

#### Storage Optimization
```yaml
storage:
  config:
    compressionLevel: 6  # Balance CPU vs storage
    maxRecords: 10000   # Records per file
    maxInterval: 60s    # Maximum file age
    retentionDays: 7    # Local retention period
```

### Network Optimization
```yaml
network:
  config:
    maxConnections: 100
    timeoutSeconds: 30
    retryAttempts: 3
    backoffMultiplier: 2
```

## Security Considerations

### RBAC Configuration
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: cloudzero-agent
rules:
- apiGroups: [""]
  resources: ["nodes", "pods", "services"]
  verbs: ["get", "list", "watch"]
- apiGroups: ["apps"]
  resources: ["deployments", "daemonsets"]
  verbs: ["get", "list", "watch"]
```

### Secret Management
```yaml
# Use Kubernetes secrets for sensitive data
apiVersion: v1
kind: Secret
metadata:
  name: cloudzero-agent-secrets
data:
  api-key: <base64-encoded-api-key>
  tls-cert: <base64-encoded-certificate>
  tls-key: <base64-encoded-private-key>
```

### Network Policies
```yaml
# Restrict network access
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: cloudzero-agent-policy
spec:
  podSelector:
    matchLabels:
      app: cloudzero-agent
  egress:
  - to: []
    ports:
    - protocol: TCP
      port: 443  # CloudZero API
    - protocol: TCP  
      port: 9090 # Prometheus
```

## Backup and Recovery

### Configuration Backup
```bash
# Backup Helm values
helm get values cloudzero-agent > backup-values.yaml

# Backup Kubernetes configurations
kubectl get configmap,secret,pvc -n cloudzero-system -o yaml > backup-configs.yaml
```

### Data Recovery
```bash
# Restore from persistent volume snapshots
kubectl apply -f pvc-snapshot.yaml

# Restart components to pick up restored data
kubectl rollout restart daemonset/cloudzero-agent-collector
kubectl rollout restart deployment/cloudzero-agent-webhook
```

## Upgrade Procedures

### Rolling Upgrades
```bash
# Update Helm chart
helm repo update
helm upgrade cloudzero-agent cloudzero/cloudzero-agent \
  --values production-values.yaml \
  --wait \
  --timeout 10m

# Verify upgrade
kubectl rollout status daemonset/cloudzero-agent-collector
kubectl rollout status deployment/cloudzero-agent-webhook
```

### Rollback Procedures
```bash
# Rollback to previous version
helm rollback cloudzero-agent

# Verify rollback
helm list
kubectl get pods -n cloudzero-system
```

## Maintenance Tasks

### Regular Maintenance
- **Weekly**: Review resource usage and performance metrics
- **Monthly**: Update agent to latest version and validate functionality
- **Quarterly**: Review and update security configurations and certificates

### Automated Maintenance
```yaml
# CronJob for cleanup tasks
apiVersion: batch/v1
kind: CronJob
metadata:
  name: cloudzero-agent-cleanup
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: cleanup
            image: cloudzero/agent-tools
            command: ["/bin/sh"]
            args: ["-c", "cleanup-old-files.sh"]
```

This operations guide provides the foundation for reliable production deployment and maintenance of the CloudZero Agent. For development-specific information, see the [Developer Guide](DEVELOPER_GUIDE.md).