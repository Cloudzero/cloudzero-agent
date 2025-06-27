# CloudZero Agent 1.2.X - Changelog

## Overview

The 1.2.X release series introduced **Federated Mode** support, comprehensive **Helm Schema Validation**, enhanced **Configuration Management**, and numerous improvements to observability, reliability, and maintainability.

## Major Features

### Federated Mode Support
- **Large Cluster Support**: Agent deployment optimized for large cluster environments
- **Node-Level Deployment**: Agent runs on each node instead of single agent for all nodes
- **Configuration**: Enable with `defaults.federation.enabled: true`

### Enhanced Configuration Management
- **Comprehensive Helm Schema Validation**: Extended JSON Schema validation covering entire configuration
- **Early Feedback**: Earlier detection and reporting of configuration issues
- **Configuration ConfigMap**: Complete Helm chart configuration stored in ConfigMap for easier debugging

### New Tools and Utilities
- **CloudZero Helmless Tool**: Shows minimal differences between default and actual configuration
- **Minimized Overrides**: Recreates minimized override files
- **Helmless Job**: Helm job for easy minimal configuration override determination

## Performance and Efficiency Improvements

### Load Balancing and Connectivity
- **Improved Load Balancing**: Enhanced HTTP connection handling
- **Periodic Connection Rotation**: Proper load distribution across service replicas
- **Multi-Replica Support**: Optimized for multi-replica deployments

### Storage Optimization
- **Reduced Storage Usage**: Metric files stored for 7 days (previously 90 days)
- **Significant Storage Reduction**: Dramatically reduced storage requirements
- **Cost Efficiency**: Lower operational costs through optimized retention

## Observability and Debugging

### Enhanced Logging
- **Configurable Prometheus Log Levels**: Flexible logging configuration
- **Reduced Log Noise**: Health checks moved to trace level
- **Positive Confirmation Logging**: Regular info-level messages confirm proper operation
- **Improved Debugging**: Better visibility into agent operations

### Monitoring Improvements
- **Enhanced Error Handling**: Better error reporting and debugging capabilities
- **Operational Visibility**: Improved insight into agent performance and health

## Reliability and Bug Fixes

### Major Bug Fixes Across 1.2.X Series

#### 1.2.0 Fixes
- **Eliminate Unnecessary Replays**: Fixed shipper file replay issues
- **Out-of-Order Metrics**: Added configuration window for out-of-order metric acceptance (default: 5 minutes)

#### 1.2.1 Fixes
- **Subchart Schema Validation**: Fixed JSON Schema validation for subchart usage
- **Global Property Support**: Resolved Helm subchart global property validation errors

#### 1.2.2 Fixes
- **Configuration Management**: Fixed component-specific configuration merging issues
- **ConfigMap References**: Updated ConfigMap name references to correct naming convention
- **Resource Lookup Failures**: Prevented resource lookup failures
- **Template Generation**: Fixed invalid Kubernetes resource template generation
- **Label Filtering**: Aggregator no longer filters out "resource_type" and "workload" labels

## Security and Availability

### Enhanced Security
- **Default Pod Disruption Budgets**: Improved availability during disruptions
- **Schema Validation**: Comprehensive validation prevents configuration errors
- **Resource Protection**: Better resource naming and reference handling

### Testing and Quality Assurance
- **Subchart Testing**: Comprehensive test coverage for subchart scenarios
- **Regression Prevention**: Tests prevent schema validation regression
- **Resource Creation Verification**: Checks ensure successful Kubernetes resource creation
- **Template Validation**: Kubeconform tests validate generated templates

## Development and Maintenance

### Code Quality Improvements
- **Helmless Tool Enhancement**: Split implementation with enhanced testing coverage
- **Unnecessary Functionality Removal**: Cleaned up unused code paths
- **Testing Infrastructure**: Improved validation and testing frameworks

### Dependency Management
- **Updated Dependencies**: Regular dependency updates for security and stability
- **Component Isolation**: Better separation of concerns in tool implementations

## Breaking Changes

- **Storage Retention**: Default metric file retention reduced from 90 to 7 days
- **Schema Validation**: Stricter validation may catch previously ignored configuration errors
- **ConfigMap Naming**: Updated naming conventions may affect existing references

## Upgrade Path

To upgrade to any 1.2.X version:

```bash
helm upgrade --install <RELEASE_NAME> cloudzero/cloudzero-agent -n <NAMESPACE> --create-namespace -f configuration.example.yaml --version 1.2.X
```

## Version History

- **1.2.0** (2025-06-05): Major release with Federated Mode and Schema Validation
- **1.2.1** (2025-06-17): Bugfix release for subchart schema validation
- **1.2.2** (2025-06-24): Maintenance release with configuration and template fixes

---

*This changelog covers the major features and changes introduced in the CloudZero Agent 1.2.X release series.*