# CloudZero Scout Package

The CloudZero Scout package provides automatic cloud environment detection across the major cloud providers. It identifies the current cloud provider and retrieves essential information like account ID and region.

## Purpose

Scout is designed to reduce the need for manual cloud environment configuration in CloudZero Agent deployments. It automatically detects which cloud provider the application is running on and retrieves the necessary metadata for cost allocation and monitoring.

## Supported Cloud Providers

- **Amazon Web Services (AWS)** - Retrieves Account ID and Region via the Instance Metadata Service (IMDS)
- **Microsoft Azure** - Retrieves Subscription ID, Region, and (best-effort) cluster name via the Azure Instance Metadata Service
- **Google Cloud (GCP)** - Retrieves project number and Region via the GCE metadata server
- **Oracle Cloud Infrastructure (OCI)** - Retrieves Region via the OCI Instance Metadata Service v2. The account ID is **not** auto-detected: OCI exposes only the tenancy OCID, not the numeric account ID CloudZero uses, so `cloudAccountId` must be set manually on OKE.

## Key Features

- **Automatic Detection** - No configuration required, detects cloud provider automatically
- **Context-Aware** - Respects timeouts and cancellation via Go context
- **Testable** - Provides mock interfaces for unit testing

## Integration

Scout is integrated into CloudZero Agent configuration packages to enable automatic environment detection when configuration values are empty.
