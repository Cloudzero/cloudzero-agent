// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package oracle provides Oracle Cloud Infrastructure (OCI) environment
// detection and metadata retrieval using the OCI Instance Metadata Service
// (IMDS) v2.
package oracle

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
	// ociMetadataURL is the OCI Instance Metadata Service v2 endpoint.
	// https://docs.oracle.com/en-us/iaas/Content/Compute/Tasks/gettingmetadata.htm
	ociMetadataURL = "http://169.254.169.254/opc/v2/instance/"

	// authorizationHeader and authorizationHeaderValue are required by OCI
	// IMDSv2; requests without them are rejected. The value is a fixed, public
	// scheme constant, not a credential.
	authorizationHeader      = "Authorization"
	authorizationHeaderValue = "Bearer Oracle" // #nosec G101 - not a credential, a fixed IMDSv2 scheme value

	requestTimeout = 5 * time.Second
)

// Scout detects and describes an OCI environment.
type Scout struct {
	client *http.Client
}

// instanceMetadata is the subset of the OCI IMDS response we consume.
type instanceMetadata struct {
	// CanonicalRegionName is the full region identifier (e.g. "us-ashburn-1"),
	// matching the region naming CloudZero uses elsewhere.
	CanonicalRegionName string `json:"canonicalRegionName"`
	// Region is the short region key (e.g. "iad"); retained as an additional
	// detection signal.
	Region string `json:"region"`
	// TenantID is the tenancy OCID. It is deliberately NOT used as the account
	// ID (see EnvironmentInfo); it serves only as a positive OCI detection
	// signal.
	TenantID string `json:"tenantId"`
}

// NewScout creates a new OCI metadata scout.
func NewScout() *Scout {
	return &Scout{
		client: &http.Client{
			Timeout: requestTimeout,
		},
	}
}

// fetchMetadata retrieves and parses the OCI IMDSv2 instance document.
func (s *Scout) fetchMetadata(ctx context.Context) (*instanceMetadata, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ociMetadataURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set(authorizationHeader, authorizationHeaderValue)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get OCI metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get OCI metadata, status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read OCI metadata response: %w", err)
	}

	var md instanceMetadata
	if err := json.Unmarshal(body, &md); err != nil {
		return nil, fmt.Errorf("failed to parse OCI metadata JSON: %w", err)
	}

	return &md, nil
}

// Detect determines whether the current environment is running on OCI by
// querying the Instance Metadata Service. Network failures, non-200 responses,
// and unparseable bodies are treated as "not OCI" rather than errors.
func (s *Scout) Detect(ctx context.Context) (types.CloudProvider, error) {
	md, err := s.fetchMetadata(ctx)
	if err != nil {
		return types.CloudProviderUnknown, nil //nolint:nilerr // inability to reach/parse IMDS means "not OCI", not a hard error
	}

	// A populated OCI-specific field confirms we're on OCI. This is intentionally
	// more lenient than EnvironmentInfo, which additionally requires
	// canonicalRegionName; detection should succeed on any positive OCI signal.
	if md.TenantID != "" || md.CanonicalRegionName != "" || md.Region != "" {
		return types.CloudProviderOCI, nil
	}

	return types.CloudProviderUnknown, nil
}

// EnvironmentInfo retrieves OCI environment information from the Instance
// Metadata Service.
func (s *Scout) EnvironmentInfo(ctx context.Context) (*types.EnvironmentInfo, error) {
	md, err := s.fetchMetadata(ctx)
	if err != nil {
		return nil, err
	}

	region := strings.TrimSpace(md.CanonicalRegionName)
	if region == "" {
		return nil, errors.New("canonicalRegionName not found in OCI metadata")
	}

	return &types.EnvironmentInfo{
		CloudProvider: types.CloudProviderOCI,
		Region:        region,
		// AccountID is intentionally left empty. OCI's Instance Metadata Service
		// exposes the tenancy OCID (tenantId), not the numeric account ID that
		// CloudZero expects, so it cannot be auto-detected and must be configured
		// manually (cloudAccountId in the Helm chart). Returning a value here
		// would also clobber any customer-provided value, since the
		// scout-detection path treats detected values as authoritative over
		// configured ones.
		//
		// ClusterName is likewise not exposed by the metadata service and is left
		// empty, consistent with the AWS scout.
	}, nil
}
