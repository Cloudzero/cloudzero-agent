# CloudZero Agent HTTP Handlers

The handlers package implements Primary Adapters in CloudZero Agent's hexagonal architecture, providing HTTP interfaces that translate external requests into domain service operations for cost allocation processing.

## Architecture Overview

```
External Systems → HTTP Handlers → Domain Services → Storage/APIs
  (Kubernetes)   → (Primary Adapters) → (Business Logic) → (Secondary Adapters)
```

## Core Handlers

### Admission Webhook API (`webhook.go`)

- **ValidationWebhookAPI**: Kubernetes admission control for cost allocation metadata
- **Fail-Open Design**: Always allows resources if processing fails to prevent cluster disruption
- **Multi-Version Support**: Compatible with admission.k8s.io/v1 and v1beta1 APIs
- **Request Validation**: Content-type checking, timeout management, size limits
- **Connection Management**: Periodic closure for load balancing across replicas

### Prometheus Remote Write API (`remote_write.go`)

- **RemoteWriteAPI**: High-throughput metric ingestion from Prometheus instances
- **Protocol Support**: Full remote_write v1 and v2 specification compliance
- **Compression Handling**: Snappy decompression for efficient network transmission
- **Size Limits**: 16MB maximum payload with memory protection
- **Load Balancing**: Connection management for distributing Prometheus load

### Metrics API (`prom_metrics.go`)

- **PromMetricsAPI**: Prometheus metrics exposition for operational monitoring
- **Standard Endpoint**: `/metrics` endpoint following Prometheus conventions
- **Comprehensive Metrics**: Agent performance, business metrics, and infrastructure data
- **Auto-Discovery**: Compatible with Kubernetes service discovery patterns

### Shipper API (`shipper.go`)

- **ShipperAPI**: Operational endpoints for metric shipping monitoring
- **Debug Capabilities**: Shipping pipeline status and performance insights
- **Health Checking**: Integration with Kubernetes liveness and readiness probes
- **Prometheus Integration**: Internal metrics for shipping operations monitoring

### Profiling API (`profiling.go`)

- **ProfilingAPI**: Go pprof profiling endpoints for performance analysis
- **Standard Endpoints**: Complete pprof interface for development and debugging
- **Security Considerations**: Should be restricted in production environments
- **Tool Compatibility**: Direct integration with `go tool pprof` and analysis tools

## Request Processing Patterns

### Kubernetes Admission Control

```go
func (a *ValidationWebhookAPI) PostAdmissionRequest(w http.ResponseWriter, r *http.Request) {
    // 1. Request validation and timeout setup
    ctx, cancel := context.WithTimeout(r.Context(), DefaultTimeout)
    defer cancel()

    // 2. Parse admission review from request body
    review, err := a.requestBodyToModelReview(body)

    // 3. Process through domain service
    _, err = a.controller.Review(ctx, review)

    // 4. Always allow resources (fail-open behavior)
    sendAllowResponse(w, r)
}
```

### Prometheus Metric Ingestion

```go
func (a *RemoteWriteAPI) PostMetrics(w http.ResponseWriter, r *http.Request) {
    // 1. Validate request size and content type
    if r.ContentLength > MaxPayloadSize {
        logErrorReply(r, w, "too big", http.StatusOK)
        return
    }

    // 2. Read and process metric data
    data, err := io.ReadAll(r.Body)
    stats, err := a.metrics.PutMetrics(r.Context(), contentType, encodingType, data)

    // 3. Implement load balancing through connection management
    if r.ProtoMajor == 1 && shouldCloseConnection() {
        w.Header().Set("Connection", "close")
    }

    // 4. Return processing statistics
    if stats != nil {
        stats.SetHeaders(w)
    }
}
```

## HTTP Infrastructure Integration

### Server Integration (`go-obvious/server`)

- **Consistent Patterns**: All handlers implement `server.API` interface
- **Middleware Support**: Logging, metrics, authentication, and CORS handling
- **Graceful Shutdown**: Coordinated shutdown with proper connection drainage
- **Health Checking**: Built-in health and readiness endpoint support

### Router Configuration (`chi`)

- **High Performance**: Chi router optimized for high-throughput operations
- **RESTful Patterns**: Standard HTTP method and path conventions
- **Middleware Chains**: Composable request processing pipeline
- **Route Groups**: Logical organization of related endpoints

### Error Handling

```go
// Consistent error response pattern
func logErrorReply(r *http.Request, w http.ResponseWriter, data string, statusCode int) {
    log.Ctx(r.Context()).Error().Msg(data)
    request.Reply(r, w, data, statusCode)
}

// Fail-open behavior for critical operations
func sendAllowResponse(w http.ResponseWriter, r *http.Request) {
    allowResponse := &types.AdmissionResponse{Allowed: true}
    resp, err := marshallResponseToJSON(ctx, review, allowResponse)
    if err != nil {
        // Use minimal JSON response to ensure we always allow
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(minimalAllowResponse))
        return
    }
    w.WriteHeader(http.StatusOK)
    w.Write(resp)
}
```

## Security Considerations

### Request Validation

- **Content-Type Verification**: Strict content type checking for all endpoints
- **Size Limits**: Maximum request body sizes to prevent memory exhaustion
- **Timeout Management**: Request-level timeouts to prevent resource exhaustion
- **Input Sanitization**: Validation of all external input data

### Authentication and Authorization

- **Kubernetes Integration**: Service account-based authentication for webhooks
- **API Keys**: CloudZero platform authentication for metric uploads
- **Network Security**: TLS encryption for all external communications
- **Access Logging**: Comprehensive request logging for audit and monitoring

### Production Hardening

- **Rate Limiting**: Protection against abuse and overload conditions
- **Circuit Breaking**: Protection against cascading failures
- **Health Monitoring**: Comprehensive health checking and alerting
- **Resource Limits**: Memory and CPU usage controls for stability

## Performance Characteristics

### High-Throughput Design

- **Concurrent Processing**: Handlers designed for concurrent request handling
- **Memory Efficiency**: Streaming processing for large payloads
- **Connection Pooling**: Efficient reuse of HTTP connections
- **Load Balancing**: Request distribution across multiple agent replicas

### Prometheus Integration Optimization

- **Remote Write Efficiency**: Optimized for high-volume metric ingestion
- **Compression Support**: Snappy decompression for bandwidth efficiency
- **Batch Processing**: Efficient handling of large metric batches
- **Connection Management**: HTTP/1.1 connection cycling for load distribution

### Kubernetes Webhook Optimization

- **Fast Response Times**: Sub-second admission control decisions
- **Fail-Open Design**: Never blocks cluster operations due to agent issues
- **Request Correlation**: Structured logging for debugging and monitoring
- **Resource Efficiency**: Minimal memory and CPU usage per request

## Monitoring and Observability

### Request Metrics

```go
// HTTP request metrics
var (
    httpRequestsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{Name: "http_requests_total"},
        []string{"method", "path", "status"},
    )

    httpRequestDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{Name: "http_request_duration_seconds"},
        []string{"method", "path"},
    )
)
```

### Health Monitoring

- **Endpoint Health**: Individual handler health status reporting
- **Dependency Status**: Database and external service connectivity
- **Performance Metrics**: Request latency and throughput tracking
- **Error Rate Monitoring**: Error classification and trending

### Distributed Tracing

- **Request Correlation**: Trace ID propagation through request processing
- **Service Boundaries**: Span creation at handler and service boundaries
- **Performance Analysis**: End-to-end request timing and bottleneck identification
- **Error Attribution**: Trace-based error tracking and root cause analysis

## Development Guidelines

### Adding New Handlers

1. **Implement API Interface**: Create struct implementing `server.API`
2. **Define Routes**: Configure chi router with appropriate HTTP methods
3. **Add Validation**: Implement request validation and error handling
4. **Domain Integration**: Delegate business logic to domain services
5. **Add Tests**: Comprehensive unit and integration testing

### Testing Strategies

```go
// Unit testing with mocked dependencies
func TestValidationWebhookAPI_PostAdmissionRequest(t *testing.T) {
    mockController := &mocks.WebhookController{}
    api := NewValidationWebhookAPI("/webhook", mockController)

    req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(testAdmissionReview))
    w := httptest.NewRecorder()

    api.PostAdmissionRequest(w, req)

    assert.Equal(t, http.StatusOK, w.Code)
    mockController.AssertExpectations(t)
}

// Integration testing with real HTTP server
func TestRemoteWriteAPI_Integration(t *testing.T) {
    collector := setupTestCollector(t)
    api := NewRemoteWriteAPI("/api/v1/write", collector)

    server := httptest.NewServer(api.Routes())
    defer server.Close()

    // Test with real Prometheus remote_write data
    resp, err := http.Post(server.URL+"/", "application/x-protobuf", bytes.NewReader(testMetrics))
    require.NoError(t, err)
    assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}
```

### Error Handling Best Practices

- **Consistent Response Format**: Standardized error response structure
- **Appropriate Status Codes**: Correct HTTP status codes for different error types
- **Detailed Logging**: Comprehensive error logging for troubleshooting
- **Client-Friendly Messages**: Clear error messages for API consumers

### Performance Optimization

- **Request Parsing**: Efficient parsing of request bodies and headers
- **Memory Management**: Minimize allocations and enable garbage collection
- **Connection Handling**: Proper connection lifecycle management
- **Resource Cleanup**: Ensure proper cleanup of resources and goroutines
