# CloudZero Agent Architecture

## Overview

The CloudZero Agent is a sophisticated Kubernetes-native application designed to collect, process, and transmit infrastructure metrics and resource information to the CloudZero cost management platform. The agent follows a modular, domain-driven architecture with clear separation of concerns and robust operational characteristics.

## High-Level Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Kubernetes    │    │   Prometheus    │    │  CloudZero API  │
│    Cluster      │    │    Metrics      │    │    Platform     │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          │                      │                      │
┌─────────▼─────────────────────▼───────────────────────▼──────────┐
│                    CloudZero Agent                              │
├──────────────────────────────────────────────────────────────────┤
│                       Components                                │
├─────────────────┬─────────────────┬─────────────────────────────┤
│  Metric         │    Webhook      │       Diagnostic &          │
│  Collector      │    Server       │       Health System         │
│  & Shipper      │                 │                             │
└─────────────────┴─────────────────┴─────────────────────────────┘
```

## Core Components

### 1. Metric Collection & Processing Pipeline
- **Collector**: Ingests metrics from Prometheus remote write endpoints
- **Storage**: High-performance disk-based storage with compression
- **Shipper**: Processes and uploads metric data to CloudZero APIs
- **Filter**: Intelligent metric filtering and transformation

### 2. Kubernetes Admission Control
- **Webhook Server**: Validates and mutates Kubernetes resource definitions
- **Resource Handlers**: Specialized handlers for different K8s resource types
- **Backfiller**: Ensures historical resource coverage

### 3. Operational Infrastructure
- **Diagnostic System**: Comprehensive health monitoring and validation
- **Health Checks**: HTTP endpoints for monitoring system integration  
- **Configuration Management**: Multi-environment configuration handling
- **Cloud Discovery**: Multi-cloud environment detection and metadata

## Directory Structure & Responsibilities

### `/app` - Application Root
The main application directory containing all functional components.

### `/app/config` - Configuration Management
- **gator/**: Main agent configuration with cloud environment detection
- **validator/**: Validation-specific configuration and settings
- **webhook/**: Admission controller webhook configuration

### `/app/domain` - Business Logic Layer
Core business logic organized by functional domains:

- **metric_collector.go**: Main metrics ingestion service
- **shipper/**: Metric file processing and upload orchestration
- **webhook/**: Kubernetes admission control logic
- **diagnostic/**: System health validation framework
- **healthz/**: HTTP health check endpoints
- **monitor/**: File and certificate monitoring services
- **k8s/**: Kubernetes API integration utilities

### `/app/storage` - Data Persistence Layer
- **core/**: Repository pattern abstractions and GORM integration
- **disk/**: High-performance disk storage with compression
- **sqlite/**: SQLite database driver configuration
- **repo/**: Concrete repository implementations

### `/app/types` - Core Data Structures
- **metric.go**: Core metric data structures and serialization
- **storage.go**: Generic storage interfaces with type parameters
- **status/**: Operational status and reporting structures
- **errors.go**: Application-specific error definitions

### `/app/utils` - Utility Libraries
- **scout/**: Multi-cloud environment detection
- **lock/**: Distributed file-based locking
- **parallel/**: Controlled concurrency execution
- **telemetry/**: Operational data transmission

### `/app/functions` - Executable Components
Entry points for different deployment modes:
- **collector/**: Metric collection service main
- **shipper/**: Metric shipping service main  
- **webhook/**: Admission controller webhook main
- **agent-validator/**: Configuration validation tools

### `/app/handlers` - HTTP Request Handlers
- **remote_write.go**: Prometheus remote write endpoint
- **webhook.go**: Kubernetes admission webhook endpoint
- **profiling.go**: Runtime profiling endpoints

## Key Architecture Patterns

### 1. Domain-Driven Design
The application is organized around business domains rather than technical layers, promoting clear ownership and maintainability.

### 2. Repository Pattern
All data access is abstracted through repository interfaces, enabling testability and storage backend flexibility.

### 3. Dependency Injection
Components receive their dependencies through constructor injection, promoting loose coupling and testability.

### 4. Interface Segregation
Small, focused interfaces ensure components only depend on what they need, following the Interface Segregation Principle.

### 5. Fail-Safe Operations
All operations are designed to fail gracefully without impacting core functionality, particularly diagnostic and telemetry systems.

## Data Flow Architecture

### Metric Collection Flow
1. **Ingestion**: Prometheus remote write → Collector HTTP handler
2. **Processing**: Validation, filtering, and metadata enrichment
3. **Storage**: Compressed disk storage with atomic file operations
4. **Shipping**: Background upload to CloudZero APIs with retry logic

### Kubernetes Admission Flow
1. **Webhook**: K8s API server → Admission webhook endpoint
2. **Validation**: Resource-specific validation and policy enforcement
3. **Mutation**: Resource enhancement with CloudZero labels/annotations
4. **Response**: Admission decision back to K8s API server

### Diagnostic Flow
1. **Collection**: Diagnostic providers → Status accessor
2. **Aggregation**: Unified health status compilation
3. **Exposition**: HTTP health endpoints and telemetry transmission

## Operational Characteristics

### Scalability
- **Horizontal**: Supports multiple replicas with distributed coordination
- **Vertical**: Configurable resource limits and efficient resource usage
- **Storage**: Bounded disk usage with automatic cleanup

### Reliability
- **Fault Tolerance**: Graceful degradation and automatic recovery
- **Data Integrity**: Atomic operations and transactional consistency
- **Monitoring**: Comprehensive health checks and diagnostic reporting

### Security
- **Authentication**: Secure API key management and rotation
- **Authorization**: Least-privilege principle with RBAC integration
- **Encryption**: TLS/HTTPS for all external communications
- **Compliance**: Security scanning and vulnerability management

### Performance
- **Low Latency**: Optimized for real-time metric processing
- **High Throughput**: Parallel processing and efficient storage
- **Resource Efficiency**: Minimal CPU and memory footprint

## Configuration Management

The agent supports multiple configuration methods:
- **Environment Variables**: Runtime configuration and secrets
- **Config Files**: YAML-based configuration for complex settings  
- **Command Line**: Override and debugging options
- **Auto-Detection**: Automatic cloud environment and cluster discovery

## Deployment Models

### DaemonSet Mode
- **Node Coverage**: Runs on every node for complete cluster visibility
- **Resource Collection**: Node-level metrics and resource utilization
- **Network Policy**: Local network access for node-specific data

### Deployment Mode  
- **Centralized**: Single replica for webhook and diagnostic services
- **High Availability**: Multiple replicas with leader election
- **Load Balancing**: Distributed processing across replicas

## Integration Points

### External Dependencies
- **Kubernetes API**: Resource discovery and admission control
- **Prometheus**: Metrics ingestion via remote write protocol
- **CloudZero API**: Cost data transmission and configuration
- **Cloud Providers**: Metadata services for environment detection

### Internal Dependencies
- **Storage Backend**: SQLite for small data, disk files for metrics
- **Message Bus**: Internal event coordination and notifications
- **Certificate Management**: TLS certificate lifecycle management
- **Log Aggregation**: Structured logging with centralized collection

This architecture enables the CloudZero Agent to provide reliable, scalable, and secure cost monitoring capabilities across diverse Kubernetes environments while maintaining operational excellence and developer productivity.