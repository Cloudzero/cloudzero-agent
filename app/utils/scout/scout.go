// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package scout provides comprehensive cloud environment detection and metadata retrieval
// capabilities for multi-cloud deployments. It serves as the central orchestrator
// for cloud provider-specific detection logic.
//
// The scout package enables automatic discovery of:
//   - Cloud provider identification (AWS, Azure, Google Cloud)
//   - Account/subscription/project IDs for cost attribution
//   - Regional location information
//   - Cluster and instance metadata
//   - Network and compute resource details
//
// Architecture:
//   - Auto-detection orchestrator that tries multiple cloud providers
//   - Provider-specific scout implementations (AWS, Azure, GCP)
//   - Unified interface for consistent metadata retrieval
//   - Timeout and error handling for reliable detection
//
// Detection strategy:
//   1. Auto-detection tries each cloud provider in sequence
//   2. Provider-specific scouts check metadata endpoints
//   3. First successful detection wins
//   4. Graceful fallback if detection fails
//
// This package is critical for CloudZero agent functionality as it provides
// the cloud context required for proper cost attribution and resource organization.
//
// Usage:
//   scout := NewScout()
//   metadata, err := scout.Detect(ctx)
//   if err == nil {
//       log.Printf("Detected %s in %s", metadata.Provider, metadata.Region)
//   }
package scout

import (
	"github.com/cloudzero/cloudzero-agent/app/utils/scout/auto"
	"github.com/cloudzero/cloudzero-agent/app/utils/scout/aws"
	"github.com/cloudzero/cloudzero-agent/app/utils/scout/azure"
	"github.com/cloudzero/cloudzero-agent/app/utils/scout/google"
	"github.com/cloudzero/cloudzero-agent/app/utils/scout/types"
)

// NewScout creates a new Scout implementation with comprehensive auto-detection
// capabilities across all supported cloud providers. The returned scout will
// attempt detection in the order: AWS, Azure, Google Cloud.
//
// The auto-detection scout tries each provider-specific implementation in sequence
// until one succeeds or all fail. This provides robust cloud environment discovery
// without requiring explicit provider configuration.
//
// Returns:
//   - types.Scout: Auto-detection scout that orchestrates provider-specific detection
//
// Provider detection order:
//   1. AWS: Checks EC2 instance metadata service and IAM role information
//   2. Azure: Checks Azure instance metadata service and subscription details
//   3. Google Cloud: Checks GCE metadata service and project information
//
// Each provider scout implements timeout handling, retry logic, and graceful
// error handling to ensure reliable detection even in challenging network conditions.
//
// Example:
//   scout := NewScout()
//   ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//   defer cancel()
//   
//   metadata, err := scout.Detect(ctx)
//   if err != nil {
//       log.Printf("Cloud detection failed: %v", err)
//       return
//   }
//   
//   log.Printf("Detected cloud: %s, account: %s, region: %s",
//       metadata.Provider, metadata.AccountID, metadata.Region)
func NewScout() types.Scout {
	return auto.NewScout(
		aws.NewScout(),
		azure.NewScout(),
		google.NewScout(),
	)
}
