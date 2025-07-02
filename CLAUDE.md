# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common Development Commands

### Building and Development
```bash
# Install development tools (required first time)
make install-tools

# Build all binaries
make build

# Run formatter, linter, and tests
make format lint test

# Generate protobuf definitions and other generated code
make generate

# Clean build artifacts
make clean
```

### Testing
```bash
# Run unit tests
make test

# Run integration tests (requires CLOUDZERO_DEV_API_KEY)
make test-integration

# Run smoke tests (requires CLOUDZERO_DEV_API_KEY)
make test-smoke
```

### Docker and Containerization
```bash
# Build Docker image locally
make package-build

# Build debug version locally
make package-build-debug

# Build and push to registry
make package
```

### Helm Chart Development
```bash
# Install Helm chart dependencies
make helm-install-deps

# Generate Helm templates
make helm-template

# Run Helm chart validation tests
make helm-test

# Install chart locally (requires CLOUDZERO_DEV_API_KEY)
make helm-install

# Uninstall chart
make helm-uninstall
```

### Environment Setup
Set up a `local-config.mk` file in the root directory to override default configuration:
```makefile
CLOUDZERO_DEV_API_KEY = your-api-key-here
GITHUB_TOKEN = your-github-token
CLOUD_ACCOUNT_ID = your-account-id
CLUSTER_NAME = your-cluster-name
```

## High-Level Architecture

### Core Components
The CloudZero Agent consists of four main applications:

1. **Collector** (`app/functions/collector/`): Prometheus-compatible metrics collection service that receives remote write requests, classifies metrics, and stores them as compressed JSON files.

2. **Shipper** (`app/functions/shipper/`): Monitors local storage for metric files and uploads them to CloudZero's S3 bucket using pre-signed URLs from the upload API.

3. **Webhook (Insights Controller)** (`app/functions/webhook/`): Kubernetes admission webhook that intercepts API operations to collect resource metadata (labels, annotations, relationships) and stores them in-memory.

4. **Agent Validator** (`app/functions/agent-validator/`): Lifecycle management tool that performs validation checks and notifies CloudZero of agent status changes.

### Domain Layer Organization
The domain layer (`app/domain/`) follows clean architecture principles:

- **Core Services**: MetricCollector, MetricShipper, WebhookController handle primary business logic
- **Supporting Services**: Monitor (file/secret watching), Pusher (remote write), Housekeeper (cleanup), Diagnostic (health checks)
- **Storage Abstraction**: DiskStore (file-based), ResourceStore (in-memory with GORM), with configurable compression and retention

### Data Flow Patterns
1. **Metrics Pipeline**: Prometheus → Collector → DiskStore → Shipper → CloudZero S3
2. **Resource Metadata**: K8s API → Webhook → ResourceStore → Pusher → CloudZero Remote Write
3. **Configuration**: Hot-reload capable with environment variables, YAML files, and auto-detection

## Key Files and Directories

### Application Structure
- `app/functions/`: Main application entry points (collector, shipper, webhook, validator)
- `app/domain/`: Core business logic and domain services
- `app/types/`: Data models and type definitions
- `app/storage/`: Storage abstraction layer
- `app/config/`: Configuration management
- `app/handlers/`: HTTP handlers for various endpoints

### Build and Deployment
- `Makefile`: Comprehensive build system with targets for development, testing, and deployment
- `docker/Dockerfile`: Multi-stage Docker build for production containers
- `helm/`: Kubernetes deployment charts with extensive configuration options
- `go.mod`: Go module definition with dependencies

### Testing Infrastructure
- `tests/integration/`: Integration tests requiring API connectivity
- `tests/smoke/`: End-to-end smoke tests
- `tests/helm/`: Helm chart validation tests
- `mock/`: Mock services for testing

## Development Workflow

### Code Generation
- Protobuf definitions are in `app/types/*/` directories
- Run `make generate` after modifying `.proto` files
- Helm values schema is auto-generated from `helm/values.schema.yaml`

### Testing Strategy
- Unit tests use Go's standard testing package with testify
- Integration tests require CloudZero API key in environment
- Helm tests validate chart templates and schema
- Mock objects are generated using go-mock

### Configuration Patterns
- Environment-specific configuration in `local-config.mk`
- Kubernetes configuration via Helm values
- Runtime configuration supports hot-reload for secrets
- Cloud provider auto-detection for AWS, GCP, Azure

### Error Handling
- Structured error types in `app/types/errors.go`
- Comprehensive retry logic in shipper and pusher components
- Graceful degradation when external services are unavailable
- Detailed logging with contextual information

## Common Issues and Solutions

### Build Issues
- If protobuf generation fails, ensure `protoc` and Go plugins are installed
- Missing development tools: run `make install-tools`
- Docker build issues: ensure Docker buildx is available

### Test Failures
- Integration/smoke tests require valid `CLOUDZERO_DEV_API_KEY`
- Helm tests may fail if Kubernetes schema files are not current
- Network connectivity required for external dependency tests

### Configuration Problems
- Check `local-config.mk` for correct environment variable overrides
- Verify Helm values match your cluster configuration
- Ensure API keys have appropriate permissions

This codebase follows Go best practices with comprehensive testing, clear domain boundaries, and production-ready deployment patterns. The architecture supports both development and production environments with appropriate configuration management and monitoring capabilities.