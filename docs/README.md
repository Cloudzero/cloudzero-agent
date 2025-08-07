# CloudZero Agent Documentation

Welcome to the comprehensive documentation for the CloudZero Kubernetes Agent. This documentation provides everything you need to understand, deploy, operate, and develop with the CloudZero Agent.

## 📋 Documentation Overview

### For Operators and DevOps Teams
- **[Operations Guide](OPERATIONS_GUIDE.md)** - Complete deployment, monitoring, and maintenance guide
- **[API Reference](API_REFERENCE.md)** - Comprehensive API documentation for integration
- **[Troubleshooting Guide](testing/webhook.md)** - Common issues and solutions

### For Developers  
- **[Developer Guide](DEVELOPER_GUIDE.md)** - Development setup, patterns, and contribution guidelines
- **[Architecture Documentation](#architecture-documentation)** - System design and component interactions
- **[Testing Documentation](testing/)** - Testing strategies and validation procedures

### For System Architects
- **[High-Level Architecture](#system-architecture)** - Overall system design and integration patterns
- **[Component Architecture](#component-architecture)** - Detailed component design and interactions
- **[Storage Architecture](#storage-architecture)** - Data persistence and storage patterns

## 🏗️ System Architecture

The CloudZero Agent is a sophisticated Kubernetes-native application designed for comprehensive cost monitoring and resource optimization.

```
┌─────────────────────────────────────────────────────────────────┐
│                    CloudZero Agent System                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────┐   ┌─────────────┐   ┌─────────────────────┐    │
│  │   Metric    │   │  Kubernetes │   │    Diagnostic &     │    │
│  │ Collection  │◄──┤  Admission  │◄──┤  Health Monitoring  │    │
│  │  Pipeline   │   │  Control    │   │      System         │    │
│  └─────────────┘   └─────────────┘   └─────────────────────┘    │
│         │                   │                     │             │
│         ▼                   ▼                     ▼             │
│  ┌─────────────┐   ┌─────────────┐   ┌─────────────────────┐    │
│  │   Storage   │   │   Config    │   │     Telemetry &     │    │
│  │    Layer    │   │ Management  │   │     Observability   │    │
│  └─────────────┘   └─────────────┘   └─────────────────────┘    │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## 📚 Architecture Documentation

### Main Application Architecture
- **[`../app/ARCHITECTURE.md`](../app/ARCHITECTURE.md)** - High-level system architecture, component relationships, and operational characteristics

### Component-Specific Architecture  
- **[`../app/domain/ARCHITECTURE.md`](../app/domain/ARCHITECTURE.md)** - Domain-driven design patterns, service coordination, and business logic architecture
- **[`../app/storage/ARCHITECTURE.md`](../app/storage/ARCHITECTURE.md)** - Repository patterns, storage backends, and data persistence strategies

### AI Development Context
- **[`../app/CLAUDE.md`](../app/CLAUDE.md)** - AI-specific development context, patterns, and guidelines *(Note: This file may be gitignored)*

## 🚀 Quick Start Guides

### For Operators
1. **Deploy the Agent**: Follow the [Operations Guide](OPERATIONS_GUIDE.md#installation-and-configuration)
2. **Configure Monitoring**: Set up health checks and metrics collection
3. **Validate Deployment**: Run diagnostic checks and verify functionality

### For Developers
1. **Setup Environment**: Follow the [Developer Guide](DEVELOPER_GUIDE.md#development-workflow) 
2. **Understand Architecture**: Read the architecture documentation
3. **Run Tests**: Execute the test suite and validate changes
4. **Contribute**: Follow the contribution guidelines and code patterns

### For Integrators
1. **API Integration**: Review the [API Reference](API_REFERENCE.md)
2. **Health Monitoring**: Implement health check integration
3. **Metrics Collection**: Set up Prometheus monitoring
4. **Webhook Integration**: Configure Kubernetes admission control

## 📖 Key Features

### Metric Collection and Processing
- **Prometheus Integration**: Remote write protocol support (v1 and v2)
- **High-Performance Storage**: Compressed disk storage with Brotli compression
- **Parallel Processing**: Controlled concurrency with semaphore-based worker pools
- **Intelligent Filtering**: Configurable metric filtering and transformation

### Kubernetes Integration
- **Admission Control**: Validating and mutating webhook support
- **Resource Monitoring**: Comprehensive Kubernetes resource tracking
- **RBAC Integration**: Secure, least-privilege access patterns
- **Multi-Cluster Support**: Distributed deployment capabilities

### Operational Excellence
- **Health Monitoring**: Comprehensive diagnostic and health check system
- **Observability**: Structured logging, metrics, and distributed tracing
- **Configuration Management**: Environment-based configuration with auto-detection
- **Security**: TLS encryption, secure credential management, audit logging

### Storage and Performance
- **Multiple Backends**: SQLite for metadata, disk storage for metrics
- **Compression**: Up to 90% storage savings with Brotli compression
- **Transaction Support**: ACID compliance with proper isolation levels
- **Scalability**: Horizontal and vertical scaling capabilities

## 🔧 Configuration Examples

### Production Deployment
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
    size: 20Gi
    storageClass: "fast-ssd"

resources:
  collector:
    requests: { cpu: 200m, memory: 256Mi }
    limits: { cpu: 500m, memory: 512Mi }
  webhook:
    requests: { cpu: 100m, memory: 128Mi }
    limits: { cpu: 200m, memory: 256Mi }

monitoring:
  enabled: true
  serviceMonitor: { enabled: true }
```

### Development Environment
```yaml
# dev-values.yaml  
global:
  cloudAccountId: "dev-account"
  clusterName: "dev-cluster"

storage:
  persistentVolume: { enabled: false }
  
resources:
  collector:
    requests: { cpu: 50m, memory: 128Mi }
  webhook:
    requests: { cpu: 25m, memory: 64Mi }

logging:
  level: debug
  format: json
```

## 🔍 Monitoring and Observability

### Health Check Endpoints
```bash
# Basic health check
curl http://localhost:8080/healthz

# Comprehensive diagnostics  
curl http://localhost:8080/diagnostic

# Prometheus metrics
curl http://localhost:8080/metrics
```

### Key Metrics to Monitor
```prometheus
# Metric collection rates
rate(cloudzero_metrics_collected_total[5m])
rate(cloudzero_metrics_processed_total[5m])

# Storage utilization  
cloudzero_storage_disk_usage_bytes
cloudzero_storage_files_pending

# Error rates
rate(cloudzero_errors_total[5m])

# Performance metrics
cloudzero_webhook_duration_seconds
histogram_quantile(0.95, rate(cloudzero_processing_duration_seconds_bucket[5m]))
```

## 🛠️ Development Patterns

### Repository Pattern Example
```go
// Domain service with dependency injection
type MetricCollector struct {
    repo   types.Storage[types.Metric, uuid.UUID]
    bus    types.Bus
    config *config.Settings
}

// Type-safe repository operations
func (mc *MetricCollector) StoreMetrics(ctx context.Context, metrics []types.Metric) error {
    return mc.repo.Tx(ctx, func(txCtx context.Context) error {
        for _, metric := range metrics {
            if err := mc.repo.Create(txCtx, &metric); err != nil {
                return fmt.Errorf("failed to store metric: %w", err)
            }
        }
        return nil
    })
}
```

### Health Check Registration
```go
// Register component health checks
func (s *Service) Initialize() error {
    healthz.Register("service-name", func() error {
        return s.healthCheck()
    })
    
    diagnostic.RegisterProvider(&ServiceDiagnostic{service: s})
    return nil
}
```

## 📝 Testing Strategy

### Unit Tests
- **Coverage**: >80% code coverage requirement
- **Mocking**: Use generated mocks for external dependencies
- **Patterns**: Test business logic with clear arrange/act/assert structure

### Integration Tests
- **Database**: Test repository implementations with real databases
- **HTTP**: Test API endpoints with real HTTP clients
- **Kubernetes**: Test webhook integration with test clusters

### End-to-End Tests
- **Workflows**: Complete metric ingestion through upload workflows
- **Performance**: Load testing under realistic conditions
- **Chaos**: Resilience testing with failure injection

## 🔒 Security Considerations

### Access Control
- **RBAC**: Kubernetes role-based access control
- **Service Accounts**: Dedicated service accounts with minimal permissions
- **Network Policies**: Restricted network access patterns

### Data Protection
- **Encryption**: TLS for all external communications
- **Secrets**: Kubernetes secrets for sensitive configuration
- **Audit**: Comprehensive audit logging for security events

### Compliance
- **Scanning**: Regular security vulnerability scanning
- **Policies**: Enforcement of security policies and best practices
- **Monitoring**: Security event monitoring and alerting

## 📞 Support and Contributing

### Getting Help
- **Documentation**: Start with the relevant guide above
- **Troubleshooting**: Check the [troubleshooting guide](testing/webhook.md)
- **Issues**: Create GitHub issues for bugs and feature requests

### Contributing
- **Code Style**: Follow established patterns and conventions
- **Testing**: Add comprehensive tests for all changes
- **Documentation**: Update documentation for significant changes
- **Review Process**: All changes require peer review

### Development Resources
- **Setup**: [Developer Guide - Development Workflow](DEVELOPER_GUIDE.md#development-workflow)
- **Patterns**: [Architecture Documentation](#architecture-documentation)
- **APIs**: [API Reference](API_REFERENCE.md)
- **Examples**: See `examples/` directory for usage examples

## 📄 Additional Resources

### Release Information
- **[Release Notes](releases/)** - Version history and changelog
- **[Release Process](releases/RELEASE_PROCESS.md)** - Release and deployment procedures

### Testing Documentation  
- **[Helm Testing](testing/helm-cheatsheet.md)** - Helm chart testing procedures
- **[Validator Testing](testing/validator.md)** - Configuration validation testing
- **[Webhook Testing](testing/webhook.md)** - Admission controller testing

### Deployment Assets
- **[Helm Charts](../helm/)** - Kubernetes deployment configurations
- **[Docker Images](../docker/)** - Container build configurations
- **[CI/CD](../scripts/)** - Continuous integration scripts

This documentation is actively maintained and updated with each release. For the most current information, always refer to the documentation in the main branch of the repository.