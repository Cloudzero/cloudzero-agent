# CloudZero Agent API Reference

## Overview

The CloudZero Agent exposes several HTTP APIs for different operational and integration purposes. This document provides comprehensive reference information for all available endpoints.

## Base URLs and Endpoints

### Collector Service (DaemonSet)
- **Base URL**: `http://<node-ip>:8080` (typically accessed via port-forward)
- **Purpose**: Metric collection, health monitoring, diagnostics

### Webhook Service (Deployment)  
- **Base URL**: `https://<webhook-service>:9443`
- **Purpose**: Kubernetes admission control, validation webhooks
- **Authentication**: TLS client certificates (managed by Kubernetes)

### Monitoring Endpoints
- **Metrics**: `http://<service>:8080/metrics` (Prometheus format)
- **Health**: `http://<service>:8080/healthz` 
- **Diagnostics**: `http://<service>:8080/diagnostic`

## Health and Monitoring APIs

### GET /healthz

**Purpose**: Basic health check endpoint for monitoring systems

**Request**:
```http
GET /healthz HTTP/1.1
Host: localhost:8080
```

**Response** (Healthy):
```http
HTTP/1.1 200 OK
Content-Type: text/plain
Content-Length: 2

ok
```

**Response** (Unhealthy):
```http
HTTP/1.1 500 Internal Server Error
Content-Type: text/plain

database failed: connection timeout
```

**Usage**:
- Kubernetes liveness and readiness probes
- Load balancer health checks
- Monitoring system integration
- Automated alerting systems

### GET /diagnostic

**Purpose**: Comprehensive system diagnostic information

**Request**:
```http
GET /diagnostic HTTP/1.1
Host: localhost:8080
Accept: application/json
```

**Response**:
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "timestamp": "2024-01-15T10:30:00Z",
  "agent_version": "1.2.6",
  "cluster_name": "production-cluster",
  "node_name": "worker-node-1",
  "checks": {
    "cloudzero_api": {
      "status": "healthy",
      "last_check": "2024-01-15T10:29:30Z",
      "response_time_ms": 150,
      "details": {
        "endpoint": "https://api.cloudzero.com",
        "auth_status": "valid"
      }
    },
    "prometheus": {
      "status": "healthy", 
      "last_check": "2024-01-15T10:29:45Z",
      "metrics_received": 15420,
      "details": {
        "remote_write_url": "http://prometheus:9090/api/v1/write",
        "last_receive": "2024-01-15T10:29:44Z"
      }
    },
    "storage": {
      "status": "healthy",
      "disk_usage": {
        "total_bytes": 10737418240,
        "used_bytes": 2147483648,
        "available_bytes": 8589934592,
        "usage_percent": 20.0
      },
      "pending_files": 3,
      "details": {
        "storage_path": "/var/lib/agent/metrics",
        "compression_enabled": true
      }
    },
    "kubernetes": {
      "status": "healthy",
      "api_server": "https://kubernetes.default.svc",
      "permissions": "valid",
      "details": {
        "service_account": "cloudzero-agent",
        "namespace": "cloudzero-system"
      }
    }
  },
  "performance": {
    "uptime_seconds": 86400,
    "memory_usage_bytes": 134217728,
    "cpu_usage_percent": 5.2,
    "goroutines": 45,
    "gc_stats": {
      "num_gc": 157,
      "pause_total_ns": 12500000
    }
  }
}
```

**Error Response**:
```http
HTTP/1.1 503 Service Unavailable
Content-Type: application/json

{
  "timestamp": "2024-01-15T10:30:00Z",
  "error": "diagnostic system unavailable",
  "details": "unable to collect system information"
}
```

### GET /metrics

**Purpose**: Prometheus-format metrics for monitoring integration

**Request**:
```http
GET /metrics HTTP/1.1  
Host: localhost:8080
Accept: text/plain
```

**Response**:
```http
HTTP/1.1 200 OK
Content-Type: text/plain; version=0.0.4; charset=utf-8

# HELP cloudzero_metrics_collected_total Total number of metrics collected
# TYPE cloudzero_metrics_collected_total counter
cloudzero_metrics_collected_total{cluster="production",node="worker-1"} 245680

# HELP cloudzero_metrics_processed_total Total number of metrics processed
# TYPE cloudzero_metrics_processed_total counter  
cloudzero_metrics_processed_total{cluster="production",node="worker-1"} 245650

# HELP cloudzero_storage_disk_usage_bytes Current disk usage in bytes
# TYPE cloudzero_storage_disk_usage_bytes gauge
cloudzero_storage_disk_usage_bytes{cluster="production",node="worker-1"} 2147483648

# HELP cloudzero_storage_files_pending Number of files pending upload
# TYPE cloudzero_storage_files_pending gauge
cloudzero_storage_files_pending{cluster="production",node="worker-1"} 3

# HELP cloudzero_errors_total Total number of errors by component
# TYPE cloudzero_errors_total counter
cloudzero_errors_total{cluster="production",component="collector",type="validation"} 12
cloudzero_errors_total{cluster="production",component="shipper",type="upload"} 3
```

## Metric Collection APIs

### POST /api/v1/receive

**Purpose**: Prometheus remote write endpoint for metric ingestion

**Request**:
```http
POST /api/v1/receive HTTP/1.1
Host: localhost:8080
Content-Type: application/x-protobuf
Content-Encoding: snappy
X-Prometheus-Remote-Write-Version: 0.1.0

[Binary Protobuf Data - Prometheus WriteRequest]
```

**Response** (Success):
```http
HTTP/1.1 204 No Content
```

**Response** (Error):
```http
HTTP/1.1 400 Bad Request
Content-Type: application/json

{
  "error": "invalid metric data",
  "details": "timestamp out of range",
  "rejected_samples": 15
}
```

**Request Format**: 
The request body contains a Prometheus `WriteRequest` protobuf message with the following structure:

```protobuf
message WriteRequest {
  repeated TimeSeries timeseries = 1;
}

message TimeSeries {
  repeated Label labels = 1;
  repeated Sample samples = 2;
}

message Label {
  string name = 1;
  string value = 2;
}

message Sample {
  double value = 1;
  int64 timestamp = 2;  // Unix timestamp in milliseconds
}
```

**Usage Examples**:
```yaml
# Prometheus configuration
remote_write:
  - url: "http://cloudzero-agent:8080/api/v1/receive"
    write_relabel_configs:
      - source_labels: [__name__]
        regex: 'kube_.*|node_.*|container_.*'
        action: keep
```

## Webhook Admission APIs

### POST /validate

**Purpose**: Kubernetes admission webhook for resource validation

**Request**:
```http
POST /validate HTTP/1.1
Host: webhook-service:9443
Content-Type: application/json
Authorization: Bearer <kubernetes-token>

{
  "kind": "AdmissionReview",
  "apiVersion": "admission.k8s.io/v1",
  "request": {
    "uid": "705ab4f5-6393-11e8-b7cc-42010a800002",
    "kind": {
      "group": "apps",
      "version": "v1", 
      "kind": "Deployment"
    },
    "resource": {
      "group": "apps",
      "version": "v1",
      "resource": "deployments"
    },
    "namespace": "default",
    "operation": "CREATE",
    "object": {
      "apiVersion": "apps/v1",
      "kind": "Deployment",
      "metadata": {
        "name": "test-deployment",
        "namespace": "default"
      },
      "spec": {
        "replicas": 3,
        "selector": {
          "matchLabels": {
            "app": "test"
          }
        },
        "template": {
          "metadata": {
            "labels": {
              "app": "test"
            }
          },
          "spec": {
            "containers": [
              {
                "name": "nginx",
                "image": "nginx:1.21"
              }
            ]
          }
        }
      }
    }
  }
}
```

**Response** (Allow):
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "kind": "AdmissionReview",
  "apiVersion": "admission.k8s.io/v1",
  "response": {
    "uid": "705ab4f5-6393-11e8-b7cc-42010a800002",
    "allowed": true,
    "status": {
      "code": 200
    },
    "patch": "W3sib3AiOiJhZGQiLCJwYXRoIjoiL21ldGFkYXRhL2xhYmVscyIsInZhbHVlIjp7ImNsb3VkemVyby5jb20vY2x1c3RlciI6InByb2R1Y3Rpb24ifX1d",
    "patchType": "JSONPatch"
  }
}
```

**Response** (Deny):
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "kind": "AdmissionReview",
  "apiVersion": "admission.k8s.io/v1", 
  "response": {
    "uid": "705ab4f5-6393-11e8-b7cc-42010a800002",
    "allowed": false,
    "status": {
      "code": 403,
      "message": "Resource does not meet cost allocation requirements: missing cost center label"
    }
  }
}
```

**Patch Operations**: The webhook can modify resources using JSON Patch operations:
```json
[
  {
    "op": "add",
    "path": "/metadata/labels/cloudzero.com~1cluster", 
    "value": "production"
  },
  {
    "op": "add",
    "path": "/metadata/annotations/cloudzero.com~1cost-center",
    "value": "engineering"
  }
]
```

### POST /mutate

**Purpose**: Kubernetes mutating admission webhook for resource enhancement

**Request**: Same format as `/validate` endpoint

**Response**:
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "kind": "AdmissionReview",
  "apiVersion": "admission.k8s.io/v1",
  "response": {
    "uid": "705ab4f5-6393-11e8-b7cc-42010a800002",
    "allowed": true,
    "status": {
      "code": 200  
    },
    "patch": "W3sib3AiOiJhZGQiLCJwYXRoIjoiL21ldGFkYXRhL2xhYmVscyIsInZhbHVlIjp7ImNsb3VkemVyby5jb20vY2x1c3RlciI6InByb2R1Y3Rpb24iLCJjbG91ZHplcm8uY29tL2Vudmlyb25tZW50IjoicHJvZCJ9fV0=",
    "patchType": "JSONPatch"
  }
}
```

## Internal APIs

### GET /api/internal/status

**Purpose**: Internal status endpoint for agent coordination

**Authentication**: Internal service-to-service authentication required

**Request**:
```http
GET /api/internal/status HTTP/1.1
Host: localhost:8080
Authorization: Bearer <internal-token>
```

**Response**:
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "component": "collector",
  "status": "healthy",
  "last_activity": "2024-01-15T10:29:45Z",
  "metrics": {
    "pending_files": 3,
    "processing_queue": 150,
    "upload_queue": 2
  },
  "configuration": {
    "cluster_name": "production",
    "storage_path": "/var/lib/agent/metrics"
  }
}
```

### POST /api/internal/coordinate

**Purpose**: Inter-component coordination and synchronization

**Request**:
```http
POST /api/internal/coordinate HTTP/1.1
Host: localhost:8080
Content-Type: application/json
Authorization: Bearer <internal-token>

{
  "action": "flush_metrics",
  "parameters": {
    "force": true,
    "max_age": "5m"
  }
}
```

**Response**:
```http
HTTP/1.1 200 OK
Content-Type: application/json

{
  "status": "accepted",
  "request_id": "req-12345",
  "estimated_completion": "2024-01-15T10:31:00Z"
}
```

## Error Responses

### Standard Error Format
All APIs use a consistent error response format:

```json
{
  "error": "error_type",
  "message": "Human readable error description",
  "details": "Additional context or troubleshooting information",
  "timestamp": "2024-01-15T10:30:00Z",
  "request_id": "req-67890"
}
```

### Common Error Codes

| Status Code | Error Type | Description |
|-------------|------------|-------------|
| 400 | `bad_request` | Invalid request format or parameters |
| 401 | `unauthorized` | Missing or invalid authentication |
| 403 | `forbidden` | Insufficient permissions |
| 404 | `not_found` | Resource or endpoint not found |
| 429 | `rate_limited` | Too many requests |
| 500 | `internal_error` | Server-side error |
| 503 | `unavailable` | Service temporarily unavailable |

## Rate Limiting

### Limits by Endpoint

| Endpoint | Rate Limit | Window |
|----------|------------|--------|
| `/healthz` | 100 req/min | Per client IP |
| `/metrics` | 60 req/min | Per client IP |
| `/diagnostic` | 10 req/min | Per client IP |
| `/api/v1/receive` | 1000 req/min | Per cluster |
| `/validate` | 500 req/min | Per cluster |
| `/mutate` | 500 req/min | Per cluster |

### Rate Limit Headers
```http
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1642248600
```

## Authentication

### Kubernetes Webhook Authentication
- **Method**: TLS client certificates
- **Managed by**: Kubernetes API server
- **Certificate rotation**: Automatic via cert-manager or manual process

### Internal API Authentication
- **Method**: Bearer tokens
- **Scope**: Service-to-service communication
- **Rotation**: Automatic via Kubernetes service account tokens

### External API Authentication
- **Method**: API keys (CloudZero platform)
- **Headers**: `Authorization: Bearer <api-key>`
- **Validation**: Against CloudZero API endpoints

## SDK and Client Libraries

### Go Client Example
```go
import "github.com/cloudzero/agent-client-go"

client := agentclient.New(&agentclient.Config{
    BaseURL: "http://cloudzero-agent:8080",
    Timeout: 30 * time.Second,
})

status, err := client.GetHealthStatus(context.Background())
if err != nil {
    log.Fatalf("Health check failed: %v", err)
}

fmt.Printf("Agent status: %s\n", status.Status)
```

### Python Client Example
```python
from cloudzero_agent_client import AgentClient

client = AgentClient(base_url="http://cloudzero-agent:8080")

try:
    status = client.get_health_status()
    print(f"Agent status: {status.status}")
except Exception as e:
    print(f"Health check failed: {e}")
```

This API reference provides comprehensive information for integrating with and monitoring the CloudZero Agent. For operational guidance, see the [Operations Guide](OPERATIONS_GUIDE.md).