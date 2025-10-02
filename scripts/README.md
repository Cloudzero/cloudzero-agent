# Development Scripts

Utility scripts for CloudZero Agent development, testing, and operations.

## Scripts

- `ci-checks.sh` - CI validation (Go version consistency, dependencies, build config)
- `merge-json-schema.jq` - JSON schema merging for Helm chart validation
- `monitor-admission-controller.sh` - Kubernetes admission controller monitoring

## Usage

Scripts are typically invoked via Makefile targets:

```bash
make ci-checks              # Run CI validation
make helm-schema-merge      # JSON schema operations
```
