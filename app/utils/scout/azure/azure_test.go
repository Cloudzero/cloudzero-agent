// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package azure

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cloudzero/cloudzero-agent/app/utils/scout/types"
	"github.com/stretchr/testify/assert"
)

const (
	validAzureMetadataResponse = `{
		"compute": {
			"location": "eastus",
			"subscriptionId": "f192f712-2862-4c95-9d8b-fc95c72dc795",
			"resourceGroupName": "MC_azure-cirrus-evan_azure-cirrus-evan_eastus"
		}
	}`

	minimalAzureMetadataResponse = `{
		"compute": {
			"location": "westus2",
			"subscriptionId": "12345678-1234-1234-1234-123456789012",
			"resourceGroupName": "MC_my-rg_my-cluster_westus2"
		}
	}`

	complexAzureMetadataResponse = `{
		"compute": {
			"location": "northeurope",
			"subscriptionId": "abcdef12-3456-7890-abcd-ef1234567890",
			"resourceGroupName": "MC_my_complex_rg_my_test_cluster_v2_northeurope"
		}
	}`
)

func TestNewScout(t *testing.T) {
	scout := NewScout()
	assert.NotNil(t, scout)
	assert.NotNil(t, scout.client)
	assert.Equal(t, requestTimeout, scout.client.Timeout)
}

func TestScout_Detect_Success(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		expected   types.CloudProvider
	}{
		{
			name:       "successful detection with 200",
			statusCode: 200,
			expected:   types.CloudProviderAzure,
		},
		{
			name:       "successful detection with 201",
			statusCode: 201,
			expected:   types.CloudProviderAzure,
		},
		{
			name:       "successful detection with 202",
			statusCode: 202,
			expected:   types.CloudProviderAzure,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "true", r.Header.Get("Metadata"))
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(validAzureMetadataResponse))
			}))
			defer server.Close()

			scout := createScoutWithCustomURL(server.URL)
			ctx := context.Background()

			result, err := scout.Detect(ctx)

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestScout_Detect_NotAzure(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		expected   types.CloudProvider
	}{
		{
			name:       "detection fails with 404",
			statusCode: 404,
			expected:   types.CloudProviderUnknown,
		},
		{
			name:       "detection fails with 500",
			statusCode: 500,
			expected:   types.CloudProviderUnknown,
		},
		{
			name:       "detection fails with 403",
			statusCode: 403,
			expected:   types.CloudProviderUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			scout := createScoutWithCustomURL(server.URL)
			ctx := context.Background()

			result, err := scout.Detect(ctx)

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestScout_Detect_NetworkError(t *testing.T) {
	scout := NewScout()
	scout.client.Timeout = 10 * time.Millisecond // Very short timeout

	ctx := context.Background()
	result, err := scout.Detect(ctx)

	// Network errors should not return an error, just unknown provider
	assert.NoError(t, err)
	assert.Equal(t, types.CloudProviderUnknown, result)
}

func TestScout_Detect_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(validAzureMetadataResponse))
	}))
	defer server.Close()

	scout := createScoutWithCustomURL(server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	result, err := scout.Detect(ctx)

	// Network errors return no error, but context timeout should return unknown
	assert.NoError(t, err)
	assert.Equal(t, types.CloudProviderUnknown, result)
}

func TestScout_EnvironmentInfo_Success(t *testing.T) {
	tests := []struct {
		name         string
		responseBody string
		expectedInfo *types.EnvironmentInfo
	}{
		{
			name:         "successful environment info retrieval",
			responseBody: validAzureMetadataResponse,
			expectedInfo: &types.EnvironmentInfo{
				CloudProvider: types.CloudProviderAzure,
				Region:        "eastus",
				AccountID:     "f192f712-2862-4c95-9d8b-fc95c72dc795",
				ClusterName:   "azure-cirrus-evan",
			},
		},
		{
			name:         "environment info with simple resource group pattern",
			responseBody: minimalAzureMetadataResponse,
			expectedInfo: &types.EnvironmentInfo{
				CloudProvider: types.CloudProviderAzure,
				Region:        "westus2",
				AccountID:     "12345678-1234-1234-1234-123456789012",
				ClusterName:   "my-cluster",
			},
		},
		{
			name:         "environment info with complex resource group pattern",
			responseBody: complexAzureMetadataResponse,
			expectedInfo: &types.EnvironmentInfo{
				CloudProvider: types.CloudProviderAzure,
				Region:        "northeurope",
				AccountID:     "abcdef12-3456-7890-abcd-ef1234567890",
				ClusterName:   "v2",
			},
		},
		{
			name:         "handles whitespace in response fields",
			responseBody: `{"compute": {"location": "  eastus  ", "subscriptionId": "  12345  ", "resourceGroupName": "MC_rg_cluster_eastus"}}`,
			expectedInfo: &types.EnvironmentInfo{
				CloudProvider: types.CloudProviderAzure,
				Region:        "eastus",
				AccountID:     "12345",
				ClusterName:   "cluster",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "true", r.Header.Get("Metadata"))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			scout := createScoutWithCustomURL(server.URL)
			ctx := context.Background()

			result, err := scout.EnvironmentInfo(ctx)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedInfo, result)
		})
	}
}

func TestScout_EnvironmentInfo_Errors(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		responseBody  string
		errorContains string
	}{
		{
			name:          "error on non-200 status",
			statusCode:    500,
			responseBody:  "",
			errorContains: "failed to get Azure metadata, status: 500",
		},
		{
			name:          "error on invalid JSON",
			statusCode:    200,
			responseBody:  `{invalid json`,
			errorContains: "failed to parse Azure metadata JSON",
		},
		{
			name:          "error on missing subscription ID",
			statusCode:    200,
			responseBody:  `{"compute": {"location": "eastus", "resourceGroupName": "MC_rg_cluster_eastus"}}`,
			errorContains: "subscriptionId not found in Azure metadata",
		},
		{
			name:          "error on missing location",
			statusCode:    200,
			responseBody:  `{"compute": {"subscriptionId": "12345", "resourceGroupName": "MC_rg_cluster_eastus"}}`,
			errorContains: "location not found in Azure metadata",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				if tt.responseBody != "" {
					_, _ = w.Write([]byte(tt.responseBody))
				}
			}))
			defer server.Close()

			scout := createScoutWithCustomURL(server.URL)
			ctx := context.Background()

			result, err := scout.EnvironmentInfo(ctx)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorContains)
			assert.Nil(t, result)
		})
	}
}

func TestScout_EnvironmentInfo_NetworkError(t *testing.T) {
	scout := NewScout()
	scout.client.Timeout = 10 * time.Millisecond // Very short timeout

	ctx := context.Background()
	result, err := scout.EnvironmentInfo(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get Azure metadata")
	assert.Nil(t, result)
}

func TestScout_EnvironmentInfo_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate slow response
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(validAzureMetadataResponse))
	}))
	defer server.Close()

	scout := createScoutWithCustomURL(server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	result, err := scout.EnvironmentInfo(ctx)

	// Should get context timeout error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
	assert.Nil(t, result)
}

func TestScout_EnvironmentInfo_ReadError(t *testing.T) {
	// Create a server that closes the connection after sending headers
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// Write headers but then close connection before body
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		if conn, ok := w.(http.Hijacker); ok {
			if c, _, err := conn.Hijack(); err == nil {
				_ = c.Close()
			}
		}
	}))
	defer server.Close()

	scout := createScoutWithCustomURL(server.URL)
	ctx := context.Background()

	result, err := scout.EnvironmentInfo(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read Azure metadata response")
	assert.Nil(t, result)
}

func TestScout_extractClusterName(t *testing.T) {
	s := &Scout{}
	tests := []struct {
		name                 string
		managedResourceGroup string
		expected             string
	}{
		{
			name:                 "standard pattern",
			managedResourceGroup: "MC_my-rg_my-cluster_eastus",
			expected:             "my-cluster",
		},
		{
			name:                 "resource group with underscores",
			managedResourceGroup: "MC_my_test_rg_cluster-name_westus2",
			expected:             "cluster-name",
		},
		{
			name:                 "complex resource group name",
			managedResourceGroup: "MC_prod_web_services_rg_aks-cluster_centralus",
			expected:             "aks-cluster",
		},
		{
			name:                 "single character components",
			managedResourceGroup: "MC_a_b_c",
			expected:             "b",
		},
		{
			name:                 "actual example from berlioz cluster",
			managedResourceGroup: "MC_azure-cirrus-evan_azure-cirrus-evan_eastus",
			expected:             "azure-cirrus-evan",
		},
		{
			name:                 "invalid format - missing MC prefix",
			managedResourceGroup: "azure-cirrus-evan_azure-cirrus-evan_eastus",
			expected:             "",
		},
		{
			name:                 "invalid format - too few parts",
			managedResourceGroup: "MC_rg_cluster",
			expected:             "",
		},
		{
			name:                 "invalid format - only MC",
			managedResourceGroup: "MC_",
			expected:             "",
		},
		{
			name:                 "empty string",
			managedResourceGroup: "",
			expected:             "",
		},
		{
			name:                 "cluster name too long",
			managedResourceGroup: "MC_rg_" + strings.Repeat("a", 64) + "_eastus",
			expected:             "",
		},
		{
			name:                 "cluster name at length limit",
			managedResourceGroup: "MC_rg_" + strings.Repeat("a", 63) + "_eastus",
			expected:             strings.Repeat("a", 63),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := s.extractClusterName(tt.managedResourceGroup)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper functions

// createScoutWithCustomURL creates a scout for testing with a custom URL
func createScoutWithCustomURL(baseURL string) *Scout {
	return &Scout{
		client: &http.Client{
			Timeout: requestTimeout,
			Transport: &customTransport{
				baseURL: baseURL,
			},
		},
	}
}

// customTransport redirects Azure metadata requests to a test server
type customTransport struct {
	baseURL string
}

func (t *customTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Replace the Azure metadata URL with our test server URL
	newURL := strings.Replace(req.URL.String(), "http://169.254.169.254", t.baseURL, 1)

	newReq := req.Clone(req.Context())
	var err error
	newReq.URL, err = newReq.URL.Parse(newURL)
	if err != nil {
		return nil, err
	}

	return http.DefaultTransport.RoundTrip(newReq)
}

func TestConstants(t *testing.T) {
	// Test that constants are properly defined
	assert.Equal(t, "http://169.254.169.254/metadata/instance?api-version=2021-02-01", azureMetadataURL)
	assert.Equal(t, "Metadata", metadataHeader)
	assert.Equal(t, 63, maxDNSLabelLength)
	assert.Equal(t, 5*time.Second, requestTimeout)
}
