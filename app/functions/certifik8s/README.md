# CloudZero Certifik8s

A Go-based TLS certificate management tool for CloudZero webhook configurations.
This tool replaces the previous bash script implementation to provide reliable
certificate generation, validation, and Kubernetes resource management without
external dependencies.

## Overview

The `cloudzero-certifik8s` tool is responsible for:

- **Certificate Generation**: Creating self-signed TLS certificates for webhook
  configurations
- **Certificate Validation**: Verifying existing certificates for validity and
  expiration
- **Kubernetes Integration**: Patching TLS secrets and webhook configurations

**⚠️ Security Note**: This tool requires cluster-scoped Kubernetes permissions to
manage TLS secrets and webhook configurations. These permissions are necessary
for the CloudZero Agent's admission controller functionality but are carefully
restricted to only specific resources. See the [Required
Permissions](#required-permissions) section for detailed explanations and
alternative approaches.

## Use Cases

This tool is primarily used by the CloudZero Agent Helm chart during deployment to:

1. **Initialize TLS certificates** for the insights controller webhook
2. **Validate existing certificates** to avoid unnecessary regeneration
3. **Update Kubernetes resources** with new certificate data
4. **Support cert-manager integration** by skipping certificate generation when enabled

## Features

- **Self-signed Certificate Generation**: Creates RSA 2048-bit certificates with 100-year validity
- **Automatic SAN Detection**: Generates certificates with proper DNS names for Kubernetes services
- **Smart Validation**: Checks certificate expiration, SANs, and format validity
- **Kubernetes Native**: Uses direct k8s.io/client-go library calls for resource management
- **Testable Design**: Interface-based architecture for comprehensive testing
- **CLI Framework**: Built with Cobra for consistent command-line experience
- **Secure RBAC**: Implements cluster-scoped permissions with resource-specific restrictions

## Installation

The tool is included in the CloudZero Agent container image and is automatically
available at `/app/cloudzero-certifik8s`.

## Usage

### Command Line Interface

```bash
cloudzero-certifik8s [flags]
```

### Required Flags

| Flag             | Description                         | Example                             |
| ---------------- | ----------------------------------- | ----------------------------------- |
| `--secret-name`  | Name of the TLS secret to manage    | `--secret-name=cloudzero-agent-tls` |
| `--namespace`    | Kubernetes namespace for the secret | `--namespace=default`               |
| `--service-name` | Service name for certificate SAN    | `--service-name=cloudzero-agent`    |
| `--webhook-name` | ValidatingWebhookConfiguration name | `--webhook-name=cloudzero-agent`    |

### Optional Flags

| Flag                   | Description                             | Default |
| ---------------------- | --------------------------------------- | ------- |
| `--enable-labels`      | Enable label-based webhook updates      | `false` |
| `--enable-annotations` | Enable annotation-based webhook updates | `false` |

### Examples

#### Basic Usage

```bash
cloudzero-certifik8s \
  --secret-name=cloudzero-agent-tls \
  --namespace=default \
  --service-name=cloudzero-agent \
  --webhook-name=cloudzero-agent
```

#### With Labels and Annotations

```bash
cloudzero-certifik8s \
  --secret-name=cloudzero-agent-tls \
  --namespace=default \
  --service-name=cloudzero-agent \
  --webhook-name=cloudzero-agent \
  --enable-labels \
--enable-annotations
```

#### Using cert-manager

```bash
cloudzero-certifik8s \
  --secret-name=cloudzero-agent-tls \
  --namespace=default \
  --service-name=cloudzero-agent \
  --webhook-name=cloudzero-agent
```

## How It Works

### 1. Certificate Decision Logic

The tool first determines if a new certificate is needed by checking:

1. **Webhook Configuration**: Examines the `caBundle` field in the ValidatingWebhookConfiguration
2. **TLS Secret**: Checks for existing `tls.crt` and `tls.key` in the specified secret
3. **Certificate Validation**: Validates existing certificates for expiration and SAN compatibility

### 2. Certificate Generation

When a new certificate is needed, the tool:

1. **Generates RSA Key**: Creates a 2048-bit private key
2. **Creates Certificate**: Generates a self-signed certificate:
3. **Encodes Data**: Base64 encodes raw DER bytes for Kubernetes secret format

### 3. Resource Updates

The tool updates Kubernetes resources:

1. **TLS Secret**: Patches the secret with new certificate data
2. **Webhook Configuration**: Updates the `caBundle` field if labels/annotations are enabled

## Required Permissions

**✅ Security Note**: This tool uses the most restrictive RBAC permissions possible while maintaining full functionality.

### Kubernetes RBAC

The tool requires **cluster-scoped permissions** because `validatingwebhookconfigurations` are cluster-scoped resources. However, security is maintained through **resource-specific access** using `resourceNames`:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole # Required for cluster-scoped resources
metadata:
  name: cloudzero-agent-webhook-server-init-cert
  annotations:
    checkov.io/skip_1: CKV_K8S_155 # Suppress legitimate security warning
rules:
  # Only access the specific TLS secret needed
  - apiGroups: [""]
    resources: ["secrets"]
    resourceNames: ["cz-agent-cloudzero-agent-webhook-server-tls"] # Specific secret only
    verbs: ["get", "patch"]

  # Only access the specific webhook configuration needed
  - apiGroups: ["admissionregistration.k8s.io"]
    resources: ["validatingwebhookconfigurations"]
    resourceNames: ["cz-agent-cloudzero-agent-webhook-server-webhook"] # Specific webhook only
    verbs: ["get", "patch"]
```

This is how the CloudZero Agent chart grants permissions.

### Permission Breakdown

#### Secrets Access (`""` API Group)

- **`get` permission on specific `secrets`**: Required to read the existing TLS certificate and key to determine if regeneration is needed
- **`patch` permission on specific `secrets`**: Required to update the TLS secret with newly generated certificate data
- **Security Impact**: **Minimal** - Access is restricted to only the specific secret needed. No access to other secrets.

#### Webhook Configuration Access (`admissionregistration.k8s.io` API Group)

- **`get` permission on specific `validatingwebhookconfigurations`**: Required to read the current `caBundle` field to determine if certificate updates are needed
- **`patch` permission on specific `validatingwebhookconfigurations`**: Required to update the `caBundle` field with new certificate data
- **Security Impact**: **Minimal** - Access is restricted to only the specific webhook configuration needed (`${CHART_NAME}-cloudzero-agent-webhook-server-webhook`). No access to other webhook configurations.
- **Cluster Scope Required**: `validatingwebhookconfigurations` are cluster-scoped resources, necessitating ClusterRole permissions

### Why These Permissions Are Required

1. **Certificate Management**: The tool must read existing certificates to validate them and determine if regeneration is needed
2. **Secret Updates**: New certificates must be stored in the specific Kubernetes secret for the webhook server to use
3. **Webhook Configuration**: The tool updates only the specific webhook configuration to ensure the new certificate is trusted
4. **Admission Controller Functionality**: Without these permissions, the CloudZero Agent's admission controller cannot function, breaking cost allocation and resource tracking capabilities

### Why Cluster-Scoped Permissions Are Necessary

**ValidatingWebhookConfigurations are Cluster-Scoped Resources**

- **Cluster Level**: These resources exist at the cluster level, not within individual namespaces
- **Global Access**: They control admission webhooks across the entire cluster
- **Namespace Isolation**: Namespace-scoped permissions cannot access cluster-scoped resources
- **Security Maintained**: Despite requiring cluster scope, access is restricted to only specific resource names

**Alternative Approaches Considered**

- **Namespace-scoped Role**: Initially attempted but failed because webhook configs are cluster-scoped
- **Broad ClusterRole**: Rejected for security reasons - would grant access to all webhook configs
- **Current Solution**: ClusterRole with resourceNames provides the right balance of functionality and security

### Service Account

The tool runs with a service account that has the minimal required permissions:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: cloudzero-agent-webhook-server-init-cert
  namespace: cza # Limited to specific namespace
```

The service account is bound to the ClusterRole using a ClusterRoleBinding:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: cloudzero-agent-webhook-server-init-cert
subjects:
  - kind: ServiceAccount
    name: cloudzero-agent-webhook-server-init-cert
    namespace: cza
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cloudzero-agent-webhook-server-init-cert
```

### Alternative Approaches

#### Use cert-manager (Alternative to this tool)

- **Enable `useCertManager: true` in your Helm values**
- **Benefits**: No additional permissions required, certificates managed by cert-manager
- **Trade-offs**: Requires cert-manager to be installed and configured in your cluster
- **When to use**: If you already have cert-manager and prefer its certificate management

## Troubleshooting

### Common Issues

#### Certificate Generation Fails

- **Check permissions**: Ensure the service account has RBAC permissions
- **Verify namespace**: Confirm the namespace exists and is accessible
- **Check kubectl**: Ensure kubectl is available and configured

#### Webhook Updates Fail

- **Verify webhook exists**: Check that the ValidatingWebhookConfiguration exists
- **Check permissions**: Ensure patch permissions on webhook resources
- **Validate flags**: Confirm `--enable-labels` or `--enable-annotations` is set

#### Secret Patching Fails

- **Check secret exists**: Verify the TLS secret exists in the namespace
- **Verify permissions**: Ensure patch permissions on secret resources
- **Check format**: Confirm the secret has the expected structure

#### Permission-Related Issues

- **RBAC errors**: Check that the service account has the required ClusterRole and ClusterRoleBinding
- **Forbidden errors**: Verify the service account has access to the target namespace and resources
- **Admission webhook errors**: Ensure the service account can read and patch webhook configurations
- **Cluster-scoped access**: Verify ClusterRoleBinding exists and service account is properly bound
- **Resource names**: Confirm the resourceNames in the ClusterRole match the actual resource names
