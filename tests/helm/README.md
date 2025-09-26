# Helm Chart Tests

This directory contains comprehensive tests for the CloudZero Agent Helm chart. These tests validate chart templates, values schemas, and deployment configurations without requiring a live Kubernetes cluster.

## Overview

The Helm tests are **fully integrated** with the main project's Makefile and provide multiple layers of validation:

- **[Schema validation](schema/)** - Tests Helm values against JSON schemas and template rendering
- **[Template generation](template/)** - Tests baseline template generation and change detection
- **[Subchart validation](subchart/)** - Tests chart dependencies and parent/child relationships
- **Unit tests** - Tests chart logic using helm-unittest plugin

## Test Structure

### [Schema Tests (`schema/`)](schema/)

Comprehensive schema and template validation tests. See **[schema/README.md](schema/README.MD)** for detailed documentation.

- **Template validation** - Tests Helm template rendering with various configurations
- **Kubeconform validation** - Tests rendered manifests against Kubernetes API schemas
- **Value validation** - Tests chart values against schema constraints

### [Template Tests (`template/`)](template/)

Template generation and change detection tests. See **[template/README.md](template/README.MD)** for detailed documentation.

- **Baseline generation** - Creates reference manifests from override configurations
- **Change detection** - Identifies modifications to generated Kubernetes resources
- **Regression prevention** - Ensures chart changes produce expected results

### [Subchart Tests (`subchart/`)](subchart/)

Subchart dependency and integration tests. See **[subchart/README.md](subchart/README.MD)** for detailed documentation.

- **Global value inheritance** - Tests value passing from parent to child charts
- **Dependency management** - Tests chart dependency resolution and installation
- **Integration scenarios** - Tests agent as subchart in umbrella charts

## Makefile Integration

The Helm tests provide comprehensive Makefile integration with granular control over test execution. All tests run from the project root and are organized into logical targets.

### Required Tools

These are automatically installed by `make install-tools`:

- `helm` - Helm CLI tool
- `kubeconform` - Kubernetes manifest validation tool
- `helm-unittest` plugin - Unit testing for Helm charts

```bash
# Install development tools (includes helm, kubeconform, etc.)
make install-tools

# Verify helm unittest plugin is installed
helm plugin list | grep unittest
```

## Running Helm Tests

Helm tests ARE fully integrated with the main Makefile and run from the project root.

### High-Level Test Targets

```bash
# Run complete Helm test suite (all validation types)
make helm-test

# Run all schema validation tests
make helm-test-schema

# Generate all templates from override files
make helm-test-template

# Test all chart dependencies and compositions
make helm-test-subchart

# Run all Helm unittest tests
make helm-test-unittest
```

### Granular Test Control

```bash
# Schema validation subtypes
make helm-test-schema-template     # Template rendering only
make helm-test-schema-kubeconform  # Kubernetes API validation only

# Template generation with change detection
make helm-test-template-diff       # Compare with git version

# Individual test execution (see subdirectory READMEs for details)
make tests/helm/schema/components.webhookServer.backfill.schedule.valid3.pass
make tests/helm/template/manifest.yaml
```

For detailed information on running specific tests and understanding test patterns, see:

- **[Schema test documentation](schema/)** - Individual schema test execution
- **[Template test documentation](template/)** - Template generation and comparison
- **[Subchart test documentation](subchart/)** - Dependency and integration testing

## Adding New Tests

For specific guidance on adding tests, see the detailed documentation:

- **[Adding schema tests](schema/)** - Creating validation tests for chart values and templates
- **[Adding template tests](template/)** - Creating baseline template generation tests
- **[Adding subchart tests](subchart/)** - Creating dependency and integration tests

## Integration with CI/CD

These tests integrate automatically with:

- `make lint` - Includes `helm-lint`
- `make helm-test` - Complete validation suite
- `make analyze` - Includes Checkov security analysis of templates

## Troubleshooting

For detailed troubleshooting information, see the subdirectory documentation:

- **[Schema test troubleshooting](schema/)** - Debugging template rendering and validation issues
- **[Template test troubleshooting](template/)** - Debugging template generation and comparison issues
- **[Subchart test troubleshooting](subchart/)** - Debugging dependency and integration issues

### Quick Debug Commands

```bash
# Reinstall tools if needed
make install-tools

# Test template manually
helm template test-release ./helm --values tests/helm/template/manifest-overrides.yml

# Check Kubernetes version compatibility
make helm-test-schema-kubeconform KUBE_VERSION=1.29.0
```

This comprehensive test suite ensures the CloudZero Agent Helm chart works correctly across different configurations and Kubernetes versions while catching issues early in development.
