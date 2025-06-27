// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package google provides functionality for detecting and gathering environment
// information from Google Cloud metadat services.
package google

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/cloudzero/cloudzero-agent/app/utils/scout/types"
)

const (
	// GCP metadata service endpoints
	metadataBaseURL     = "http://metadata.google.internal/computeMetadata/v1"
	projectEndpoint     = metadataBaseURL + "/project/project-id"
	zoneEndpoint        = metadataBaseURL + "/instance/zone"
	clusterNameEndpoint = metadataBaseURL + "/instance/attributes/cluster-name"

	requestTimeout = 5 * time.Second
)

type Scout struct {
	client *http.Client
}

// NewScout creates a new GCP metadata scout
func NewScout() *Scout {
	return &Scout{
		client: &http.Client{
			Timeout: requestTimeout,
		},
	}
}

// EnvironmentInfo retrieves GCP environment information from metadata service
func (s *Scout) EnvironmentInfo(ctx context.Context) (*types.EnvironmentInfo, error) {
	// Get project ID (equivalent to account ID)
	projectID, err := s.getMetadata(ctx, projectEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get project ID: %w", err)
	}

	// Get zone (to extract region)
	zone, err := s.getMetadata(ctx, zoneEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get zone: %w", err)
	}

	// Extract region from zone (e.g., "projects/123456789/zones/us-central1-a" -> "us-central1")
	region := s.extractRegionFromZone(zone)

	// Get cluster name (may not be available in non-GKE environments)
	clusterName, err := s.getMetadata(ctx, clusterNameEndpoint)
	if err != nil {
		// Cluster name is optional - log debug info but don't fail
		// This allows the scout to work in non-GKE environments
		clusterName = ""
	}

	return &types.EnvironmentInfo{
		CloudProvider: types.CloudProviderGoogle,
		Region:        strings.TrimSpace(region),
		AccountID:     strings.TrimSpace(projectID),
		ClusterName:   strings.TrimSpace(clusterName),
	}, nil
}

// getMetadata retrieves metadata from the specified GCP metadata endpoint
func (s *Scout) getMetadata(ctx context.Context, endpoint string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return "", err
	}

	// GCP metadata service requires this header
	req.Header.Set("Metadata-Flavor", "Google")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get metadata, status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// extractRegionFromZone extracts the region from a GCP zone string
// Zone format: "projects/{project}/zones/{zone}" or just "{zone}"
// Zone examples: "us-central1-a", "europe-west1-b"
// Region examples: "us-central1", "europe-west1"
func (s *Scout) extractRegionFromZone(zone string) string {
	// Handle full zone path format
	if strings.Contains(zone, "/zones/") {
		parts := strings.Split(zone, "/zones/")
		if len(parts) == 2 {
			zone = parts[1]
		}
	}

	// Extract region from zone by removing the last part after the last hyphen
	// e.g., "us-central1-a" -> "us-central1"
	zoneParts := strings.Split(zone, "-")
	if len(zoneParts) > 1 {
		// Join all parts except the last one
		region := strings.Join(zoneParts[:len(zoneParts)-1], "-")
		return region
	}

	// If we can't parse it properly, return the zone as-is
	return zone
}

// Detect determines if the current environment is running on GCP by testing the metadata service.
func (s *Scout) Detect(ctx context.Context) (types.CloudProvider, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", metadataBaseURL+"/", nil)
	if err != nil {
		return types.CloudProviderUnknown, err
	}

	// GCP metadata service requires this header
	req.Header.Set("Metadata-Flavor", "Google")

	resp, err := s.client.Do(req)
	if err != nil {
		// Network errors mean we can't detect, but not an error condition
		return types.CloudProviderUnknown, nil
	}
	defer resp.Body.Close()

	// Consider 2xx status codes as successful detection
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices {
		return types.CloudProviderGoogle, nil
	}

	return types.CloudProviderUnknown, nil
}
