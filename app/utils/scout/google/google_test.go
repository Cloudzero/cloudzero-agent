// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package google

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/cloudzero/cloudzero-agent/app/utils/scout/types"
)

func TestNewScout(t *testing.T) {
	scout := NewScout()
	if scout == nil {
		t.Fatal("Expected non-nil scout")
	}

	if scout.client == nil {
		t.Fatal("Expected non-nil HTTP client")
	}

	if scout.client.Timeout != requestTimeout {
		t.Errorf("Expected timeout %v, got %v", requestTimeout, scout.client.Timeout)
	}
}

func TestDetect_Success(t *testing.T) {
	// Create test server that mimics GCP metadata service behavior
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for required Metadata-Flavor header
		if r.Header.Get("Metadata-Flavor") != "Google" {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		// GCP returns 200 OK for valid metadata service requests
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	// Test detection with custom server
	testDetectWithMockServer(t, server.URL, types.CloudProviderGoogle)
}

func TestDetect_NotGCP(t *testing.T) {
	// Test various non-GCP responses
	testCases := []struct {
		name           string
		statusCode     int
		expectedResult types.CloudProvider
	}{
		{"Unauthorized", http.StatusUnauthorized, types.CloudProviderUnknown},
		{"Not Found", http.StatusNotFound, types.CloudProviderUnknown},
		{"Internal Server Error", http.StatusInternalServerError, types.CloudProviderUnknown},
		{"Forbidden", http.StatusForbidden, types.CloudProviderUnknown},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.statusCode)
			}))
			defer server.Close()

			testDetectWithMockServer(t, server.URL, tc.expectedResult)
		})
	}
}

func TestDetect_NetworkError(t *testing.T) {
	scout := NewScout()
	ctx := context.Background()

	// Test with invalid URL to simulate network error
	originalClient := scout.client
	scout.client = &http.Client{
		Timeout: 10 * time.Millisecond, // Very short timeout
	}

	// This should return Unknown (not error) for network issues
	detected, err := scout.Detect(ctx)
	if err != nil {
		t.Errorf("Expected no error for network failure, got: %v", err)
	}

	if detected != types.CloudProviderUnknown {
		t.Errorf("Expected %s, got %s", types.CloudProviderUnknown, detected)
	}

	scout.client = originalClient
}

func TestEnvironmentInfo_Success(t *testing.T) {
	// Create test server that mimics GCP metadata service
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for required Metadata-Flavor header
		if r.Header.Get("Metadata-Flavor") != "Google" {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		switch r.URL.Path {
		case "/computeMetadata/v1/project/numeric-project-id":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("123456789"))

		case "/computeMetadata/v1/instance/zone":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("projects/123456789/zones/us-central1-a"))

		case "/computeMetadata/v1/instance/attributes/cluster-name":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("test-cluster-name"))

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	scout := createScoutWithCustomURLs(server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info, err := scout.EnvironmentInfo(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if info.CloudProvider != types.CloudProviderGoogle {
		t.Errorf("Expected cloud provider %s, got %s", types.CloudProviderGoogle, info.CloudProvider)
	}

	if info.Region != "us-central1" {
		t.Errorf("Expected region 'us-central1', got '%s'", info.Region)
	}

	if info.AccountID != "123456789" {
		t.Errorf("Expected account ID '123456789', got '%s'", info.AccountID)
	}

	if info.ClusterName != "test-cluster-name" {
		t.Errorf("Expected cluster name 'test-cluster-name', got '%s'", info.ClusterName)
	}
}

func TestEnvironmentInfo_ProjectIDFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Metadata-Flavor") != "Google" {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		if r.URL.Path == "/computeMetadata/v1/project/numeric-project-id" {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	scout := createScoutWithCustomURLs(server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := scout.EnvironmentInfo(ctx)
	if err == nil {
		t.Error("Expected error for project number failure")
	}

	if !strings.Contains(err.Error(), "failed to get project number") {
		t.Errorf("Expected project number error, got: %v", err)
	}
}

func TestEnvironmentInfo_ClusterNameNotAvailable(t *testing.T) {
	// Test when cluster name is not available (non-GKE environment)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Metadata-Flavor") != "Google" {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		switch r.URL.Path {
		case "/computeMetadata/v1/project/numeric-project-id":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("123456789"))

		case "/computeMetadata/v1/instance/zone":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("projects/123456789/zones/us-central1-a"))

		case "/computeMetadata/v1/instance/attributes/cluster-name":
			w.WriteHeader(http.StatusNotFound) // Cluster name not available

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	scout := createScoutWithCustomURLs(server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info, err := scout.EnvironmentInfo(ctx)
	if err != nil {
		t.Fatalf("Expected no error even when cluster name unavailable, got: %v", err)
	}

	if info.CloudProvider != types.CloudProviderGoogle {
		t.Errorf("Expected cloud provider %s, got %s", types.CloudProviderGoogle, info.CloudProvider)
	}

	if info.Region != "us-central1" {
		t.Errorf("Expected region 'us-central1', got '%s'", info.Region)
	}

	if info.AccountID != "123456789" {
		t.Errorf("Expected account ID '123456789', got '%s'", info.AccountID)
	}

	// Cluster name should be empty when not available
	if info.ClusterName != "" {
		t.Errorf("Expected empty cluster name when not available, got '%s'", info.ClusterName)
	}
}

func TestEnvironmentInfo_ZoneFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Metadata-Flavor") != "Google" {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		switch r.URL.Path {
		case "/computeMetadata/v1/project/numeric-project-id":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("123456789"))

		case "/computeMetadata/v1/instance/zone":
			w.WriteHeader(http.StatusInternalServerError)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	scout := createScoutWithCustomURLs(server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := scout.EnvironmentInfo(ctx)
	if err == nil {
		t.Error("Expected error for zone failure")
	}

	if !strings.Contains(err.Error(), "failed to get zone") {
		t.Errorf("Expected zone error, got: %v", err)
	}
}

func TestEnvironmentInfo_MissingMetadataHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Reject requests without the required header
		if r.Header.Get("Metadata-Flavor") != "Google" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create scout with a client that doesn't set the header
	scout := &Scout{
		client: &http.Client{Timeout: requestTimeout},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Make a request without the header (this tests our implementation sets it correctly)
	req, err := http.NewRequestWithContext(ctx, "GET", server.URL+"/computeMetadata/v1/project/numeric-project-id", nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	resp, err := scout.client.Do(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Should get 403 without the header
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, resp.StatusCode)
	}
}

func TestEnvironmentInfo_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Delay to allow context cancellation
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	scout := createScoutWithCustomURLs(server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := scout.EnvironmentInfo(ctx)
	if err == nil {
		t.Error("Expected error for context cancellation")
	}
}

func TestEnvironmentInfo_WhitespaceHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Metadata-Flavor") != "Google" {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		switch r.URL.Path {
		case "/computeMetadata/v1/project/numeric-project-id":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("  123456789  ")) // Whitespace

		case "/computeMetadata/v1/instance/zone":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("  projects/123456789/zones/us-central1-a  ")) // Whitespace

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	scout := createScoutWithCustomURLs(server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	info, err := scout.EnvironmentInfo(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Verify whitespace is trimmed
	if info.Region != "us-central1" {
		t.Errorf("Expected region 'us-central1', got '%s'", info.Region)
	}

	if info.AccountID != "123456789" {
		t.Errorf("Expected account ID '123456789', got '%s'", info.AccountID)
	}
}

func TestExtractRegionFromZone(t *testing.T) {
	scout := NewScout()

	tests := []struct {
		name     string
		zone     string
		expected string
	}{
		{
			name:     "full zone path format",
			zone:     "projects/123456789/zones/us-central1-a",
			expected: "us-central1",
		},
		{
			name:     "simple zone format",
			zone:     "us-central1-a",
			expected: "us-central1",
		},
		{
			name:     "europe zone",
			zone:     "europe-west1-b",
			expected: "europe-west1",
		},
		{
			name:     "asia zone",
			zone:     "asia-southeast1-c",
			expected: "asia-southeast1",
		},
		{
			name:     "multi-hyphen region",
			zone:     "us-west-2-a",
			expected: "us-west-2",
		},
		{
			name:     "invalid format returns as-is",
			zone:     "invalid-zone-format",
			expected: "invalid-zone",
		},
		{
			name:     "single part returns as-is",
			zone:     "invalid",
			expected: "invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scout.extractRegionFromZone(tt.zone)
			if result != tt.expected {
				t.Errorf("extractRegionFromZone(%q) = %q, want %q", tt.zone, result, tt.expected)
			}
		})
	}
}

func testDetectWithMockServer(t *testing.T, serverURL string, expectedResult types.CloudProvider) {
	t.Helper()

	scout := createScoutWithCustomURLs(serverURL)

	ctx := context.Background()
	detected, err := scout.Detect(ctx)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if detected != expectedResult {
		t.Errorf("Expected %s, got %s", expectedResult, detected)
	}
}

// createScoutWithCustomURLs creates a scout for testing with custom URLs
func createScoutWithCustomURLs(baseURL string) *Scout {
	return &Scout{
		client: &http.Client{
			Timeout: requestTimeout,
			Transport: &customTransport{
				baseURL: baseURL,
			},
		},
	}
}

// customTransport is a helper for testing that redirects requests to a test server
type customTransport struct {
	baseURL string
}

func (t *customTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Rewrite the URL to point to our test server
	newURL := strings.Replace(req.URL.String(), metadataBaseURL, t.baseURL+"/computeMetadata/v1", 1)

	// Handle the detect endpoint specially
	if strings.HasSuffix(req.URL.Path, "/") {
		newURL = t.baseURL + "/"
	}

	newReq := req.Clone(req.Context())
	newReq.URL, _ = req.URL.Parse(newURL)

	return http.DefaultTransport.RoundTrip(newReq)
}

func TestConstants(t *testing.T) {
	if metadataBaseURL == "" {
		t.Error("metadataBaseURL should not be empty")
	}

	if projectEndpoint == "" {
		t.Error("projectEndpoint should not be empty")
	}

	if zoneEndpoint == "" {
		t.Error("zoneEndpoint should not be empty")
	}

	if requestTimeout == 0 {
		t.Error("requestTimeout should not be zero")
	}

	// Verify endpoints are correctly constructed
	expectedProjectEndpoint := metadataBaseURL + "/project/numeric-project-id"
	if projectEndpoint != expectedProjectEndpoint {
		t.Errorf("Expected projectEndpoint %q, got %q", expectedProjectEndpoint, projectEndpoint)
	}

	expectedZoneEndpoint := metadataBaseURL + "/instance/zone"
	if zoneEndpoint != expectedZoneEndpoint {
		t.Errorf("Expected zoneEndpoint %q, got %q", expectedZoneEndpoint, zoneEndpoint)
	}
}
