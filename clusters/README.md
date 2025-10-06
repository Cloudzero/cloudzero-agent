# Cluster Configuration System

This directory contains cluster configuration files that define how to connect to and configure different Kubernetes clusters for CloudZero Agent deployment and testing. The system uses the `CLUSTER_NAME` environment variable to dynamically select cluster configurations.

## Overview

The cluster configuration system provides a unified way to:

- **Target different clusters** - Switch between development, staging, and production environments
- **Configure cluster access** - Specify kubeconfig files, contexts, and namespaces
- **Customize deployments** - Override Helm values per cluster
- **Simplify workflows** - Use consistent commands across different environments

## File Structure

Each cluster configuration consists of two files:

- **`<cluster-name>.yaml`** - Cluster connection configuration (kubeconfig, namespace, release name)
- **`<cluster-name>-overrides.yaml`** - Helm values overrides for the CloudZero Agent deployment

**Example files:**

```text
clusters/
├── kind.yaml                 # KIND cluster connection config
├── kind-overrides.yaml       # KIND cluster Helm overrides
├── dev.yaml                  # Development cluster config (your file)
└── dev-overrides.yaml        # Development cluster overrides (your file)
```

## Configuration File Format

### Cluster Configuration (`<name>.yaml`)

Defines how to connect to the cluster:

```yaml
# clusters/kind.yaml
kubeConfig: tests/kuttl/kubeconfig # Path to kubeconfig file (optional)
namespace: im-a-namespace # Target namespace
release: im-a-release # Helm release name
context: my-context # Kubectl context (optional)
```

**Fields:**

- **`kubeConfig`** _(optional)_ - Path to kubeconfig file. If not specified, uses system default
- **`namespace`** _(required)_ - Kubernetes namespace for agent deployment
- **`release`** _(required)_ - Helm release name for the CloudZero Agent
- **`context`** _(optional)_ - Kubectl context to use within the kubeconfig

### Helm Overrides (`<name>-overrides.yaml`)

Defines cluster-specific Helm values:

```yaml
# clusters/kind-overrides.yaml
clusterName: kind
apiKey: "test-api-key-for-local-testing"
cloudAccountId: "1234567890"
region: "us-east-1"
```

## Using the CLUSTER_NAME Variable

The `CLUSTER_NAME` environment variable selects which cluster configuration to use:

### Basic Usage

```bash
# Use KIND cluster (this is the default if CLUSTER_NAME is not set)
CLUSTER_NAME=kind make helm-install

# Or simply use the default
make helm-install  # Same as CLUSTER_NAME=kind

# Use development cluster
CLUSTER_NAME=dev make helm-install

# Use production cluster
CLUSTER_NAME=production make helm-install
```

### Variable Resolution

When `CLUSTER_NAME=dev`, the system uses:

- **Cluster config:** `clusters/dev.yaml`
- **Helm overrides:** `clusters/dev-overrides.yaml`

**Default:** If `CLUSTER_NAME` is not set, it defaults to `kind`.

## Makefile Integration

The cluster configuration system is deeply integrated with Make targets:

### Helm Operations

```bash
# Install chart to specific cluster
CLUSTER_NAME=my-cluster make helm-install

# Install with current image tag (uses dev-$(git rev-parse HEAD) tag)
CLUSTER_NAME=my-cluster make helm-install-current

# Install and wait for deployment to be ready
CLUSTER_NAME=my-cluster make helm-install helm-wait

# Uninstall from cluster
CLUSTER_NAME=my-cluster make helm-uninstall
```

### Common Development Tasks

```bash
# List available clusters
ls clusters/*.yaml | sed 's/clusters\///g' | sed 's/\.yaml//g' | grep -v overrides

# Wait for deployment to be fully ready
CLUSTER_NAME=my-cluster make helm-wait

# Get logs for specific service (example)
kubectl --context="$(.tools/bin/gojq --yaml-input -r .context clusters/my-cluster.yaml)" -n "$(.tools/bin/gojq --yaml-input -r .namespace clusters/my-cluster.yaml)" logs deployment/my-release-aggregator -c my-release-aggregator-collector

# Run KUTTL integration tests
CLUSTER_NAME=my-cluster make helm-test-kuttl
```

### Testing Operations

```bash
# Run KUTTL tests against specific cluster
CLUSTER_NAME=my-cluster make helm-test-kuttl
```

### Low-Level Operations

The configuration system provides helper functions used by Makefile targets:

- **`$(call get-cluster-property,.namespace)`** - Extract namespace from cluster config
- **`$(call get-kubeconfig-env)`** - Get KUBECONFIG environment variable setting
- **`$(call invoke-kubectl)`** - Generate kubectl command with proper context
- **`$(call invoke-helm)`** - Generate helm command with cluster settings
- **`$(call invoke-kuttl)`** - Generate kuttl command with kubeconfig

## Creating New Cluster Configurations

### 1. Create Cluster Configuration File

```yaml
# clusters/my-cluster.yaml
kubeConfig: ~/.kube/my-cluster-config
namespace: cloudzero-agent
release: cz-agent
context: my-cluster-context
```

### 2. Create Helm Overrides File

```yaml
# clusters/my-cluster-overrides.yaml
clusterName: my-cluster
apiKey: "${CLOUDZERO_DEV_API_KEY}"
cloudAccountId: "987654321"
region: "us-west-2"

components:
  agent:
    replicas: 3
    resources:
      requests:
        cpu: 200m
        memory: 256Mi
```

### 3. Test the Configuration

```bash
# Test cluster connectivity
CLUSTER_NAME=my-cluster make helm-lint

# Deploy to cluster
CLUSTER_NAME=my-cluster make helm-install

# Run tests
CLUSTER_NAME=my-cluster make helm-test-kuttl
```

## Git Ignore Strategy

The `.gitignore` file in this directory:

```gitignore
# Ignore all cluster configuration files except the example kuttl-testing files
*.yaml
!/kind.yaml
!/kind-overrides.yaml
```

**This means:**

- **`kind.yaml` and `kind-overrides.yaml`** - Committed as examples for testing
- **All other `*.yaml` files** - Ignored by git, allowing personal cluster configs
- **You can freely add your own cluster files** without affecting the repository

## Common Cluster Configurations

### Development Cluster

```yaml
# clusters/dev.yaml
kubeConfig: ~/.kube/dev-config
namespace: cloudzero-dev
release: cz-agent-dev
```

```yaml
# clusters/dev-overrides.yaml
clusterName: dev-cluster
apiKey: "$(CLOUDZERO_DEV_API_KEY)"
cloudAccountId: "$(DEV_CLOUD_ACCOUNT_ID)"
region: "us-east-1"

components:
  agent:
    image:
      tag: "latest"
    logLevel: debug
```

### Staging Cluster

```yaml
# clusters/staging.yaml
kubeConfig: ~/.kube/staging-config
namespace: cloudzero-staging
release: cz-agent-staging
context: staging-context
```

```yaml
# clusters/staging-overrides.yaml
clusterName: staging-cluster
apiKey: "$(CLOUDZERO_STAGING_API_KEY)"
cloudAccountId: "$(STAGING_CLOUD_ACCOUNT_ID)"
region: "us-west-2"

components:
  agent:
    replicas: 2
    resources:
      requests:
        cpu: 100m
        memory: 128Mi
```

### Production Cluster

```yaml
# clusters/production.yaml
kubeConfig: ~/.kube/production-config
namespace: cloudzero
release: cloudzero-agent
context: production-context
```

```yaml
# clusters/production-overrides.yaml
clusterName: production-cluster
apiKey: "$(CLOUDZERO_PRODUCTION_API_KEY)"
cloudAccountId: "$(PRODUCTION_CLOUD_ACCOUNT_ID)"
region: "us-east-1"

components:
  agent:
    replicas: 5
    resources:
      requests:
        cpu: 500m
        memory: 1Gi
      limits:
        cpu: 2000m
        memory: 4Gi
```

## Local Configuration

Use `local-config.mk` for sensitive data and personal settings:

```makefile
# local-config.mk (ignored by git)
CLOUDZERO_DEV_API_KEY = your-dev-api-key
CLUSTER_NAME = brahms
```

## Specialized Configurations (Advanced)

Specialized configurations allow us to test cluster configurations that are relatively uncommon using the exact same tests that we run on our standard clusters. This increases test coverage of these more obscure configurations by ensuring they work with the full test suite.

### Using HELM_EXTRA_OVERRIDES

The `HELM_EXTRA_OVERRIDES` environment variable allows you to specify an additional values file that overrides both the chart defaults and cluster overrides:

**Priority Order (later takes precedence):**

1. `helm/values.yaml` (chart defaults)
2. `clusters/$(CLUSTER_NAME)-overrides.yaml` (cluster-specific overrides)
3. `$(HELM_EXTRA_OVERRIDES)` (specialized configuration overrides)
4. `--set` arguments (command-line overrides)

### Example: Using Specialized Configuration

```sh
# Create temporary specialized configuration
cat <<EOF > /tmp/test-config-overrides.yaml
# Enable specific features for this test run
defaults:
  federation:
    enabled: true
EOF

# Deploy and test
make helm-install-current HELM_EXTRA_OVERRIDES=/tmp/test-config-overrides.yaml
make helm-test-kuttl
make helm-uninstall
```

### Notes

- Extra overrides files are only used if they exist (no error if file not found)
- This system is designed for testing specialized configurations, not for production deployments
- For production, use proper cluster-specific overrides files instead
- The `get-helm-extra-overrides` function in the Makefile handles the conditional inclusion

## Troubleshooting

### Cluster Configuration Issues

```bash
# Verify cluster config files exist
ls clusters/my-cluster.yaml clusters/my-cluster-overrides.yaml

# Test cluster connectivity
CLUSTER_NAME=my-cluster kubectl get nodes

# Validate Helm overrides
CLUSTER_NAME=my-cluster make helm-lint
```
