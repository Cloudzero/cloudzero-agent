# CloudZero Scout Package

The CloudZero Scout package provides automatic cloud environment detection for AWS deployments. It identifies the current cloud provider and retrieves essential information like account ID and region.

## Purpose

Scout is designed to reduce the need for manual cloud environment configuration in CloudZero Agent deployments. It automatically detects whether the application is running on AWS and retrieves the necessary metadata for cost allocation and monitoring.

## Supported Cloud Providers

- **Amazon Web Services (AWS)** - Retrieves Account ID and Region via Instance Metadata Service (IMDS)

## Key Features

- **Automatic Detection** - No configuration required, detects cloud provider automatically
- **Context-Aware** - Respects timeouts and cancellation via Go context
- **Testable** - Provides mock interfaces for unit testing

## Integration

Scout is integrated into CloudZero Agent configuration packages to enable automatic environment detection when configuration values are empty.
