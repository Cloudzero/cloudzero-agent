// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package azure provides Azure cloud environment detection and metadata retrieval
// capabilities using the Azure Instance Metadata Service (IMDS).
package azure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cloudzero/cloudzero-agent/app/utils/scout/types"
)

const (
	// azureMetadataURL is the URL for the Azure Instance Metadata Service (IMDS)
	// https://docs.microsoft.com/en-us/azure/virtual-machines/windows/instance-metadata-service
	azureMetadataURL = "http://169.254.169.254/metadata/instance?api-version=2021-02-01"

	// metadataHeader is the required header for Azure IMDS requests
	metadataHeader = "Metadata"

	// maxDNSLabelLength is the maximum length for a DNS label (RFC 1035)
	// This applies to Kubernetes cluster names which must be valid DNS labels
	maxDNSLabelLength = 63

	requestTimeout = 5 * time.Second
)

type Scout struct {
	client *http.Client
}

// instanceMetadata represents the structure of Azure IMDS response
type instanceMetadata struct {
	Compute struct {
		Location          string `json:"location"`
		SubscriptionID    string `json:"subscriptionId"`
		ResourceGroupName string `json:"resourceGroupName"`
	} `json:"compute"`
}

// NewScout creates a new Azure metadata scout
func NewScout() *Scout {
	return &Scout{
		client: &http.Client{
			Timeout: requestTimeout,
		},
	}
}

// Detect determines if the current environment is running on Azure by testing the metadata service.
func (s *Scout) Detect(ctx context.Context) (types.CloudProvider, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", azureMetadataURL, nil)
	if err != nil {
		return types.CloudProviderUnknown, err
	}

	// Azure metadata service requires this header
	req.Header.Set(metadataHeader, "true")

	resp, err := s.client.Do(req)
	if err != nil {
		// Network errors mean we can't detect, but not an error condition
		return types.CloudProviderUnknown, nil
	}
	defer resp.Body.Close()

	// Consider 2xx status codes as successful detection
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return types.CloudProviderAzure, nil
	}

	return types.CloudProviderUnknown, nil
}

// EnvironmentInfo retrieves Azure environment information from Instance Metadata Service
func (s *Scout) EnvironmentInfo(ctx context.Context) (*types.EnvironmentInfo, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", azureMetadataURL, nil)
	if err != nil {
		return nil, err
	}

	// Azure metadata service requires this header
	req.Header.Set(metadataHeader, "true")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get Azure metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get Azure metadata, status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read Azure metadata response: %w", err)
	}

	var metadata instanceMetadata
	if err := json.Unmarshal(body, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse Azure metadata JSON: %w", err)
	}

	if metadata.Compute.SubscriptionID == "" {
		return nil, errors.New("subscriptionId not found in Azure metadata")
	}

	if metadata.Compute.Location == "" {
		return nil, errors.New("location not found in Azure metadata")
	}

	// Extract cluster name using heuristic approach (best effort)
	clusterName := s.extractClusterName(metadata.Compute.ResourceGroupName)

	return &types.EnvironmentInfo{
		CloudProvider: types.CloudProviderAzure,
		Region:        strings.TrimSpace(metadata.Compute.Location),
		AccountID:     strings.TrimSpace(metadata.Compute.SubscriptionID),
		ClusterName:   clusterName,
	}, nil
}

// extractClusterName attempts to extract the AKS cluster name from the managed
// resource group name.
//
// Note that Azure IMDS only provides the managed resource group information
// (MC_*), not the original resource group where the AKS cluster was created.
// This creates a fundamental ambiguity in parsing cluster names when both
// resource group and cluster names contain underscores.
//
// Azure AKS creates a managed resource group with the pattern:
// MC_{resourceGroupName}_{clusterName}_{region}
//
// This implementation assumes cluster names do not contain underscores, but
// resource group names might. Under this assumption, we can parse
// deterministically:
//
// - The region is the last underscore-separated component
// - The cluster name is the second-to-last component
// - The resource group name is everything between "MC_" and "_{clusterName}_{region}"
//
// However, note that the cluster name *can* contain underscores. In this
// situation, you will unfortunately need to specify the cluster name manually.
func (s *Scout) extractClusterName(resourceGroupName string) string {
	// Must start with "MC_"
	if !strings.HasPrefix(resourceGroupName, "MC_") {
		return ""
	}

	// Remove "MC_" prefix and split by underscores
	withoutPrefix := strings.TrimPrefix(resourceGroupName, "MC_")
	parts := strings.Split(withoutPrefix, "_")

	// Need at least 3 parts: {resourceGroup}_{clusterName}_{region}
	// Resource group could be multiple parts joined by underscores
	if len(parts) < 3 {
		return ""
	}

	// Under the assumption that cluster names don't contain underscores:
	// - parts[len(parts)-1] is the region
	// - parts[len(parts)-2] is the cluster name
	clusterName := parts[len(parts)-2]

	// Validate cluster name (basic DNS label validation)
	if len(clusterName) == 0 || len(clusterName) > maxDNSLabelLength {
		return ""
	}

	return clusterName
}
