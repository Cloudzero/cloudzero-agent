# CloudZero Agent Domain Layer

The domain layer implements the Application Core in CloudZero Agent's hexagonal architecture, containing core business logic for cost allocation, metric processing, and Kubernetes resource management. Most services use dependency injection through types/ interfaces, though some services (shipper, backfiller) have direct HTTP client dependencies.

## Architecture Overview

```
Primary Adapters → Domain Services → Secondary Adapters
   (HTTP/CLI)     →  (Business Logic) →  (Storage/APIs)
```

## Core Services

### Cost Allocation Pipeline

- **MetricCollector**: Processes Prometheus remote_write metrics for cost attribution
- **MetricFilter**: High-performance classification of cost-relevant metrics
- **MetricShipper**: Reliable upload to CloudZero platform with retry logic

### Kubernetes Integration

- **WebhookController**: Admission control for resource cost allocation metadata
- **ResourceHandlers**: Per-resource-type logic for 20+ Kubernetes resources
- **BackfillService**: Historical resource discovery and metadata attribution

### Operational Services

- **HealthCheck**: Service health monitoring and dependency validation
- **Certificate**: TLS certificate lifecycle management for webhooks
- **Monitor**: Dynamic secret rotation and credential management
- **Diagnostic**: System health analysis and troubleshooting automation

## Package Organization

### Core Domain (`app/domain/`)

- **webhook/**: Kubernetes admission webhook business logic
- **shipper/**: CloudZero platform integration and data upload
- **filter/**: Metric classification and cost relevance analysis
- **monitor/**: Secret management and certificate lifecycle
- **healthz/**: Health checking and dependency validation

### Kubernetes Support (`app/domain/k8s/`)

- **client**: Kubernetes API client abstractions
- **resources**: Resource-specific processing logic

### Diagnostics (`app/domain/diagnostic/`)

- **catalog**: Diagnostic test registry and execution
- **k8s/**: Kubernetes cluster diagnostics
- **prom/**: Prometheus integration diagnostics
- **cz/**: CloudZero platform connectivity diagnostics

## Design Principles

### Hexagonal Architecture

- **Port Interfaces**: All external dependencies accessed through interfaces
- **Dependency Injection**: Services receive dependencies, never create them
- **Business Logic Purity**: No HTTP, database, or I/O concerns in domain services
- **Testability**: Clean interfaces enable comprehensive unit testing

### Error Handling

- **Typed Errors**: Domain-specific error types for different failure scenarios
- **Context Propagation**: Request contexts flow through all operations
- **Graceful Degradation**: Services continue operating with reduced functionality
- **Observability**: Comprehensive logging and metrics for operational monitoring

### Performance

- **Streaming Processing**: Large datasets processed without loading into memory
- **Concurrent Operations**: Parallel processing where safe and beneficial
- **Resource Pooling**: Reuse expensive resources like HTTP clients
- **Caching**: Strategic caching for frequently accessed data

## Integration Patterns

### Primary Adapters → Domain

```go
// HTTP handlers inject domain services
func NewRemoteWriteAPI(collector *domain.MetricCollector) *RemoteWriteAPI

// CLI functions use domain services
func runCollector(collector *domain.MetricCollector) error
```

### Domain → Secondary Adapters

```go
// Domain services use interface abstractions
type MetricCollector struct {
    store types.WritableStore  // Storage interface
    client types.HTTPClient    // HTTP interface
    filter types.MetricFilter  // Business logic interface
}
```

### Service Composition

```go
// Services compose through dependency injection
func NewWebhookController(
    store types.ResourceStore,
    handlers []webhook.ResourceHandler,
    validator types.PolicyValidator,
) *WebhookController
```

## Testing Approach

### Unit Testing

- **Mock Dependencies**: All interfaces have mock implementations
- **Business Logic Focus**: Test domain logic without external dependencies
- **Edge Case Coverage**: Comprehensive error condition testing
- **Performance Validation**: Benchmark critical paths

### Integration Testing

- **Service Boundaries**: Test integration between domain services
- **End-to-End Flows**: Complete request processing validation
- **Error Propagation**: Verify error handling across service boundaries
- **Resource Cleanup**: Ensure proper resource management

## Operational Considerations

### Monitoring

- **Metrics Collection**: Prometheus metrics for all domain operations
- **Health Checking**: Comprehensive health status reporting
- **Error Tracking**: Structured error logging for troubleshooting
- **Performance Monitoring**: Latency and throughput tracking

### Scalability

- **Stateless Design**: Services scale horizontally without coordination
- **Resource Efficiency**: Optimized memory and CPU usage patterns
- **Backpressure Handling**: Graceful handling of load spikes
- **Circuit Breaking**: Protection against cascading failures

### Security

- **Input Validation**: All external input validated and sanitized
- **Secret Management**: Secure handling of API keys and credentials
- **Access Control**: Authentication and authorization where required
- **Audit Logging**: Comprehensive audit trails for compliance

## Development Guidelines

### Adding New Domain Services

1. Define clear interface contracts in `app/types/`
2. Implement pure business logic without external dependencies
3. Use dependency injection for all external integrations
4. Add comprehensive unit tests with mocked dependencies
5. Document business requirements and architectural decisions

### Modifying Existing Services

1. Maintain interface compatibility or version appropriately
2. Add comprehensive tests for new functionality
3. Update documentation and architectural diagrams
4. Consider performance impact on existing operations
5. Validate error handling and edge cases

### Integration Patterns

1. Use interfaces defined in `app/types/` for all dependencies
2. Accept dependencies through constructor injection
3. Return domain-specific errors that can be handled appropriately
4. Implement comprehensive logging for operational visibility
5. Design for concurrent access and thread safety
