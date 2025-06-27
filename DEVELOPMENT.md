# CloudZero Agent Development Guide

## Repository Overview

This repository contains the complete CloudZero Agent ecosystem for Kubernetes integration with the CloudZero platform. It includes:

### Core Applications

- **CloudZero Insights Controller** - Webhook application that collects resource labels, annotations, and metadata
- **CloudZero Collector** - Prometheus-compatible metrics collection service
- **CloudZero Shipper** - File monitoring and S3 upload service
- **CloudZero Agent Validator** - Installation validation and lifecycle management
- **CloudZero Agent** - Metrics scraping service (supports federated mode for large clusters)

### Helm Chart

- **Helm Chart** (`helm/`) - Complete Kubernetes deployment configuration
- **Chart Testing** - Comprehensive validation and testing framework
- **Schema Validation** - JSON schema validation for configuration

### Utilities and Tools

- **Helmless Tool** - Configuration analysis and minimal override generation
- **Agent Inspector** - Debugging and inspection utilities
- **Scout** - Cloud provider metadata detection (AWS, Google Cloud)

## Development Setup

### Prerequisites

- [Go 1.24+](https://go.dev/doc/install)
- [Docker](https://docs.docker.com/engine/install/)
- [Helm 3.14+](https://helm.sh/docs/intro/install/)
- [Kubernetes cluster](https://kubernetes.io/docs/setup/) (local or remote)
- [Protocol Buffers compiler](https://protobuf.dev/downloads/)

### Quick Start

1. **Clone and Setup**
   ```bash
   git clone https://github.com/cloudzero/cloudzero-agent.git
   cd cloudzero-agent
   make install-tools
   ```

2. **Build Applications**
   ```bash
   make build
   ```

3. **Run Tests**
   ```bash
   make test
   make helm-test
   ```

4. **Build Docker Images**
   ```bash
   make package-build
   ```

## Development Workflow

### Code Organization

```
app/
├── functions/          # Standalone applications
│   ├── agent-validator/
│   ├── collector/
│   ├── shipper/
│   └── webhook/
├── domain/            # Core business logic
├── handlers/          # HTTP handlers
├── storage/           # Data persistence
└── types/            # Shared types and protobuf definitions

helm/                  # Helm chart
├── templates/         # Kubernetes manifests
├── values.yaml        # Default configuration
└── docs/             # Chart documentation

tests/                 # Test suites
├── helm/             # Helm chart tests
├── integration/      # Integration tests
└── smoke/           # Smoke tests
```

### Key Make Targets

```bash
# Development
make build                    # Build all binaries
make test                     # Run unit tests
make test-integration         # Run integration tests
make lint                     # Run linters
make format                   # Format code

# Docker
make package-build           # Build Docker images locally
make package                 # Build and push Docker images

# Helm
make helm-install           # Install chart locally
make helm-test              # Run helm validation tests
make helm-lint              # Lint helm chart
make helm-template          # Generate templates

# Changelog
make generate-changelog     # Generate changelog (TAG_VERSION=1.2.3)
```

### Environment Configuration

Create `local-config.mk` for local overrides:

```makefile
# API Configuration
CLOUDZERO_DEV_API_KEY=your-dev-api-key
CLOUDZERO_HOST=dev-api.cloudzero.com

# Cluster Configuration
CLUSTER_NAME=my-test-cluster
CLOUD_ACCOUNT_ID=123456789
CSP_REGION=us-east-1
```

## Release Process

### Overview

The CloudZero Agent follows a structured release process with automated chart mirroring to the [cloudzero-charts](https://github.com/cloudzero/cloudzero-charts) repository.

### Release Workflow

1. **Development** - Work on `develop` branch
2. **Chart Mirroring** - Automatic sync to `cloudzero-charts` on push to `develop`
3. **Release Preparation** - Manual workflow creates release branch and tags
4. **Release Notes** - Must exist in `helm/docs/releases/{version}.md`

### Creating a Release

1. **Prepare Release Notes**
   ```bash
   # Create release notes file
   touch helm/docs/releases/1.2.3.md
   # Add release content following existing format
   ```

2. **Generate Changelog** (Optional)
   ```bash
   TAG_VERSION=1.2.3 make generate-changelog
   ```

3. **Trigger Release**
   - Go to GitHub Actions
   - Run "Manual Prepare Release" workflow
   - Input version (e.g., `1.2.3`)

4. **Release Process**
   - Updates image version in Helm chart
   - Merges `develop` into `main`
   - Creates Git tag
   - Creates GitHub release (draft)

### Chart Mirroring

The `mirror-chart.yml` workflow automatically:
- Syncs `helm/` directory to `cloudzero-charts/charts/cloudzero-agent/`
- Preserves commit history and authorship
- Runs on every push to `develop` branch

## Testing

### Unit Tests
```bash
make test
```

### Integration Tests
```bash
export CLOUDZERO_DEV_API_KEY=your-key
make test-integration
```

### Helm Chart Tests
```bash
make helm-test                    # All helm tests
make helm-test-schema            # Schema validation
make helm-test-subchart          # Subchart tests
```

### Smoke Tests
```bash
make test-smoke
```

## Debugging

### Local Development

1. **Use Debug Images**
   ```bash
   make package-build-debug
   ```

2. **Deploy Debug Container**
   ```bash
   kubectl run debug --image=busybox:stable-uclibc --rm -it -- sh
   ```

3. **Monitor Application Logs**
   ```bash
   kubectl logs -f deployment/cloudzero-agent
   ```

### Common Issues

- **Certificate Issues** - Check `docs/cert-trouble-shooting.md`
- **Validation Failures** - See `docs/deploy-validation.md`
- **Storage Issues** - Review `docs/storage/` guides

## Configuration

### Helm Values

Key configuration areas:
- **API Authentication** - `global.apiKey`, `global.existingSecretName`
- **Cluster Identification** - `clusterName`, `cloudAccountId`, `region`
- **Component Control** - `components.*.enabled`
- **Resource Limits** - `components.*.resources`

### Environment Variables

Applications support configuration via:
- Helm chart values
- Environment variables
- ConfigMaps
- Secrets

## Contributing

1. **Follow existing patterns** - Review similar components
2. **Add tests** - Unit tests for new functionality
3. **Update documentation** - Keep docs current
4. **Validate changes** - Run full test suite

### Code Style

- Use `gofumpt` for formatting
- Follow Go best practices
- Add godoc comments for public APIs
- Use structured logging

### Commit Messages

Follow conventional commit format:
```
type(scope): description

- feat: new feature
- fix: bug fix
- docs: documentation
- test: testing
- refactor: code refactoring
```

## Troubleshooting

### Build Issues
- Ensure Go version matches `go.mod`
- Run `make install-tools` to install dependencies
- Check Docker daemon is running

### Test Failures
- Verify API key is set for integration tests
- Ensure Kubernetes cluster is accessible
- Check resource limits and permissions

### Deployment Issues
- Validate Helm chart with `make helm-lint`
- Check cluster permissions
- Review application logs

## Additional Resources

- [Configuration Guide](../CONFIGURATION.md) - Detailed configuration options
- [Usage Guide](../USAGE.md) - Usage examples and patterns
- [Contributing Guide](../CONTRIBUTING.md) - Contribution guidelines
- [Release Process](releases/RELEASE_PROCESS.md) - Detailed release procedures