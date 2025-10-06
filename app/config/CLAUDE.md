# Configuration Management

This directory contains configuration management code for all CloudZero Agent components. It provides shared configuration interfaces and component-specific configuration packages.

## Package Overview

**Core Configuration:**

- `config.go` - Shared configuration interfaces (Serializable)

**Component Configurations:**

- `gator/` - Configuration for Gator component (cluster discovery and settings)
- `validator/` - Configuration for Agent Validator component (deployment validation, diagnostics, Prometheus config)
- `webhook/` - Configuration for Webhook Server component (Kubernetes admission controller settings, certificates, filtering rules)

## Architecture

Uses the `cleanenv` library for environment variable binding with struct tags for JSON/YAML/ENV binding and validation. Each component has its own settings structure with validation rules and test coverage.

## Testing

```bash
make test GO_TEST_TARGET=./app/config/...
```

All configuration packages include comprehensive test coverage with test data in `testdata/` directories.
