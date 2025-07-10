// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package types defines core types and interfaces for cloud environment detection
// and metadata retrieval.
package types

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
