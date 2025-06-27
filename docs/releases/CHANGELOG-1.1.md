# CloudZero Agent 1.1.X - Changelog

## Overview

The 1.1.X release series introduced the **CloudZero Aggregator**, a high-performance, cost-efficient replacement for the CloudZero OLTP metrics API, along with significant enhancements to reliability, performance, and user experience.

## Major Features

### CloudZero Aggregator
- **High-Performance Collector**: Local aggregator application for enhanced telemetry processing
- **End-to-End Telemetry Tracking**: Complete visibility and traceability of telemetry files
- **Resilience During Key Rotation**: Seamless API key rotation with Multiple API Key feature
- **Direct S3 Upload**: Data uploads directly to dedicated customer S3 buckets
- **Improved Onboarding Feedback**: Faster deployment configuration feedback (typically 10 minutes)
- **Configurable Upload Intervals**: Flexible upload timing (default: 10 minutes)

### Architecture Improvements
- **Single Binary and Single Version**: Unified CloudZero image reference tagged to chart release
- **Simplified Image Management**: Streamlined image mirroring, versioning, and operational identification
- **Consolidated Webhooks**: Multiple Kubernetes validating webhooks merged into single webhook

## Performance Enhancements

### Monitoring and Metrics
- **Improved Scrape Frequency**: Metrics captured every 1 minute (previously 2 minutes)
- **Greater Granularity**: Enhanced monitoring precision and faster actionable insights
- **Enhanced HTTP Error Logging**: Improved debugging and monitoring efficiency

### Configuration and Deployment
- **New Configuration API**: Minimalistic Helm values/overrides API for future compatibility
- **Automatic DNS Configuration**: Auto-generated DNS configuration and priority class settings
- **Enhanced Disk Management**: Configurable disk monitoring and improved space management
- **Pod Disruption Budgets**: Added for higher availability during maintenance

## Reliability and Stability

### Enhanced Functionality
- **Improved Labels and Annotations**: Better performance for backfilling operations
- **Enhanced Debug Logging**: Debug logging for abandoned file IDs in shipper component
- **Reduced Complexity**: Consolidated webhooks reduce operational complexity

### Compatibility Improvements (1.1.1)
- **Extended Kubernetes Support**: Reduced requirement from 1.23 to 1.21
- **Expanded Installation Compatibility**: Support for clusters back to mid-2022 EOL versions
- **Reduced Permissions**: Removed patch permission requirement on deployments for cert initialization

## Bug Fixes Across 1.1.X Series

### 1.1.0 Fixes
- **Duplicate Affinity**: Resolved affinity settings duplication in insights deployment
- **Agent Termination**: Fixed hang issue during agent termination in redeployments
- **Validation Improvement**: Enhanced validation and check results output

### 1.1.1 Fixes
- **Shipper Directory Creation**: Recursive subdirectory creation prevents restart failures
- **Dependency Updates**: Various dependency updates for improved stability

### 1.1.2 Fixes
- Additional maintenance and stability improvements

## Documentation and User Experience

### Comprehensive Documentation
- **API Scopes Guidance**: Detailed required API scopes for Kubernetes agent
- **Smoother Onboarding**: Enhanced configuration guidance
- **Operational Identification**: Clearer versioning and image identification

### Configuration Flexibility
- **Configurable Components**: Images, labels, annotations, tolerations, affinities
- **Node Selectors**: Flexible node selector configuration
- **Priority Classes**: Configurable priority class settings
- **DNS Settings**: Enhanced DNS configuration options

## Breaking Changes

- **OLTP API Replacement**: CloudZero Aggregator replaces OLTP metrics API
- **Webhook Consolidation**: Multiple webhooks consolidated into single webhook
- **Configuration Changes**: New configuration API may require override file updates

## Upgrade Path

To upgrade to any 1.1.X version:

```bash
helm upgrade --install <RELEASE_NAME> cloudzero/cloudzero-agent -n <NAMESPACE> --create-namespace -f configuration.example.yaml --version 1.1.X
```

## Version History

- **1.1.0** (2025-04-29): Major release with CloudZero Aggregator
- **1.1.1** (2025-05-02): Maintenance release expanding compatibility
- **1.1.2**: Additional stability and maintenance improvements

---

*This changelog covers the major features and changes introduced in the CloudZero Agent 1.1.X release series.*