# Scripts

Utility scripts for CloudZero Agent development, testing, and operations.

## Available Scripts

### Development & CI

- `ci-checks.sh` - CI validation (Go version consistency, dependencies, build config)
- `merge-json-schema.jq` - JSON schema merging for Helm chart validation
- `monitor-admission-controller.sh` - Kubernetes admission controller monitoring

### Customer Support

- `anaximander.sh` - Diagnostic information gatherer for customer support (gathers logs, resource descriptions, and configuration from a CloudZero Agent installation)

## Usage

### Development Scripts

Most scripts are invoked via Makefile targets:

```bash
make ci-checks              # Run CI validation
make helm-schema-merge      # JSON schema operations
```

### Customer Support Scripts

The `anaximander.sh` script is provided to customers for gathering diagnostic information:

```bash
# Basic usage
./scripts/anaximander.sh <kube-context> <namespace>

# Example
./scripts/anaximander.sh production-cluster cloudzero-agent

# Specify output directory
./scripts/anaximander.sh prod-cluster cloudzero /tmp/diagnostics
```

This creates a timestamped directory with comprehensive diagnostic information:

- Helm release details
- Kubernetes resource listings and descriptions
- Secret size information (for troubleshooting large secrets)
- Container logs from all pods (current and previous)
- Job logs
- Events
- ConfigMaps
- Network policies
- Pod resource usage (kubectl top)
- Service mesh detection (Istio, Linkerd, Consul)
- Scrape configuration
- cAdvisor metrics (from one node for configuration verification)

The script automatically creates a `.tar.gz` archive suitable for sharing with support.
