# CloudZero Agent Developer Guide

## Getting Started

This guide provides comprehensive information for developers working on the CloudZero Kubernetes Agent. The agent is a sophisticated cost monitoring and resource optimization platform built with Go and designed for production Kubernetes environments.

## Prerequisites

### Development Environment
- **Go**: 1.21 or later
- **Docker**: For containerized development and testing
- **Kubernetes**: Access to a development cluster (kind, minikube, or cloud)
- **Helm**: 3.x for chart development and testing

### Required Tools
```bash
# Core development tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/onsi/ginkgo/v2/ginkgo@latest
go install honnef.co/go/tools/cmd/staticcheck@latest

# Kubernetes tools  
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
chmod +x ./kind && sudo mv ./kind /usr/local/bin/kind

# Helm
curl https://get.helm.sh/helm-v3.12.0-linux-amd64.tar.gz | tar xz
sudo mv linux-amd64/helm /usr/local/bin/
```

## Repository Structure

The codebase follows a domain-driven architecture with comprehensive documentation:

### Core Application (`app/`)
- **📖 [`app/ARCHITECTURE.md`](../app/ARCHITECTURE.md)** - High-level system architecture
- **📖 [`app/CLAUDE.md`](../app/CLAUDE.md)** - AI development context and patterns

### Domain Layer (`app/domain/`)
- **📖 [`app/domain/ARCHITECTURE.md`](../app/domain/ARCHITECTURE.md)** - Domain-driven design patterns
- **Metric Collection**: Core metrics ingestion and processing
- **Shipper Services**: Metric data upload orchestration  
- **Webhook Server**: Kubernetes admission control
- **Diagnostic System**: Health monitoring and validation
- **Health Checks**: HTTP endpoints for monitoring

### Storage Layer (`app/storage/`)
- **📖 [`app/storage/ARCHITECTURE.md`](../app/storage/ARCHITECTURE.md)** - Storage architecture and patterns
- **Repository Pattern**: Generic CRUD interfaces with type safety
- **SQLite Backend**: Lightweight persistent storage
- **Disk Storage**: High-performance compressed file storage

### Utility Libraries (`app/utils/`)
- **Parallel Processing**: Controlled concurrency with worker pools
- **Distributed Locking**: File-based coordination across processes
- **Cloud Detection**: Multi-provider environment discovery
- **Telemetry**: Operational data transmission

## Development Workflow

### 1. Local Development Setup

```bash
# Clone and setup
git clone https://github.com/cloudzero/cloudzero-agent
cd cloudzero-agent

# Install dependencies
go mod download
go mod tidy

# Run tests
make test

# Start development environment
make dev-environment
```

### 2. Code Organization Patterns

#### Repository Pattern Usage
```go
// Define repository interface
type MetricRepository interface {
    types.Storage[types.Metric, uuid.UUID]
    FindByTimeRange(ctx context.Context, start, end time.Time) ([]types.Metric, error)
}

// Implement with storage backend
type metricRepo struct {
    storage.BaseRepoImpl
}

// Use in domain services
type MetricCollector struct {
    repo MetricRepository
    bus  types.Bus
}
```

#### Domain Service Pattern
```go
// Domain service with dependency injection
func NewMetricCollector(repo MetricRepository, bus types.Bus, config *config.Settings) *MetricCollector {
    return &MetricCollector{
        repo:   repo,
        bus:    bus,
        config: config,
    }
}

// Register health checks
func (mc *MetricCollector) RegisterHealthChecks() {
    healthz.Register("metric-collector", func() error {
        return mc.healthCheck()
    })
}
```

#### Configuration Management
```go
// Environment-based configuration
type Settings struct {
    Database DatabaseConfig `yaml:"database"`
    Storage  StorageConfig  `yaml:"storage"`
    
    // Auto-detected fields
    CloudProvider string `yaml:"-"`
    ClusterName   string `yaml:"-"`
}

// Configuration loading with validation
func LoadSettings() (*Settings, error) {
    settings := &Settings{}
    
    // Load from file, env vars, auto-detection
    if err := loadFromSources(settings); err != nil {
        return nil, err
    }
    
    return settings.Validate()
}
```

### 3. Testing Patterns

#### Unit Tests with Mocks
```go
func TestMetricCollector_ProcessMetrics(t *testing.T) {
    // Arrange
    mockRepo := mocks.NewMetricRepository(t)
    mockBus := mocks.NewBus(t)
    collector := NewMetricCollector(mockRepo, mockBus, testConfig)
    
    metrics := []types.Metric{...}
    mockRepo.On("Store", mock.Anything, metrics).Return(nil)
    
    // Act
    err := collector.ProcessMetrics(context.Background(), metrics)
    
    // Assert
    assert.NoError(t, err)
    mockRepo.AssertExpectations(t)
}
```

#### Integration Tests
```go
func TestMetricCollector_Integration(t *testing.T) {
    // Setup real storage backend
    db, err := sqlite.NewSQLiteDriver(sqlite.InMemoryDSN)
    require.NoError(t, err)
    
    repo := storage.NewMetricRepository(db)
    collector := NewMetricCollector(repo, realBus, testConfig)
    
    // Test complete workflow
    metrics := generateTestMetrics(100)
    err = collector.ProcessMetrics(context.Background(), metrics)
    require.NoError(t, err)
    
    // Verify persistence
    stored, err := repo.FindByTimeRange(ctx, start, end)
    assert.Len(t, stored, 100)
}
```

### 4. Performance Considerations

#### Memory Management
- Use streaming processing for large datasets
- Implement bounded queues for metric ingestion
- Configure garbage collection for high-throughput scenarios

#### Concurrency Patterns
```go
// Use utils/parallel for controlled concurrency
manager := parallel.New(-2) // 2x CPU cores
waiter := parallel.NewWaiter()

for _, file := range files {
    manager.Run(func() error {
        return processFile(file)
    }, waiter)
}

waiter.Wait()
for err := range waiter.Err() {
    log.Printf("Processing error: %v", err)
}
```

#### Storage Optimization
- Use batch operations for bulk inserts
- Configure appropriate compression levels
- Implement connection pooling for database access

## Operational Integration

### Health Checks
```go
// Register component health checks during initialization
func (s *Service) Initialize() error {
    healthz.Register("service-name", func() error {
        return s.healthCheck()
    })
    
    diagnostic.RegisterProvider(&ServiceDiagnostic{service: s})
    return nil
}
```

### Monitoring and Observability
```go
// Structured logging with context
log.WithContext(ctx).
    WithField("operation", "metric_processing").
    WithField("count", len(metrics)).
    Info("Processing metrics batch")

// Metrics collection
processingDuration.WithLabelValues("success").Observe(duration.Seconds())
```

### Configuration Management
```yaml
# config.yaml
database:
  type: sqlite
  dsn: /var/lib/agent/data.sqlite
  
storage:
  type: disk
  path: /var/lib/agent/metrics
  compressionLevel: 6
  maxRecords: 10000
  
cloudzero:
  apiKey: ${CLOUDZERO_API_KEY}
  host: https://api.cloudzero.com
```

## Deployment and Operations

### Local Testing
```bash
# Start development cluster
kind create cluster --config tests/kind/cluster-config.yaml

# Deploy agent for testing
helm install agent ./helm --values dev-values.yaml

# View logs
kubectl logs -f daemonset/agent-collector
kubectl logs -f deployment/agent-webhook
```

### Production Deployment
- Use Helm charts with appropriate resource limits
- Configure persistent storage for metrics data
- Set up monitoring and alerting for agent health
- Implement proper RBAC and security policies

### Troubleshooting
- Check agent health endpoints: `curl http://agent:8080/healthz`
- Review diagnostic information: `curl http://agent:8080/diagnostic`
- Monitor resource usage: `kubectl top pods -n cloudzero-system`
- Validate configuration: Run validator tool before deployment

## Contributing Guidelines

### Code Style
- Follow standard Go conventions and formatting
- Use `golangci-lint` for code quality validation
- Maintain test coverage above 80%
- Document all public APIs with comprehensive examples

### Pull Request Process
1. **Feature Branch**: Create feature branch from `develop`
2. **Implementation**: Follow established patterns and architecture
3. **Testing**: Add unit and integration tests
4. **Documentation**: Update relevant documentation
5. **Review**: Submit PR with comprehensive description
6. **Validation**: Ensure CI/CD passes all checks

### Documentation Requirements
- Update architecture documents for significant changes
- Add inline code documentation following established patterns
- Include usage examples for new APIs
- Update configuration documentation for new settings

## Advanced Topics

### Custom Storage Backends
Implement the `types.Storage` interface to add new storage backends:

```go
type CustomStorage struct {
    // Implementation fields
}

func (cs *CustomStorage) Create(ctx context.Context, entity *Model) error {
    // Custom storage logic
}

// Implement all interface methods...
```

### Custom Diagnostic Providers
Add new health checks by implementing the diagnostic interface:

```go
type CustomDiagnostic struct {
    service ExternalService
}

func (cd *CustomDiagnostic) Check(ctx context.Context, client *http.Client, accessor status.Accessor) error {
    // Perform health check
    // Update accessor with results
    return nil
}
```

### Performance Tuning
- Configure worker pool sizes based on deployment resources
- Tune compression levels for storage vs CPU trade-offs
- Optimize batch sizes for network and storage characteristics
- Implement custom metrics for application-specific monitoring

## Resources

### Internal Documentation
- [Architecture Overview](../app/ARCHITECTURE.md)
- [Domain Layer Design](../app/domain/ARCHITECTURE.md)
- [Storage Architecture](../app/storage/ARCHITECTURE.md)
- [AI Development Context](../app/CLAUDE.md)

### External Resources
- [Go Best Practices](https://golang.org/doc/effective_go.html)
- [Kubernetes Development](https://kubernetes.io/docs/concepts/)
- [GORM Documentation](https://gorm.io/docs/)
- [Prometheus Integration](https://prometheus.io/docs/instrumenting/writing_exporters/)

For questions or support, please refer to the internal documentation first, then reach out to the development team through established communication channels.