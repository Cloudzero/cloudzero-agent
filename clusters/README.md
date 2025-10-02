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

components:
  agent:
    image:
      repository: ghcr.io/cloudzero/cloudzero-agent
      tag: "intentionally-invalid-tag"
```

## Using the CLUSTER_NAME Variable

The `CLUSTER_NAME` environment variable selects which cluster configuration to use:

### Basic Usage

```bash
# Use KIND cluster (default)
CLUSTER_NAME=kind make helm-install

# Use development cluster
CLUSTER_NAME=dev make helm-install

# Use production cluster
CLUSTER_NAME=production make helm-install
```

### Variable Resolution

When `CLUSTER_NAME=dev`, the system uses:

- **Cluster config:** `clusters/dev.yaml`
- **Helm overrides:** `clusters/dev-overrides.yaml`

If `CLUSTER_NAME` is not set, it defaults to `kind`.

## Makefile Integration

The cluster configuration system is deeply integrated with Make targets:

### Helm Operations

```bash
# Install chart to specific cluster
CLUSTER_NAME=brahms make helm-install

# Install with current image tag
CLUSTER_NAME=brahms make helm-install-current

# Uninstall from cluster
CLUSTER_NAME=brahms make helm-uninstall
```

### Testing Operations

```bash
# Run KUTTL tests against specific cluster
CLUSTER_NAME=brahms make helm-test-kuttl
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
- **All other `*.yaml` files** - Ignored, allowing personal cluster configs
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
