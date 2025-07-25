// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package aws provides AWS cloud environment detection and metadata retrieval
// capabilities using the EC2 instance metadata service (IMDS) v2 with fallback to v1.
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

// ErrIMDSv2Unavailable is returned when IMDSv2 token endpoint is not available
var ErrIMDSv2Unavailable = errors.New("IMDSv2 token endpoint unavailable, falling back to IMDSv1")

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
// For IMDSv1 environments, we check for AWS-specific metadata fields.
func (s *Scout) Detect(ctx context.Context) (types.CloudProvider, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, metadataBaseURL+"/", nil)
	if err != nil {
		return types.CloudProviderUnknown, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		// Network errors mean we can't detect, but not an error condition
		return types.CloudProviderUnknown, nil
	}
	defer resp.Body.Close()

	// AWS returns 401 for unauthenticated requests to the metadata service This
	// is a positive indicator that we're on AWS, though IMDSv2 is required.
	if resp.StatusCode == http.StatusUnauthorized {
		return types.CloudProviderAWS, nil
	}

	// If we get a 200 response, it might be IMDSv1 environment
	// Check for AWS-specific metadata fields to confirm
	if resp.StatusCode == http.StatusOK {
		// Try to get instance-id which is unique to AWS
		instanceReq, err := http.NewRequestWithContext(ctx, http.MethodGet, metadataBaseURL+"/instance-id", nil)
		if err != nil {
			return types.CloudProviderUnknown, nil
		}

		instanceResp, err := s.client.Do(instanceReq)
		if err != nil {
			return types.CloudProviderUnknown, nil
		}
		defer instanceResp.Body.Close()

		// If we can get instance-id, we're on AWS
		if instanceResp.StatusCode == http.StatusOK {
			body, err := io.ReadAll(instanceResp.Body)
			if err != nil {
				return types.CloudProviderUnknown, nil
			}

			// If we can retrieve an instance ID from the metadata service, we're on AWS
			// Instance IDs can have various formats, so we just check for non-empty response
			if strings.TrimSpace(string(body)) != "" {
				return types.CloudProviderAWS, nil
			}
		}
	}

	return types.CloudProviderUnknown, nil
}

// EnvironmentInfo retrieves AWS environment information from EC2 metadata service
// with IMDSv2 support and fallback to IMDSv1 for compatibility.
func (s *Scout) EnvironmentInfo(ctx context.Context) (*types.EnvironmentInfo, error) {
	// Try IMDSv2 first
	token, err := s.getToken(ctx)
	if err != nil {
		if errors.Is(err, ErrIMDSv2Unavailable) {
			// Fall back to IMDSv1
			return s.environmentInfoIMDSv1(ctx)
		}
		return nil, fmt.Errorf("failed to get IMDSv2 token: %w", err)
	}

	// Get region using IMDSv2
	region, err := s.getMetadata(ctx, regionEndpoint, token)
	if err != nil {
		return nil, fmt.Errorf("failed to get region: %w", err)
	}

	// Get account ID from identity document using IMDSv2
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

// environmentInfoIMDSv1 retrieves AWS environment information using IMDSv1
func (s *Scout) environmentInfoIMDSv1(ctx context.Context) (*types.EnvironmentInfo, error) {
	// Get region using IMDSv1
	region, err := s.getMetadataIMDSv1(ctx, regionEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get region using IMDSv1: %w", err)
	}

	// Get account ID from identity document using IMDSv1
	accountID, err := s.getAccountIDIMDSv1(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get account ID using IMDSv1: %w", err)
	}

	return &types.EnvironmentInfo{
		CloudProvider: types.CloudProviderAWS,
		Region:        strings.TrimSpace(region),
		AccountID:     strings.TrimSpace(accountID),
	}, nil
}

// getToken retrieves an IMDSv2 token
func (s *Scout) getToken(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, tokenURL, nil)
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
		switch resp.StatusCode {
		case http.StatusMethodNotAllowed:
			return "", ErrIMDSv2Unavailable
		case http.StatusForbidden:
			return "", fmt.Errorf("access denied to IMDSv2 token endpoint (status: %d). Check IAM permissions and metadata service configuration", resp.StatusCode)
		case http.StatusUnauthorized:
			return "", fmt.Errorf("authentication required for IMDSv2 token endpoint (status: %d)", resp.StatusCode)
		default:
			return "", fmt.Errorf("failed to get IMDSv2 token, unexpected status: %d", resp.StatusCode)
		}
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// getMetadata retrieves metadata from the specified endpoint using IMDSv2
func (s *Scout) getMetadata(ctx context.Context, endpoint, token string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
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

// getMetadataIMDSv1 retrieves metadata from the specified endpoint using IMDSv1
func (s *Scout) getMetadataIMDSv1(ctx context.Context, endpoint string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get metadata using IMDSv1, status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// getAccountID retrieves the AWS account ID from the identity document using proper JSON parsing
func (s *Scout) getAccountID(ctx context.Context, token string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, identityDocEndpoint, nil)
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

// getAccountIDIMDSv1 retrieves the AWS account ID from the identity document using IMDSv1
func (s *Scout) getAccountIDIMDSv1(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, identityDocEndpoint, nil)
	if err != nil {
		return "", err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to get identity document using IMDSv1, status: %d", resp.StatusCode)
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
