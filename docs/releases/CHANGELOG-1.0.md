# CloudZero Agent 1.0.X - Changelog

## Overview

The 1.0.X release series introduced native Kubernetes **Labels** and **Annotations** support to the CloudZero platform, marking a major milestone in resource categorization and management capabilities.

## Major Features

### Kubernetes Labels and Annotations Support

- **Native Integration**: Direct support for Kubernetes Labels and Annotations within the CloudZero platform
- **Enhanced Categorization**: Improved ability to categorize and manage resources based on Labels and Annotations
- **Dimension Identification**: Kubernetes dimensions can now be identified based on deployment Labels and Annotations

### New Components

- **Insights Controller**: New ValidatingAdmissionWebhook for recording created labels and annotations
- **Service Account Management**: New service account configuration for the Insights Controller
- **Certificate Management**: Integration with Jetstack.io "cert-manager" for TLS certificate handling

## Configuration Changes

### New Configuration Options

- `cert-manager.enabled`: Deploy cert-manager (default: depends on environment)
- `serviceAccount.create`: Create service account (default: true)
- `insightsController.enabled`: Enable ValidatingAdmissionWebhook (default: true)
- `insightsController.labels.enabled`: Enable label collection (default: true)
- `insightsController.annotations.enabled`: Enable annotation collection (default: false)
- Label and annotation pattern filtering with regular expressions

### API Key Management Changes

- API key arguments moved to `global` section
- `apiKey` → `global.apiKey`
- `existingSecretName` → `global.existingSecretName`

## Breaking Changes and Deprecations

### Deprecated Components

- **node-exporter**: Completely deprecated and no longer used
- **External kube-state-metrics**: Replaced with internal `cloudzero-state-metrics` instance

### Configuration Breaking Changes

- API key management arguments relocated to global section
- Some existing values no longer necessary in override configurations

## Bug Fixes Across 1.0.X Series

### 1.0.1 Fixes

- Fixed webhook resource naming validation issues
- Resolved TLS certificate generation for webhook configuration changes
- Fixed invalid Prometheus metric label names causing panics
- Removed default Kubernetes logger usage for proper logging level respect

### 1.0.2 Fixes

- Template rendering improvements
- Enhanced certificate generation reliability

## Improvements

### Performance and Reliability

- Shorter TTL for `init-cert` Job (5 seconds cleanup)
- Improved SQLite connection handling and testing
- Enhanced logging with appropriate debug/info level separation
- Improved validation and check results output

### Documentation

- Added comprehensive Istio cluster documentation
- Enhanced upgrade instructions and configuration examples
- Detailed security scan results and vulnerability reporting

## Security

### Vulnerability Status

All images in 1.0.X series show zero critical, high, medium, low, or negligible vulnerabilities according to Grype security scans.

## Upgrade Path

To upgrade to any 1.0.X version:

```bash
helm repo add cloudzero https://cloudzero.github.io/cloudzero-charts
helm repo update
helm upgrade --install <RELEASE_NAME> cloudzero/cloudzero-agent -n <NAMESPACE> --create-namespace -f configuration.yaml --version 1.0.X
```

## Version History

- **1.0.0** (2025-02-17): Initial major release with Labels/Annotations support
- **1.0.1** (2025-03-02): Bug fixes for template rendering and TLS certificates
- **1.0.2**: Additional stability improvements

---

_This changelog covers the major features and changes introduced in the CloudZero Agent 1.0.X release series._
