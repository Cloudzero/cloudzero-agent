# Domain Layer Architecture

## Overview

The domain layer represents the core business logic of the CloudZero Agent, implementing the business rules, workflows, and domain services that define how the agent collects, processes, and manages infrastructure cost data.

## Domain-Driven Design Principles

### Bounded Contexts

Each directory represents a distinct bounded context with clear responsibilities:

```
domain/
├── diagnostic/          # System health validation and monitoring
├── healthz/            # HTTP health endpoint management  
├── webhook/            # Kubernetes admission control
├── shipper/            # Metric data upload orchestration
├── monitor/            # File and certificate monitoring
├── k8s/               # Kubernetes API integration
└── filter/            # Metric filtering and transformation
```

### Domain Services

#### Metric Collection Domain (`metric_collector.go`)
- **Responsibility**: Core metrics ingestion from Prometheus remote write
- **Interactions**: Storage layer, configuration, diagnostic system
- **Key Operations**: Metric validation, enrichment, and persistence

#### Shipping Domain (`shipper/`)
- **Responsibility**: Metric file processing and CloudZero API upload
- **Interactions**: Disk storage, HTTP clients, parallel processing utilities
- **Key Operations**: File discovery, parallel upload, error handling, retry logic

#### Admission Control Domain (`webhook/`)
- **Responsibility**: Kubernetes resource validation and mutation
- **Interactions**: Kubernetes API, resource handlers, configuration
- **Key Operations**: Resource validation, policy enforcement, metadata injection

#### Diagnostic Domain (`diagnostic/`)
- **Responsibility**: System health monitoring and validation
- **Interactions**: External services, configuration, status reporting
- **Key Operations**: Health check execution, status aggregation, failure reporting

## Service Coordination Patterns

### Event-Driven Architecture
```go
// Components communicate through event bus
type Bus interface {
    Publish(event Event) error
    Subscribe(eventType string, handler EventHandler) error
}
```

### Service Lifecycle Management
1. **Initialization**: Service registration and dependency injection
2. **Health Registration**: Diagnostic provider registration
3. **Background Processing**: Goroutine management with context cancellation
4. **Graceful Shutdown**: Resource cleanup and pending operation completion

### Error Propagation Strategy
- **Recoverable Errors**: Log and continue with degraded functionality
- **Critical Errors**: Propagate to service coordinator for restart
- **External Service Errors**: Circuit breaker pattern with backoff

## Domain Service Implementations

### MetricCollector Service
```go
type MetricCollector struct {
    store      types.Store
    bus        types.Bus
    config     *config.Settings
    diagnostic diagnostic.Provider
}
```

**Responsibilities**:
- Prometheus remote write protocol handling
- Metric validation and enrichment with cluster metadata
- Storage coordination with atomic operations
- Real-time diagnostic reporting

### Shipper Service
```go
type Shipper struct {
    storage    disk.Store
    uploader   Uploader
    parallel   *parallel.Manager
    monitor    StoreMonitor
}
```

**Responsibilities**:
- Periodic file discovery and processing
- Parallel upload coordination with rate limiting
- Upload failure recovery and retry logic
- Storage cleanup after successful uploads

### Webhook Service
```go
type WebhookServer struct {
    handlers   map[string]ResourceHandler
    validator  Validator
    config     *webhook.Settings
}
```

**Responsibilities**:
- Kubernetes admission request processing
- Resource-specific validation and mutation
- Policy enforcement and compliance checking
- Audit logging and security monitoring

## Cross-Cutting Concerns

### Configuration Management
All domain services receive configuration through dependency injection:
- **Environment Detection**: Automatic cloud provider and cluster detection
- **Feature Flags**: Runtime behavior modification without deployment
- **Credential Management**: Secure handling of API keys and certificates

### Observability
- **Structured Logging**: Consistent log formats with correlation IDs
- **Metrics Collection**: Domain-specific performance and business metrics
- **Health Monitoring**: Continuous health status reporting
- **Distributed Tracing**: Request flow tracking across service boundaries

### Security
- **Input Validation**: All external inputs validated at domain boundaries
- **Authorization**: Role-based access control for sensitive operations
- **Audit Logging**: Security-relevant events logged with context
- **Secrets Management**: Encrypted storage and rotation of sensitive data

## Domain Integration Patterns

### Repository Pattern Integration
```go
// Domain services depend on repository interfaces
type MetricRepository interface {
    Store(ctx context.Context, metrics []types.Metric) error
    FindPending(ctx context.Context) ([]types.MetricFile, error)
}
```

### Event Sourcing for Audit
```go
// Important domain events are persisted for audit and replay
type DomainEvent struct {
    ID        uuid.UUID
    Type      string
    Aggregate string
    Data      json.RawMessage
    Timestamp time.Time
}
```

### Command Query Responsibility Segregation (CQRS)
- **Commands**: State-changing operations (metric ingestion, file upload)
- **Queries**: Read-only operations (health status, diagnostic information)
- **Separate Models**: Optimized data structures for read vs write operations

## Domain Model Relationships

### Core Entities
- **Metric**: Time-series data point with metadata and value
- **MetricFile**: Aggregated metrics stored in compressed files
- **ClusterResource**: Kubernetes resources with cost attribution metadata
- **DiagnosticResult**: Health check outcomes and system status

### Value Objects
- **CloudMetadata**: Immutable cloud provider and account information
- **TimeRange**: Immutable time period with validation logic
- **ResourceIdentifier**: Composite identifier for Kubernetes resources

### Aggregates
- **MetricBatch**: Collection of related metrics with transactional boundaries
- **UploadSession**: Group of files uploaded together with retry logic
- **DiagnosticReport**: Comprehensive system health assessment

## Performance Considerations

### Throughput Optimization
- **Batch Processing**: Group operations to reduce overhead
- **Parallel Execution**: Utilize multiple CPU cores effectively
- **Connection Pooling**: Reuse HTTP connections and database connections
- **Compression**: Reduce storage and network overhead

### Latency Optimization  
- **Asynchronous Processing**: Non-blocking operations where possible
- **Local Caching**: Reduce external service dependencies
- **Connection Reuse**: Minimize connection establishment overhead
- **Resource Preallocation**: Avoid allocation in hot paths

### Resource Management
- **Memory Bounds**: Configurable limits on in-memory data structures  
- **Disk Space Management**: Automatic cleanup of processed files
- **Connection Limits**: Prevent resource exhaustion
- **Graceful Degradation**: Maintain core functionality under load

## Testing Strategy

### Unit Testing
- **Domain Logic**: Pure functions and business rule validation
- **Mock Dependencies**: External services and storage layers
- **Edge Cases**: Error conditions and boundary values
- **Performance**: Critical path latency and throughput

### Integration Testing
- **Service Interactions**: Real dependencies with test fixtures
- **Database Operations**: Transactional behavior and consistency
- **External APIs**: Contract testing with CloudZero platform
- **Event Processing**: Asynchronous workflow validation

### End-to-End Testing
- **Complete Workflows**: Metric ingestion through upload completion
- **Error Scenarios**: Failure modes and recovery behavior
- **Performance Testing**: Load testing under realistic conditions
- **Chaos Engineering**: Resilience under adverse conditions

This domain architecture enables the CloudZero Agent to maintain clean separation of concerns while providing robust, scalable, and maintainable cost monitoring capabilities.