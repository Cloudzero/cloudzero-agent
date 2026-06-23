// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package oracle

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cloudzero/cloudzero-agent/app/utils/scout/types"
	"github.com/stretchr/testify/assert"
)

const (
	// validOCIMetadataResponse is a representative subset of the JSON returned
	// by the OCI Instance Metadata Service v2 (/opc/v2/instance/).
	validOCIMetadataResponse = `{
		"availabilityDomain": "Uocm:US-ASHBURN-AD-1",
		"canonicalRegionName": "us-ashburn-1",
		"region": "iad",
		"compartmentId": "ocid1.tenancy.oc1..aaaaaaaacompartment",
		"tenantId": "ocid1.tenancy.oc1..aaaaaaaatenancy",
		"id": "ocid1.instance.oc1.iad.aaaaaaaainstance",
		"displayName": "oke-test-cirrus-evan-node-0"
	}`

	// phoenixRegionResponse exercises a different region.
	phoenixRegionResponse = `{
		"canonicalRegionName": "us-phoenix-1",
		"region": "phx",
		"tenantId": "ocid1.tenancy.oc1..aaaaaaaatenancy"
	}`
)

func TestNewScout(t *testing.T) {
	scout := NewScout()
	assert.NotNil(t, scout)
	assert.NotNil(t, scout.client)
	assert.Equal(t, requestTimeout, scout.client.Timeout)
}

func TestScout_Detect_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// OCI IMDSv2 requires the Bearer Oracle authorization header.
		assert.Equal(t, authorizationHeaderValue, r.Header.Get(authorizationHeader))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(validOCIMetadataResponse))
	}))
	defer server.Close()

	scout := createScoutWithCustomURL(server.URL)
	result, err := scout.Detect(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, types.CloudProviderOCI, result)
}

func TestScout_Detect_NotOCI(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
	}{
		{name: "not found", statusCode: http.StatusNotFound, body: ""},
		{name: "server error", statusCode: http.StatusInternalServerError, body: ""},
		{name: "unauthorized", statusCode: http.StatusUnauthorized, body: ""},
		{name: "ok but not OCI metadata", statusCode: http.StatusOK, body: `{"foo": "bar"}`},
		{name: "ok but invalid json", statusCode: http.StatusOK, body: `{not json`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				if tt.body != "" {
					_, _ = w.Write([]byte(tt.body))
				}
			}))
			defer server.Close()

			scout := createScoutWithCustomURL(server.URL)
			result, err := scout.Detect(context.Background())

			assert.NoError(t, err)
			assert.Equal(t, types.CloudProviderUnknown, result)
		})
	}
}

func TestScout_Detect_NetworkError(t *testing.T) {
	scout := &Scout{
		client: &http.Client{
			Timeout: requestTimeout,
			Transport: &customTransport{
				baseURL: "http://192.0.2.1", // RFC 5737 TEST-NET-1 - guaranteed unreachable
			},
		},
	}

	result, err := scout.Detect(context.Background())

	// Network errors should not return an error, just unknown provider
	assert.NoError(t, err)
	assert.Equal(t, types.CloudProviderUnknown, result)
}

func TestScout_Detect_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(validOCIMetadataResponse))
	}))
	defer server.Close()

	scout := createScoutWithCustomURL(server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	result, err := scout.Detect(ctx)

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
			name:         "ashburn region",
			responseBody: validOCIMetadataResponse,
			expectedInfo: &types.EnvironmentInfo{
				CloudProvider: types.CloudProviderOCI,
				Region:        "us-ashburn-1",
				// AccountID is intentionally empty: OCI's metadata service exposes
				// the tenancy OCID, not the numeric account ID CloudZero expects, so
				// it must be configured manually. ClusterName is not exposed either
				// (consistent with the AWS scout).
				AccountID:   "",
				ClusterName: "",
			},
		},
		{
			name:         "phoenix region",
			responseBody: phoenixRegionResponse,
			expectedInfo: &types.EnvironmentInfo{
				CloudProvider: types.CloudProviderOCI,
				Region:        "us-phoenix-1",
				AccountID:     "",
				ClusterName:   "",
			},
		},
		{
			name:         "trims whitespace in region",
			responseBody: `{"canonicalRegionName": "  us-ashburn-1  ", "tenantId": "ocid1.tenancy.oc1..aaaa"}`,
			expectedInfo: &types.EnvironmentInfo{
				CloudProvider: types.CloudProviderOCI,
				Region:        "us-ashburn-1",
				AccountID:     "",
				ClusterName:   "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, authorizationHeaderValue, r.Header.Get(authorizationHeader))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			scout := createScoutWithCustomURL(server.URL)
			result, err := scout.EnvironmentInfo(context.Background())

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
			statusCode:    http.StatusInternalServerError,
			responseBody:  "",
			errorContains: "failed to get OCI metadata, status: 500",
		},
		{
			name:          "error on invalid JSON",
			statusCode:    http.StatusOK,
			responseBody:  `{invalid json`,
			errorContains: "failed to parse OCI metadata JSON",
		},
		{
			name:          "error on missing region",
			statusCode:    http.StatusOK,
			responseBody:  `{"tenantId": "ocid1.tenancy.oc1..aaaa"}`,
			errorContains: "canonicalRegionName not found in OCI metadata",
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
			result, err := scout.EnvironmentInfo(context.Background())

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.errorContains)
			assert.Nil(t, result)
		})
	}
}

func TestScout_EnvironmentInfo_NetworkError(t *testing.T) {
	scout := &Scout{
		client: &http.Client{
			Timeout: requestTimeout,
			Transport: &customTransport{
				baseURL: "http://192.0.2.1", // RFC 5737 TEST-NET-1 - guaranteed unreachable
			},
		},
	}

	result, err := scout.EnvironmentInfo(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get OCI metadata")
	assert.Nil(t, result)
}

func TestScout_EnvironmentInfo_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(validOCIMetadataResponse))
	}))
	defer server.Close()

	scout := createScoutWithCustomURL(server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	result, err := scout.EnvironmentInfo(ctx)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
	assert.Nil(t, result)
}

// TestScout_Detect_PositiveSignals confirms Detect fires on ANY single OCI
// signal field, locking in the documented "any positive OCI signal confirms
// OCI" contract (each field is the sole reason detection succeeds).
func TestScout_Detect_PositiveSignals(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "tenantId only", body: `{"tenantId":"ocid1.tenancy.oc1..aaaa"}`},
		{name: "canonicalRegionName only", body: `{"canonicalRegionName":"us-ashburn-1"}`},
		{name: "short region key only", body: `{"region":"iad"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			scout := createScoutWithCustomURL(server.URL)
			result, err := scout.Detect(context.Background())

			assert.NoError(t, err)
			assert.Equal(t, types.CloudProviderOCI, result)
		})
	}
}

// TestScout_EnvironmentInfo_ReadError covers the response-body read-error path,
// mirroring the Azure scout's equivalent test, by hijacking the connection and
// closing it after headers are sent.
func TestScout_EnvironmentInfo_ReadError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
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
	result, err := scout.EnvironmentInfo(context.Background())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read OCI metadata response")
	assert.Nil(t, result)
}

// TestScout_RealCapturedMetadata runs the scout against a REAL OCI IMDSv2
// /opc/v2/instance/ response captured from a live OCI compute instance (us-ashburn-1)
// during CP-40907 end-to-end validation. Resource identifiers (tenancy/compartment/
// instance OCIDs) have been redacted in the fixture. This proves the scout parses an
// actual response — not just hand-written fixtures — and confirms the deliberate empty
// AccountID (OCI's metadata exposes the tenancy OCID, not a numeric account ID).
func TestScout_RealCapturedMetadata(t *testing.T) {
	body, err := os.ReadFile("testdata/real-imds-us-ashburn-1.json")
	if err != nil {
		t.Fatalf("failed to read captured fixture: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, authorizationHeaderValue, r.Header.Get(authorizationHeader))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	defer server.Close()

	scout := createScoutWithCustomURL(server.URL)
	ctx := context.Background()

	provider, err := scout.Detect(ctx)
	assert.NoError(t, err)
	assert.Equal(t, types.CloudProviderOCI, provider)

	info, err := scout.EnvironmentInfo(ctx)
	assert.NoError(t, err)
	assert.Equal(t, &types.EnvironmentInfo{
		CloudProvider: types.CloudProviderOCI,
		Region:        "us-ashburn-1",
		AccountID:     "", // tenantId is the tenancy OCID, not the numeric account ID -> must be configured manually
		ClusterName:   "",
	}, info)
}

func TestConstants(t *testing.T) {
	assert.Equal(t, "http://169.254.169.254/opc/v2/instance/", ociMetadataURL)
	assert.Equal(t, "Authorization", authorizationHeader)
	assert.Equal(t, "Bearer Oracle", authorizationHeaderValue)
	assert.Equal(t, 5*time.Second, requestTimeout)
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

// customTransport redirects OCI metadata requests to a test server
type customTransport struct {
	baseURL string
}

func (t *customTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	newURL := strings.Replace(req.URL.String(), "http://169.254.169.254", t.baseURL, 1)

	newReq := req.Clone(req.Context())
	var err error
	newReq.URL, err = newReq.URL.Parse(newURL)
	if err != nil {
		return nil, err
	}

	return http.DefaultTransport.RoundTrip(newReq)
}
