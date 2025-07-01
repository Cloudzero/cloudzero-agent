// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package types defines core types and interfaces for cloud environment detection
// and metadata retrieval.
package types

import "context"

//go:generate mockgen -destination=mocks/scout_mock.go -package=mocks . Scout

// CloudProvider represents the detected cloud provider.
type CloudProvider string

const (
	// CloudProviderAWS represents Amazon Web Services
	CloudProviderAWS CloudProvider = "aws"
	// CloudProviderGoogle represents Google Cloud Platform
	CloudProviderGoogle CloudProvider = "google"
	// CloudProviderAzure represents Microsoft Azure
	CloudProviderAzure CloudProvider = "azure"
	// CloudProviderUnknown represents an undetected or unsupported cloud provider
	CloudProviderUnknown CloudProvider = "unknown"
	// CloudProviderMock represents a mock provider for testing
	CloudProviderMock CloudProvider = "mock"
)

// EnvironmentInfo contains cloud environment information retrieved from
// metadata services.
//
// This structure provides the essential information needed for cloud cost
// allocation and monitoring:
//   - CloudProvider: The detected cloud provider
//   - Region: The cloud region/location where the instance is running
//   - AccountID: The cloud account identifier
//   - ClusterName: The cluster identifier for the environment
type EnvironmentInfo struct {
	CloudProvider CloudProvider `json:"cloudProvider"`
	Region        string        `json:"region"`
	AccountID     string        `json:"accountId"`
	ClusterName   string        `json:"clusterName"`
}

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
