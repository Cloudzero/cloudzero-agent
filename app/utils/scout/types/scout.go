// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package types defines core types and interfaces for cloud environment detection
// and metadata retrieval.
package types

import "context"

// Scout defines the interface for cloud environment detection and metadata
// retrieval.
type Scout interface {
	// Detect determines if the current environment is running in this cloud
	// provider.
	//
	// Returns:
	//   - CloudProvider: CloudProvider value if detected (e.g.,
	//     CloudProviderAWS), CloudProviderUnknown if not detected
	//   - error: If detection fails due to network or other errors
	Detect(ctx context.Context) (CloudProvider, error)

	// EnvironmentInfo retrieves cloud environment information including
	// cloud provider, region, and account ID.
	//
	// The context should include a reasonable timeout (e.g., 10 seconds) to
	// prevent hanging if metadata services are unreachable.
	//
	// Returns:
	//   - EnvironmentInfo: The detected environment information
	//   - error: If detection or retrieval fails
	EnvironmentInfo(ctx context.Context) (*EnvironmentInfo, error)
}
