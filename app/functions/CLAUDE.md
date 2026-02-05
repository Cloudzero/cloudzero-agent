# CloudZero Agent Functions

This directory contains the individual CLI applications and services that make up the CloudZero Agent system. Each function is a standalone Go application with specific responsibilities in the cost allocation and monitoring workflow.

## Function Overview

**Core Agent Functions:**

- `collector/` - Prometheus metrics collection and processing
- `shipper/` - S3 upload orchestration and data delivery
- `webhook/` - Kubernetes admission controller for resource validation
- `agent-validator/` - Deployment validation and health checking

**Utility Functions:**

- `helmless/` - Kubernetes deployment without Helm
- `agent-inspector/` - Agent introspection and debugging
- `certifik8s/` - Certificate management for Kubernetes
- `regurgitator/` - Multi-format metric transcoding (CSV, Parquet, JSON, remote_write)
- `scout/` - Resource discovery and scanning
- `cluster-config/` - Cluster configuration management

## Architecture Pattern

Each function follows the hexagonal architecture pattern:

- **Main package** - CLI entry point with dependency injection
- **Business logic** - Uses interfaces from `../domain/` and `../types/`
- **External adapters** - Database, HTTP clients, Kubernetes APIs

## Build Integration

Functions are built via Make targets:

```bash
make build                    # Build all functions
make test GO_TEST_TARGET=./app/functions/collector/...  # Test specific function
```

Built binaries are placed in `bin/cloudzero-{function-name}`.
