// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package aws provides AWS cloud environment detection and metadata retrieval
// capabilities using the EC2 instance metadata service (IMDS) v2.
package aws

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
	// EC2 metadata service endpoints
	metadataBaseURL     = "http://169.254.169.254/latest/meta-data"
	tokenURL            = "http://169.254.169.254/latest/api/token" // #nosec G101 - This is a metadata service URL, not a credential
	regionEndpoint      = metadataBaseURL + "/placement/region"
	instanceIDEndpoint  = metadataBaseURL + "/instance-id"
	identityDocEndpoint = "http://169.254.169.254/latest/dynamic/instance-identity/document"

	// HTTP headers
	tokenTTLHeader = "X-aws-ec2-metadata-token-ttl-seconds" // #nosec G101 - This is a header name, not a credential
	tokenHeader    = "X-aws-ec2-metadata-token"             // #nosec G101 - This is a header name, not a credential

	requestTimeout = 5 * time.Second
	tokenTTL       = "21600" // 6 hours
)

type Scout struct {
	client *http.Client
}

type identityDocument struct {
	AccountID string `json:"accountId"`
	Region    string `json:"region"`
}

// NewScout creates a new AWS metadata scout
func NewScout() *Scout {
	return &Scout{
		client: &http.Client{
			Timeout: requestTimeout,
		},
	}
}

// Detect determines if the current environment is running on AWS by testing the metadata service.
// AWS metadata service returns 401 when accessed without IMDSv2 token, which is a positive indicator.
func (s *Scout) Detect(ctx context.Context) (types.CloudProvider, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", metadataBaseURL+"/", nil)
	if err != nil {
		return types.CloudProviderUnknown, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		// Network errors mean we can't detect, but not an error condition
		return types.CloudProviderUnknown, nil
	}
	defer resp.Body.Close()

	// AWS returns 401 for unauthenticated requests to the metadata service
	// This is a positive indicator that we're on AWS
	if resp.StatusCode == http.StatusUnauthorized {
		return types.CloudProviderAWS, nil
	}

	return types.CloudProviderUnknown, nil
}

// EnvironmentInfo retrieves AWS environment information from EC2 metadata service
func (s *Scout) EnvironmentInfo(ctx context.Context) (*types.EnvironmentInfo, error) {
	// Get IMDSv2 token
	token, err := s.getToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get IMDSv2 token: %w", err)
	}

	// Get region
	region, err := s.getMetadata(ctx, regionEndpoint, token)
	if err != nil {
		return nil, fmt.Errorf("failed to get region: %w", err)
	}

	// Get account ID from identity document
	accountID, err := s.getAccountID(ctx, token)
	if err != nil {
		return nil, fmt.Errorf("failed to get account ID: %w", err)
	}

	return &types.EnvironmentInfo{
		CloudProvider: types.CloudProviderAWS,
		Region:        strings.TrimSpace(region),
		AccountID:     strings.TrimSpace(accountID),
	}, nil
}

// getToken retrieves an IMDSv2 token
func (s *Scout) getToken(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "PUT", tokenURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set(tokenTTLHeader, tokenTTL)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get token, status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// getMetadata retrieves metadata from the specified endpoint using IMDSv2
func (s *Scout) getMetadata(ctx context.Context, endpoint, token string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set(tokenHeader, token)

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

// getAccountID retrieves the AWS account ID from the identity document using proper JSON parsing
func (s *Scout) getAccountID(ctx context.Context, token string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", identityDocEndpoint, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set(tokenHeader, token)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get identity document, status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Parse the JSON response properly
	var doc identityDocument
	if err := json.Unmarshal(body, &doc); err != nil {
		return "", fmt.Errorf("failed to parse identity document JSON: %w", err)
	}

	if doc.AccountID == "" {
		return "", errors.New("accountId not found in identity document")
	}

	return doc.AccountID, nil
}
