# Helm Template Generation Tests

This directory contains template generation tests that ensure Helm chart modifications are intentional and produce the expected Kubernetes manifests. These tests help catch unintended changes to the generated resources.

## Overview

Template generation tests provide:

- **Baseline templates** - Reference manifests generated from specific configurations
- **Change detection** - Identification of modifications to generated resources
- **Regression prevention** - Ensures chart changes don't break existing functionality
- **Configuration validation** - Verification that overrides produce expected results

## File Structure

Each test consists of two files:

- **`<name>-overrides.yml`** - Helm values to override chart defaults
- **`<name>.yaml`** - Generated Kubernetes manifests (baseline)

**Example:**

- `manifest-overrides.yml` - Values for basic deployment
- `manifest.yaml` - Generated manifests from those values

## Available Template Tests

### Basic Deployment (`manifest.*`)

Tests standard deployment configuration:

```yaml
# manifest-overrides.yml
cloudAccountId: "1234567890"
clusterName: "my-cluster"
region: "us-east-1"
apiKey: "not-a-real-api-key"
jobConfigID: "DEADBEEF-FEED-FACE-CAFE-FEE10D15EA5E"
```

**What it tests:**

- Basic agent deployment
- Core required configurations
- Standard resource creation

### Federated Configuration (`federated.*`)

Tests federated deployment scenarios:

**What it tests:**

- Multi-cluster configurations
- Federation-specific settings
- Cross-cluster resource management

### Certificate Manager Integration (`cert-manager.*`)

Tests integration with cert-manager:

**What it tests:**

- TLS certificate management
- Certificate issuer configuration
- Webhook certificate automation

## Running Template Tests

### Generate All Templates

```bash
# Generate all template files from their overrides
make helm-test-template
```

### Generate Specific Templates

```bash
# Generate specific template
make tests/helm/template/manifest.yaml
make tests/helm/template/federated.yaml
make tests/helm/template/cert-manager.yaml
```

### Compare Templates with Git Version

```bash
# Compare all templates with committed versions
make helm-test-template-diff

# Compare specific template with semantic diff
make tests/helm/template/manifest.yaml-semdiff
```

## Template Generation Process

### How Templates are Generated

1. **Load chart** - Helm loads the CloudZero Agent chart
2. **Apply overrides** - Values from `-overrides.yml` file override defaults
3. **Render templates** - Helm generates Kubernetes manifests
4. **Save output** - Generated manifests saved to `.yaml` file

### Generated Resources

Templates typically include:

- **Deployments** - Agent pods and containers
- **Services** - Network exposure for components
- **ConfigMaps** - Configuration data
- **Secrets** - Sensitive information (API keys, certificates)
- **ServiceAccounts** - RBAC identity
- **ClusterRoles/ClusterRoleBindings** - Permissions
- **Webhooks** - Admission controllers
- **CronJobs** - Scheduled tasks

## Template Validation Workflow

### Development Workflow

1. **Modify chart templates** - Make changes to Helm templates
2. **Update overrides** - Modify test configurations if needed
3. **Regenerate templates** - Run `make helm-test-template`
4. **Review changes** - Use diff tools to see modifications
5. **Commit changes** - Include both template and generated manifest changes

### Change Detection

```bash
# See what changed in templates
make helm-test-template-diff

# Review changes before committing
git diff tests/helm/template/*.yaml
```

### Reviewing Changes

When templates change, review:

- **Resource modifications** - New, removed, or changed resources
- **Configuration changes** - Modified environment variables, volumes, etc.
- **Security implications** - Changes to RBAC, network policies, certificates
- **Backward compatibility** - Ensure existing deployments aren't broken

## Adding New Template Tests

### Create New Test Configuration

1. **Create override file** - `<test-name>-overrides.yml`
2. **Add configuration** - Include test-specific Helm values
3. **Generate baseline** - Run `make tests/helm/template/<test-name>.yaml`
4. **Commit both files** - Include both override and generated files

**Example new test:**

```yaml
# monitoring-overrides.yml
components:
  insightsController:
    enabled: true
    monitoring:
      enabled: true
      serviceMonitor:
        enabled: true

prometheus:
  enabled: true
  alerting:
    enabled: true
```

### Test Scenarios

Consider creating tests for:

- **Component combinations** - Different enabled/disabled components
- **Environment variations** - Development, staging, production configs
- **Integration scenarios** - External service integrations
- **Security configurations** - Different security settings
- **Resource constraints** - Various resource limits and requests

## Template Comparison and Analysis

### Using Diff Tools

```bash
# Standard diff
diff tests/helm/template/manifest.yaml.previous tests/helm/template/manifest.yaml

# Semantic diff (understands YAML structure)
make tests/helm/template/manifest.yaml-semdiff

# Git-based comparison
git diff HEAD^ tests/helm/template/manifest.yaml
```

### Key Areas to Review

When reviewing template changes:

1. **Resource Names** - Ensure consistent naming conventions
2. **Labels and Annotations** - Verify metadata is correct
3. **Environment Variables** - Check configuration propagation
4. **Volume Mounts** - Ensure proper data access
5. **Network Policies** - Verify connectivity requirements
6. **Security Contexts** - Review permissions and constraints

## Troubleshooting Template Tests

### Common Issues

1. **Template rendering failures:**

   ```bash
   # Debug template generation
   helm template test-release ../../helm/chart --values <test>-overrides.yml --debug
   ```

2. **Unexpected output:**

   ```bash
   # Compare with previous version
   git show HEAD:tests/helm/template/<test>.yaml > /tmp/previous.yaml
   diff /tmp/previous.yaml tests/helm/template/<test>.yaml
   ```

3. **Missing resources:**
   ```bash
   # Verify override values are correct
   helm template test-release ../../helm/chart --values <test>-overrides.yml --show-only templates/deployment.yaml
   ```

### Validation Steps

Before committing template changes:

1. **Verify generation** - Ensure templates generate without errors
2. **Review diffs** - Understand all changes
3. **Test deployment** - Deploy generated manifests to test cluster
4. **Run schema validation** - Ensure manifests pass kubeconform
5. **Update documentation** - Document significant changes

## Best Practices

### Override File Design

- **Minimal overrides** - Only include necessary changes from defaults
- **Clear intent** - Make test purpose obvious from configuration
- **Realistic values** - Use plausible (but fake) data
- **Security awareness** - Don't include real credentials or sensitive data

### Template Maintenance

- **Regular updates** - Regenerate templates when chart changes
- **Meaningful commits** - Include both template and generated changes together
- **Change documentation** - Explain significant template modifications
- **Backward compatibility** - Consider impact on existing deployments

Template generation tests ensure the CloudZero Agent Helm chart produces consistent, expected Kubernetes manifests across different configurations.
