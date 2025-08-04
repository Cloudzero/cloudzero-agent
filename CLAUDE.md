# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common Development Commands

### Building and Testing

- `make -j build` - Build all binaries to `bin/` directory
- `make test` - Run unit tests with race detection and coverage
- `make test-integration` - Run integration tests (requires `CLOUDZERO_DEV_API_KEY`)
- `make test-smoke` - Run smoke tests against live API
- `make -j format` - Format Go code with gofumpt and run `go mod tidy`
- `make -j lint` - Run golangci-lint on all packages
- `make -j analyze` - Run staticcheck static analysis

### Dependencies and Tools

- `make install-tools` - Install all development tools (Go tools, Node.js, golangci-lint)
- `make generate` - Regenerate protobuf files and other generated code
- `go mod download` - Download Go module dependencies

### Container Images

- `make package-build` - Build Docker image locally
- `make package` - Build and push Docker image to registry

### Helm Development

- `make helm-install` - Install Helm chart with default values
- `make helm-template` - Generate Helm templates for inspection
- `make helm-test` - Run Helm schema validation tests
- `make helm-lint` - Lint Helm chart

## Architecture Overview

This is a CloudZero Kubernetes cost intelligence agent with multiple specialized applications:

### Core Applications (`app/functions/`)

- **Collector** - Prometheus remote write API server that stores metrics in compressed parquet files
- **Shipper** - Monitors disk storage and uploads metric files to CloudZero API via S3
- **Webhook** - Kubernetes admission controller that captures resource metadata for cost allocation
- **Agent Validator** - CLI tool for deployment validation and diagnostics

### Domain Logic (`app/domain/`)

- **Metric Collection** (`metric_collector.go`) - Core metrics processing and classification
- **Shipper** (`shipper/`) - File upload orchestration and S3 interaction
- **Webhook** (`webhook/`) - Kubernetes resource handling and admission control
- **Diagnostics** (`diagnostic/`) - System health checks and validation

### Storage Layer (`app/storage/`)

- **Disk** (`disk/`) - Parquet file storage with compression
- **SQLite** (`sqlite/`) - Embedded database option
- **Repository Pattern** (`repo/`) - Storage abstractions

### Configuration (`app/config/`)

- **Gator** (`gator/`) - Main collector/shipper configuration
- **Webhook** (`webhook/`) - Admission controller specific config
- **Validator** (`validator/`) - Validation tool configuration

## Key Data Flow

```
Prometheus → Collector → Disk Storage → Shipper → CloudZero API
Kubernetes → Webhook → Resource Store → CloudZero API
```

## Important File Patterns

### Main Entry Points

- `app/functions/*/main.go` - Application entry points
- `cmd/*/main.go` - CLI command entry points (if any)

### Core Types

- `app/types/metric.go` - Prometheus metric structures
- `app/types/resource.go` - Kubernetes resource metadata
- `app/types/storage.go` - Storage interface definitions

### Configuration

- `helm/values.yaml` - Default Helm chart configuration
- `app/config/gator/settings.go` - Main application settings structure

## Cursor Rules Usage

### How to Use Cursor Rules

**Claude should reference and follow the cursor rules as appropriate for each task:**

1. **Always consult relevant rules** before starting development work
2. **Follow the mandatory patterns** especially from core-development.mdc (Make commands, file formatting, etc.)
3. **Apply context-appropriate rules** - don't apply Helm rules to Go code, but do apply core rules to everything

### When to Use Each Rule Set

- **Before ANY development task** → @.cursor/rules/core-development.mdc (mandatory patterns)
- **Go code changes** → @.cursor/rules/go-development.mdc (testing patterns, mock usage)
- **Helm chart work** → @.cursor/rules/helm-development.mdc (schema validation, template patterns)
- **Architecture questions** → @.cursor/rules/code-patterns.mdc (CloudZero-specific patterns)
- **Project navigation** → @.cursor/rules/project-overview.mdc (understand structure)
- **API or commit standards** → @.cursor/rules/project-reference.mdc (API formats, commit messages)
- **Workflow questions** → @.cursor/rules/collaborative-development.mdc (rules maintenance)
- **Deploying** → @.cursor/rules/personal-testing-deployment.mdc (deployment to personal cluster)

### Critical Rule Compliance

**Claude must always:**

- Use Make commands instead of direct Go commands (from core-development)
- Update cursor rules when learning new patterns (from collaborative-development)
- Follow established testing patterns (from go-development)
- Use proper file formatting and structure (from code-patterns)

## Claude Code and Cursor Integration

**For comprehensive development guidelines, see the cursor rules in `.cursor/rules/`:**

- **📖 [core-development.mdc](.cursor/rules/core-development.mdc)** - Essential commands, task system, session context management
- **📖 [go-development.mdc](.cursor/rules/go-development.mdc)** - Go testing patterns and development practices
- **📖 [helm-development.mdc](.cursor/rules/helm-development.mdc)** - Helm chart development and testing
- **📖 [code-patterns.mdc](.cursor/rules/code-patterns.mdc)** - Project-specific patterns and conventions
- **📖 [project-overview.mdc](.cursor/rules/project-overview.mdc)** - High-level project structure
- **📖 [personal-testing-deployment.mdc](.cursor/rules/personal-testing-deployment.mdc)** - Cluster deployment configurations

**Key highlights from cursor rules:**

- **Task-based workflow**: Use modular tasks (`task-*.mdc`) for development operations
- **Session context**: Automatic creation and maintenance of `scratch/ai-ctx/*.md` files
- **Make commands**: Always use Make targets, never direct Go commands
- **Testing strategy**: Intelligent test selection based on what changed
- **Deployment**: Multi-cluster testing (AWS EKS, Google GKE, Azure AKS)

## Development Environment

### Required Tools

- Go 1.24+
- Docker/Rancher Desktop
- Protocol Buffers compiler (`protoc`)
- Node.js

### Environment Variables

- `CLOUDZERO_DEV_API_KEY` - Required for integration/smoke tests and Helm deployment
- `CLOUDZERO_HOST` - API host (defaults to dev-api.cloudzero.com)
- `CLOUD_ACCOUNT_ID`, `CSP_REGION`, `CLUSTER_NAME` - Cloud provider settings for tests

### Local Configuration

- Create `local-config.mk` to override Makefile variables
- Use `make secrets-act` to generate GitHub Actions secrets for local testing

## Testing Strategy

### Test Types

- **Unit Tests** - Fast tests with mocks, run with `make test`
- **Integration Tests** - Test against live APIs, require credentials
- **Smoke Tests** - End-to-end validation of deployed components
- **Helm Tests** - Schema validation and template generation tests

### Mock Generation

- Uses `go.uber.org/mock` for interface mocking
- Mock files in `app/types/mocks/`
- Regenerate with `make generate`

## Code Conventions

### Project Structure

- Domain-driven design with clear separation between applications
- Interfaces defined in `app/types/` for dependency injection
- Configuration structs use `cleanenv` for environment variable binding
- Storage layer uses repository pattern for testability

### Go Standards

- All packages use consistent error handling with `github.com/pkg/errors`
- Logging via `github.com/rs/zerolog` for structured output
- HTTP clients use `github.com/hashicorp/go-retryablehttp` for resilience
- Time handling uses injectable time providers for testing

### Container Strategy

- Scratch-based container images (~8MB) for production
- Debug variants available with `busybox` for troubleshooting
- Multi-architecture builds (linux/amd64, linux/arm64)
- All binaries compiled with static linking (`netgo osusergo` tags)
