// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package istio

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cloudzero/cloudzero-agent/app/types/status"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testLogger returns a logger for use in tests
func testLogger() *logrus.Entry {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	return logger.WithField("test", true)
}

// mockAccessor implements status.Accessor for testing
type mockAccessor struct {
	checks []*status.StatusCheck
}

func (m *mockAccessor) AddCheck(checks ...*status.StatusCheck) {
	m.checks = append(m.checks, checks...)
}

func (m *mockAccessor) WriteToReport(fn func(*status.ClusterStatus)) {
	// No-op for tests
}

func (m *mockAccessor) ReadFromReport(fn func(*status.ClusterStatus)) {
	// No-op for tests
}

func (m *mockAccessor) GetCheck(name string) *status.StatusCheck {
	for _, c := range m.checks {
		if c.Name == name {
			return c
		}
	}
	return nil
}

// setupTestRetries configures retry settings for fast tests and returns a cleanup function
func setupTestRetries() func() {
	origMaxRetry := MaxRetry
	origRetryInterval := RetryInterval
	MaxRetry = 1
	RetryInterval = 0
	return func() {
		MaxRetry = origMaxRetry
		RetryInterval = origRetryInterval
	}
}

// mockEnvoyServer creates a test server that responds to both /server_info and /clusters endpoints
func mockEnvoyServer(serverInfoResponse, clustersResponse string, serverInfoStatus, clustersStatus int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/server_info"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(serverInfoStatus)
			w.Write([]byte(serverInfoResponse))
		case strings.HasSuffix(r.URL.Path, "/clusters"):
			w.WriteHeader(clustersStatus)
			w.Write([]byte(clustersResponse))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

// ============================================================================
// Full Check() Flow Tests with Mock Servers
// ============================================================================

func TestCheck_NotInIstioMesh_SidecarUnreachable(t *testing.T) {
	cleanup := setupTestRetries()
	defer cleanup()

	// Create checker with URL pointing to non-existent server
	c := &checker{
		logger:              testLogger(),
		configuredClusterID: "test-cluster",
		aggregatorService:   "aggregator",
		namespace:           "cza",
		serverInfoURL:       "http://localhost:1/server_info", // Non-existent
		clustersURL:         "http://localhost:1/clusters",
	}

	accessor := &mockAccessor{}
	err := c.Check(context.Background(), http.DefaultClient, accessor)
	require.NoError(t, err)

	check := accessor.GetCheck(DiagnosticIstioXClusterLB)
	require.NotNil(t, check)
	assert.True(t, check.Passing, "should pass when sidecar is unreachable")
	assert.Empty(t, check.Error)
}

func TestCheck_InIstioMesh_NoCrossClusterLB_ClusterIDMatch(t *testing.T) {
	cleanup := setupTestRetries()
	defer cleanup()

	serverInfo := `{"node":{"metadata":{"CLUSTER_ID":"my-cluster"}}}`
	clusters := `outbound|8080||aggregator.cza.svc.cluster.local::10.0.1.5:8080::region::us-east-1
outbound|8080||aggregator.cza.svc.cluster.local::10.0.1.6:8080::region::us-east-1`

	server := mockEnvoyServer(serverInfo, clusters, http.StatusOK, http.StatusOK)
	defer server.Close()

	c := &checker{
		logger:              testLogger(),
		configuredClusterID: "my-cluster",
		aggregatorService:   "aggregator",
		namespace:           "cza",
		serverInfoURL:       server.URL + "/server_info",
		clustersURL:         server.URL + "/clusters",
	}

	accessor := &mockAccessor{}
	err := c.Check(context.Background(), http.DefaultClient, accessor)
	require.NoError(t, err)

	check := accessor.GetCheck(DiagnosticIstioXClusterLB)
	require.NotNil(t, check)
	assert.True(t, check.Passing, "should pass when in Istio mesh with matching cluster ID and no cross-cluster LB")
	assert.Empty(t, check.Error)
}

func TestCheck_InIstioMesh_ClusterIDMismatch(t *testing.T) {
	cleanup := setupTestRetries()
	defer cleanup()

	// Istio reports "gke-cluster" but we configured "foo"
	serverInfo := `{"node":{"metadata":{"CLUSTER_ID":"gke-cluster"}}}`

	server := mockEnvoyServer(serverInfo, "", http.StatusOK, http.StatusOK)
	defer server.Close()

	c := &checker{
		logger:              testLogger(),
		configuredClusterID: "foo", // Mismatch!
		aggregatorService:   "aggregator",
		namespace:           "cza",
		serverInfoURL:       server.URL + "/server_info",
		clustersURL:         server.URL + "/clusters",
	}

	accessor := &mockAccessor{}
	err := c.Check(context.Background(), http.DefaultClient, accessor)
	require.NoError(t, err)

	check := accessor.GetCheck(DiagnosticIstioXClusterLB)
	require.NotNil(t, check)
	assert.False(t, check.Passing, "should fail when cluster ID mismatches")
	assert.Contains(t, check.Error, "does not match Istio cluster ID")
	assert.Contains(t, check.Error, "foo")
	assert.Contains(t, check.Error, "gke-cluster")
}

func TestCheck_InIstioMesh_CrossClusterLB_WithClusterID(t *testing.T) {
	cleanup := setupTestRetries()
	defer cleanup()

	serverInfo := `{"node":{"metadata":{"CLUSTER_ID":"my-cluster"}}}`
	// Multiple regions = cross-cluster LB detected
	clusters := `outbound|8080||aggregator.cza.svc.cluster.local::10.0.1.5:8080::region::us-east-1
outbound|8080||aggregator.cza.svc.cluster.local::10.0.2.10:8080::region::us-west-2`

	server := mockEnvoyServer(serverInfo, clusters, http.StatusOK, http.StatusOK)
	defer server.Close()

	c := &checker{
		logger:              testLogger(),
		configuredClusterID: "my-cluster",
		aggregatorService:   "aggregator",
		namespace:           "cza",
		serverInfoURL:       server.URL + "/server_info",
		clustersURL:         server.URL + "/clusters",
	}

	accessor := &mockAccessor{}
	err := c.Check(context.Background(), http.DefaultClient, accessor)
	require.NoError(t, err)

	check := accessor.GetCheck(DiagnosticIstioXClusterLB)
	require.NotNil(t, check)
	assert.True(t, check.Passing, "should pass when cross-cluster LB detected but cluster ID is configured")
	assert.Empty(t, check.Error)
}

func TestCheck_InIstioMesh_EffectiveClusterIDMismatch(t *testing.T) {
	// Test: clusterID not explicitly set, clusterName (fallback) doesn't match Istio
	// This tests the scenario where the Helm chart uses clusterName as fallback
	// but it doesn't match the Istio cluster ID
	cleanup := setupTestRetries()
	defer cleanup()

	serverInfo := `{"node":{"metadata":{"CLUSTER_ID":"istio-cluster"}}}`
	clusters := `outbound|8080||aggregator.cza.svc.cluster.local::10.0.1.5:8080::region::us-east-1`

	server := mockEnvoyServer(serverInfo, clusters, http.StatusOK, http.StatusOK)
	defer server.Close()

	c := &checker{
		logger:              testLogger(),
		configuredClusterID: "",            // Not explicitly configured
		clusterName:         "eks-cluster", // This is what Helm would use as fallback
		aggregatorService:   "aggregator",
		namespace:           "cza",
		serverInfoURL:       server.URL + "/server_info",
		clustersURL:         server.URL + "/clusters",
	}

	accessor := &mockAccessor{}
	err := c.Check(context.Background(), http.DefaultClient, accessor)
	require.NoError(t, err)

	check := accessor.GetCheck(DiagnosticIstioXClusterLB)
	require.NotNil(t, check)
	assert.False(t, check.Passing, "should fail when effective cluster ID doesn't match Istio")
	assert.Contains(t, check.Error, "Effective cluster ID 'eks-cluster' does not match Istio cluster ID 'istio-cluster'")
}

func TestCheck_InIstioMesh_NoClusterIDInSidecar(t *testing.T) {
	cleanup := setupTestRetries()
	defer cleanup()

	// Sidecar responds but has no CLUSTER_ID in metadata
	serverInfo := `{"node":{"metadata":{"NAMESPACE":"cza"}}}`
	clusters := `outbound|8080||aggregator.cza.svc.cluster.local::10.0.1.5:8080::region::us-east-1`

	server := mockEnvoyServer(serverInfo, clusters, http.StatusOK, http.StatusOK)
	defer server.Close()

	c := &checker{
		logger:              testLogger(),
		configuredClusterID: "my-cluster",
		aggregatorService:   "aggregator",
		namespace:           "cza",
		serverInfoURL:       server.URL + "/server_info",
		clustersURL:         server.URL + "/clusters",
	}

	accessor := &mockAccessor{}
	err := c.Check(context.Background(), http.DefaultClient, accessor)
	require.NoError(t, err)

	check := accessor.GetCheck(DiagnosticIstioXClusterLB)
	require.NotNil(t, check)
	// When Istio cluster ID is empty, we skip the mismatch check
	assert.True(t, check.Passing, "should pass when sidecar has no cluster ID")
}

func TestCheck_InIstioMesh_ClustersEndpointFails(t *testing.T) {
	cleanup := setupTestRetries()
	defer cleanup()

	serverInfo := `{"node":{"metadata":{"CLUSTER_ID":"my-cluster"}}}`

	server := mockEnvoyServer(serverInfo, "", http.StatusOK, http.StatusInternalServerError)
	defer server.Close()

	c := &checker{
		logger:              testLogger(),
		configuredClusterID: "my-cluster",
		aggregatorService:   "aggregator",
		namespace:           "cza",
		serverInfoURL:       server.URL + "/server_info",
		clustersURL:         server.URL + "/clusters",
	}

	accessor := &mockAccessor{}
	err := c.Check(context.Background(), http.DefaultClient, accessor)
	require.NoError(t, err)

	check := accessor.GetCheck(DiagnosticIstioXClusterLB)
	require.NotNil(t, check)
	// When clusters endpoint fails, we pass (can't detect cross-cluster LB)
	assert.True(t, check.Passing, "should pass when clusters endpoint fails")
}

func TestCheck_InIstioMesh_ServerInfoReturns500(t *testing.T) {
	cleanup := setupTestRetries()
	defer cleanup()

	server := mockEnvoyServer("", "", http.StatusInternalServerError, http.StatusOK)
	defer server.Close()

	c := &checker{
		logger:              testLogger(),
		configuredClusterID: "my-cluster",
		aggregatorService:   "aggregator",
		namespace:           "cza",
		serverInfoURL:       server.URL + "/server_info",
		clustersURL:         server.URL + "/clusters",
	}

	accessor := &mockAccessor{}
	err := c.Check(context.Background(), http.DefaultClient, accessor)
	require.NoError(t, err)

	check := accessor.GetCheck(DiagnosticIstioXClusterLB)
	require.NotNil(t, check)
	// When server_info fails, we treat it as "not in Istio mesh"
	assert.True(t, check.Passing, "should pass when server_info returns error")
}

// ============================================================================
// parseEnvoyResponse Unit Tests
// ============================================================================

func TestParseEnvoyResponse_SingleRegion(t *testing.T) {
	envoyResponse := `outbound|8080||aggregator.cza.svc.cluster.local::10.0.1.5:8080::region::us-east-1
outbound|8080||aggregator.cza.svc.cluster.local::10.0.1.5:8080::zone::us-east-1a
outbound|8080||aggregator.cza.svc.cluster.local::10.0.1.6:8080::region::us-east-1
outbound|8080||aggregator.cza.svc.cluster.local::10.0.1.6:8080::zone::us-east-1b`

	c := &checker{
		logger:            testLogger(),
		aggregatorService: "aggregator",
		namespace:         "cza",
	}

	assert.False(t, c.parseEnvoyResponse(envoyResponse), "should not detect cross-cluster LB with single region")
}

func TestParseEnvoyResponse_MultipleRegions(t *testing.T) {
	envoyResponse := `outbound|8080||aggregator.cza.svc.cluster.local::10.0.1.5:8080::region::us-east-1
outbound|8080||aggregator.cza.svc.cluster.local::10.0.2.10:8080::region::us-west-2`

	c := &checker{
		logger:            testLogger(),
		aggregatorService: "aggregator",
		namespace:         "cza",
	}

	assert.True(t, c.parseEnvoyResponse(envoyResponse), "should detect cross-cluster LB with multiple regions")
}

func TestParseEnvoyResponse_NoServiceMatch(t *testing.T) {
	envoyResponse := `outbound|8080||other-service.other-ns.svc.cluster.local::10.0.1.5:8080::region::us-east-1
outbound|8080||other-service.other-ns.svc.cluster.local::10.0.2.10:8080::region::us-west-2`

	c := &checker{
		logger:            testLogger(),
		aggregatorService: "aggregator",
		namespace:         "cza",
	}

	assert.False(t, c.parseEnvoyResponse(envoyResponse), "should not detect cross-cluster LB for non-matching service")
}

func TestParseEnvoyResponse_NoRegionInfo(t *testing.T) {
	envoyResponse := `outbound|8080||aggregator.cza.svc.cluster.local::10.0.1.5:8080::health_flags::healthy
outbound|8080||aggregator.cza.svc.cluster.local::10.0.1.6:8080::health_flags::healthy`

	c := &checker{
		logger:            testLogger(),
		aggregatorService: "aggregator",
		namespace:         "cza",
	}

	assert.False(t, c.parseEnvoyResponse(envoyResponse), "should not detect cross-cluster LB without region info")
}

func TestParseEnvoyResponse_EmptyResponse(t *testing.T) {
	c := &checker{
		logger:            testLogger(),
		aggregatorService: "aggregator",
		namespace:         "cza",
	}

	assert.False(t, c.parseEnvoyResponse(""), "should not detect cross-cluster LB with empty response")
}

func TestParseEnvoyResponse_MixedServices(t *testing.T) {
	// Other services have multiple regions, but aggregator only has one
	envoyResponse := `outbound|8080||other-service.cza.svc.cluster.local::10.0.1.5:8080::region::us-east-1
outbound|8080||other-service.cza.svc.cluster.local::10.0.2.10:8080::region::us-west-2
outbound|8080||aggregator.cza.svc.cluster.local::10.0.1.5:8080::region::us-east-1
outbound|8080||aggregator.cza.svc.cluster.local::10.0.1.6:8080::region::us-east-1`

	c := &checker{
		logger:            testLogger(),
		aggregatorService: "aggregator",
		namespace:         "cza",
	}

	assert.False(t, c.parseEnvoyResponse(envoyResponse), "should only consider aggregator service")
}

// ============================================================================
// serverInfoResponse Parsing Tests
// ============================================================================

func TestServerInfoResponse_WithClusterID(t *testing.T) {
	serverInfoJSON := `{
		"node": {
			"metadata": {
				"CLUSTER_ID": "gke-cluster",
				"NAMESPACE": "cza"
			}
		}
	}`

	var info serverInfoResponse
	err := json.Unmarshal([]byte(serverInfoJSON), &info)
	require.NoError(t, err)

	clusterID, ok := info.Node.Metadata["CLUSTER_ID"].(string)
	assert.True(t, ok, "should find CLUSTER_ID in metadata")
	assert.Equal(t, "gke-cluster", clusterID)
}

func TestServerInfoResponse_WithoutClusterID(t *testing.T) {
	serverInfoJSON := `{
		"node": {
			"metadata": {
				"NAMESPACE": "cza"
			}
		}
	}`

	var info serverInfoResponse
	err := json.Unmarshal([]byte(serverInfoJSON), &info)
	require.NoError(t, err)

	clusterID, ok := info.Node.Metadata["CLUSTER_ID"].(string)
	assert.False(t, ok || clusterID != "", "should not find CLUSTER_ID in metadata")
}

func TestServerInfoResponse_EmptyMetadata(t *testing.T) {
	serverInfoJSON := `{
		"node": {
			"metadata": {}
		}
	}`

	var info serverInfoResponse
	err := json.Unmarshal([]byte(serverInfoJSON), &info)
	require.NoError(t, err)

	clusterID, ok := info.Node.Metadata["CLUSTER_ID"].(string)
	assert.False(t, ok || clusterID != "", "should not find CLUSTER_ID in empty metadata")
}

// ============================================================================
// getIstioClusterID Method Tests
// ============================================================================

func TestGetIstioClusterID_Success(t *testing.T) {
	cleanup := setupTestRetries()
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"node":{"metadata":{"CLUSTER_ID":"test-cluster"}}}`))
	}))
	defer server.Close()

	c := &checker{
		logger:        testLogger(),
		serverInfoURL: server.URL,
	}

	clusterID, err := c.getIstioClusterID(context.Background(), http.DefaultClient)
	require.NoError(t, err)
	assert.Equal(t, "test-cluster", clusterID)
}

func TestGetIstioClusterID_NoClusterID(t *testing.T) {
	cleanup := setupTestRetries()
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"node":{"metadata":{"NAMESPACE":"cza"}}}`))
	}))
	defer server.Close()

	c := &checker{
		logger:        testLogger(),
		serverInfoURL: server.URL,
	}

	clusterID, err := c.getIstioClusterID(context.Background(), http.DefaultClient)
	require.NoError(t, err)
	assert.Empty(t, clusterID, "should return empty when no CLUSTER_ID in metadata")
}

func TestGetIstioClusterID_ServerError(t *testing.T) {
	cleanup := setupTestRetries()
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	c := &checker{
		logger:        testLogger(),
		serverInfoURL: server.URL,
	}

	_, err := c.getIstioClusterID(context.Background(), http.DefaultClient)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}

func TestGetIstioClusterID_InvalidJSON(t *testing.T) {
	cleanup := setupTestRetries()
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not valid json`))
	}))
	defer server.Close()

	c := &checker{
		logger:        testLogger(),
		serverInfoURL: server.URL,
	}

	_, err := c.getIstioClusterID(context.Background(), http.DefaultClient)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse")
}

// ============================================================================
// detectCrossClusterLB Method Tests
// ============================================================================

func TestDetectCrossClusterLB_Success(t *testing.T) {
	cleanup := setupTestRetries()
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`outbound|8080||aggregator.cza.svc.cluster.local::10.0.1.5:8080::region::us-east-1
outbound|8080||aggregator.cza.svc.cluster.local::10.0.2.10:8080::region::us-west-2`))
	}))
	defer server.Close()

	c := &checker{
		logger:            testLogger(),
		aggregatorService: "aggregator",
		namespace:         "cza",
		clustersURL:       server.URL,
	}

	detected, err := c.detectCrossClusterLB(context.Background(), http.DefaultClient)
	require.NoError(t, err)
	assert.True(t, detected, "should detect cross-cluster LB")
}

func TestDetectCrossClusterLB_ServerError(t *testing.T) {
	cleanup := setupTestRetries()
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	c := &checker{
		logger:            testLogger(),
		aggregatorService: "aggregator",
		namespace:         "cza",
		clustersURL:       server.URL,
	}

	_, err := c.detectCrossClusterLB(context.Background(), http.DefaultClient)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}

// ============================================================================
// detectIstioMode Tests
// ============================================================================

func TestDetectIstioMode_Sidecar(t *testing.T) {
	cleanup := setupTestRetries()
	defer cleanup()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"node":{"metadata":{"CLUSTER_ID":"test-cluster"}}}`))
	}))
	defer server.Close()

	c := &checker{
		logger:        testLogger(),
		serverInfoURL: server.URL,
	}

	mode := c.detectIstioMode(context.Background(), http.DefaultClient)
	assert.Equal(t, IstioModeSidecar, mode, "should detect sidecar mode when localhost:15000 is reachable")
}

func TestDetectIstioMode_Ambient(t *testing.T) {
	cleanup := setupTestRetries()
	defer cleanup()

	// Set the ambient env var
	t.Setenv(envIstioAmbientRedirection, istioAmbientRedirectionEnabled)

	// Point to non-existent server (sidecar not available)
	c := &checker{
		logger:        testLogger(),
		serverInfoURL: "http://localhost:1/server_info",
	}

	mode := c.detectIstioMode(context.Background(), http.DefaultClient)
	assert.Equal(t, IstioModeAmbient, mode, "should detect ambient mode when env var is set and sidecar unreachable")
}

func TestDetectIstioMode_None(t *testing.T) {
	cleanup := setupTestRetries()
	defer cleanup()

	// Ensure ambient env var is not set (t.Setenv auto-cleans up)
	t.Setenv(envIstioAmbientRedirection, "")

	// Point to non-existent server (sidecar not available)
	c := &checker{
		logger:        testLogger(),
		serverInfoURL: "http://localhost:1/server_info",
	}

	mode := c.detectIstioMode(context.Background(), http.DefaultClient)
	assert.Equal(t, IstioModeNone, mode, "should detect no Istio when sidecar unreachable and no ambient env var")
}

func TestDetectIstioMode_SidecarTakesPrecedenceOverAmbient(t *testing.T) {
	cleanup := setupTestRetries()
	defer cleanup()

	// Set the ambient env var (shouldn't matter if sidecar is reachable)
	t.Setenv(envIstioAmbientRedirection, istioAmbientRedirectionEnabled)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"node":{"metadata":{"CLUSTER_ID":"test-cluster"}}}`))
	}))
	defer server.Close()

	c := &checker{
		logger:        testLogger(),
		serverInfoURL: server.URL,
	}

	mode := c.detectIstioMode(context.Background(), http.DefaultClient)
	assert.Equal(t, IstioModeSidecar, mode, "sidecar mode should take precedence over ambient when both indicators present")
}

// ============================================================================
// Ambient Mode Check Tests
// ============================================================================

func TestCheck_AmbientMode_WithClusterID(t *testing.T) {
	cleanup := setupTestRetries()
	defer cleanup()

	// Set ambient mode
	t.Setenv(envIstioAmbientRedirection, istioAmbientRedirectionEnabled)

	// Point to non-existent server (no sidecar in ambient mode)
	c := &checker{
		logger:              testLogger(),
		configuredClusterID: "my-cluster",
		aggregatorService:   "aggregator",
		namespace:           "cza",
		serverInfoURL:       "http://localhost:1/server_info",
		clustersURL:         "http://localhost:1/clusters",
	}

	accessor := &mockAccessor{}
	err := c.Check(context.Background(), http.DefaultClient, accessor)
	require.NoError(t, err)

	check := accessor.GetCheck(DiagnosticIstioXClusterLB)
	require.NotNil(t, check)
	assert.True(t, check.Passing, "should pass in ambient mode with cluster ID configured")
	assert.Empty(t, check.Error)
}

func TestCheck_AmbientMode_WithoutClusterID(t *testing.T) {
	cleanup := setupTestRetries()
	defer cleanup()

	// Set ambient mode
	t.Setenv(envIstioAmbientRedirection, istioAmbientRedirectionEnabled)

	// Point to non-existent server (no sidecar in ambient mode)
	c := &checker{
		logger:              testLogger(),
		configuredClusterID: "", // Not configured!
		aggregatorService:   "aggregator",
		namespace:           "cza",
		serverInfoURL:       "http://localhost:1/server_info",
		clustersURL:         "http://localhost:1/clusters",
	}

	accessor := &mockAccessor{}
	err := c.Check(context.Background(), http.DefaultClient, accessor)
	require.NoError(t, err)

	check := accessor.GetCheck(DiagnosticIstioXClusterLB)
	require.NotNil(t, check)
	assert.False(t, check.Passing, "should fail in ambient mode without cluster ID")
	assert.Contains(t, check.Error, "Ambient mode detected")
	assert.Contains(t, check.Error, "clusterID not configured")
}

func TestCheckAmbientMode_TrustsConfiguredClusterID(t *testing.T) {
	c := &checker{
		logger:              testLogger(),
		configuredClusterID: "trusted-cluster-id",
	}

	accessor := &mockAccessor{}
	err := c.checkAmbientMode(context.Background(), accessor)
	require.NoError(t, err)

	check := accessor.GetCheck(DiagnosticIstioXClusterLB)
	require.NotNil(t, check)
	assert.True(t, check.Passing, "should pass when cluster ID is configured")
	assert.Empty(t, check.Error)
}

func TestCheckAmbientMode_FailsWithoutClusterID(t *testing.T) {
	c := &checker{
		logger:              testLogger(),
		configuredClusterID: "",
	}

	accessor := &mockAccessor{}
	err := c.checkAmbientMode(context.Background(), accessor)
	require.NoError(t, err)

	check := accessor.GetCheck(DiagnosticIstioXClusterLB)
	require.NotNil(t, check)
	assert.False(t, check.Passing, "should fail when cluster ID is not configured")
	assert.Contains(t, check.Error, "Ambient mode")
}

// ============================================================================
// effectiveClusterID Helper Tests
// ============================================================================

func TestEffectiveClusterID_ReturnsConfiguredWhenSet(t *testing.T) {
	c := &checker{
		configuredClusterID: "explicit-cluster",
		clusterName:         "fallback-cluster",
	}

	assert.Equal(t, "explicit-cluster", c.effectiveClusterID(),
		"should return configured cluster ID when explicitly set")
}

func TestEffectiveClusterID_FallsBackToClusterName(t *testing.T) {
	c := &checker{
		configuredClusterID: "", // Not set
		clusterName:         "my-eks-cluster",
	}

	assert.Equal(t, "my-eks-cluster", c.effectiveClusterID(),
		"should fall back to clusterName when configuredClusterID is empty")
}

func TestEffectiveClusterID_ReturnsEmptyWhenBothEmpty(t *testing.T) {
	c := &checker{
		configuredClusterID: "",
		clusterName:         "",
	}

	assert.Equal(t, "", c.effectiveClusterID(),
		"should return empty when both configuredClusterID and clusterName are empty")
}

// ============================================================================
// Topology Label Validation Tests
// ============================================================================

func TestCheckAmbientMode_TopologyLabelMatch(t *testing.T) {
	c := &checker{
		logger:              testLogger(),
		configuredClusterID: "my-cluster",
		topologyCluster:     "my-cluster", // Matches!
	}

	accessor := &mockAccessor{}
	err := c.checkAmbientMode(context.Background(), accessor)
	require.NoError(t, err)

	check := accessor.GetCheck(DiagnosticIstioXClusterLB)
	require.NotNil(t, check)
	assert.True(t, check.Passing, "should pass when topology label matches configured cluster ID")
	assert.Empty(t, check.Error)
}

func TestCheckAmbientMode_TopologyLabelMismatch(t *testing.T) {
	c := &checker{
		logger:              testLogger(),
		configuredClusterID: "configured-cluster",
		topologyCluster:     "actual-istio-cluster", // Mismatch!
	}

	accessor := &mockAccessor{}
	err := c.checkAmbientMode(context.Background(), accessor)
	require.NoError(t, err)

	check := accessor.GetCheck(DiagnosticIstioXClusterLB)
	require.NotNil(t, check)
	assert.False(t, check.Passing, "should fail when topology label doesn't match configured cluster ID")
	assert.Contains(t, check.Error, "configured-cluster")
	assert.Contains(t, check.Error, "actual-istio-cluster")
	assert.Contains(t, check.Error, "does not match pod topology label")
}

func TestCheckAmbientMode_TopologyLabelEmpty(t *testing.T) {
	c := &checker{
		logger:              testLogger(),
		configuredClusterID: "my-cluster",
		topologyCluster:     "", // Not available
	}

	accessor := &mockAccessor{}
	err := c.checkAmbientMode(context.Background(), accessor)
	require.NoError(t, err)

	check := accessor.GetCheck(DiagnosticIstioXClusterLB)
	require.NotNil(t, check)
	assert.True(t, check.Passing, "should pass when topology label is not available (can't validate)")
	assert.Empty(t, check.Error)
}

func TestCheckSidecarMode_TopologyLabelMismatch(t *testing.T) {
	cleanup := setupTestRetries()
	defer cleanup()

	serverInfo := `{"node":{"metadata":{"CLUSTER_ID":"my-cluster"}}}`
	clusters := `outbound|8080||aggregator.cza.svc.cluster.local::10.0.1.5:8080::region::us-east-1`

	server := mockEnvoyServer(serverInfo, clusters, http.StatusOK, http.StatusOK)
	defer server.Close()

	c := &checker{
		logger:              testLogger(),
		configuredClusterID: "my-cluster",
		clusterName:         "my-cluster",
		topologyCluster:     "different-cluster", // Mismatch with effective!
		aggregatorService:   "aggregator",
		namespace:           "cza",
		serverInfoURL:       server.URL + "/server_info",
		clustersURL:         server.URL + "/clusters",
	}

	accessor := &mockAccessor{}
	err := c.checkSidecarMode(context.Background(), http.DefaultClient, accessor)
	require.NoError(t, err)

	check := accessor.GetCheck(DiagnosticIstioXClusterLB)
	require.NotNil(t, check)
	assert.False(t, check.Passing, "should fail when topology label doesn't match effective cluster ID")
	assert.Contains(t, check.Error, "my-cluster")
	assert.Contains(t, check.Error, "different-cluster")
}

func TestCheckSidecarMode_TopologyLabelMatch(t *testing.T) {
	cleanup := setupTestRetries()
	defer cleanup()

	serverInfo := `{"node":{"metadata":{"CLUSTER_ID":"my-cluster"}}}`
	clusters := `outbound|8080||aggregator.cza.svc.cluster.local::10.0.1.5:8080::region::us-east-1`

	server := mockEnvoyServer(serverInfo, clusters, http.StatusOK, http.StatusOK)
	defer server.Close()

	c := &checker{
		logger:              testLogger(),
		configuredClusterID: "my-cluster",
		clusterName:         "my-cluster",
		topologyCluster:     "my-cluster", // Matches!
		aggregatorService:   "aggregator",
		namespace:           "cza",
		serverInfoURL:       server.URL + "/server_info",
		clustersURL:         server.URL + "/clusters",
	}

	accessor := &mockAccessor{}
	err := c.checkSidecarMode(context.Background(), http.DefaultClient, accessor)
	require.NoError(t, err)

	check := accessor.GetCheck(DiagnosticIstioXClusterLB)
	require.NotNil(t, check)
	assert.True(t, check.Passing, "should pass when topology label matches effective cluster ID")
	assert.Empty(t, check.Error)
}

// ============================================================================
// Sidecar Mode with clusterName Fallback Tests
// ============================================================================

func TestCheckSidecarMode_ClusterNameFallbackMatch(t *testing.T) {
	// Test: clusterID not explicitly set, clusterName fallback matches Istio
	cleanup := setupTestRetries()
	defer cleanup()

	serverInfo := `{"node":{"metadata":{"CLUSTER_ID":"my-eks-cluster"}}}`
	clusters := `outbound|8080||aggregator.cza.svc.cluster.local::10.0.1.5:8080::region::us-east-1`

	server := mockEnvoyServer(serverInfo, clusters, http.StatusOK, http.StatusOK)
	defer server.Close()

	c := &checker{
		logger:              testLogger(),
		configuredClusterID: "",               // Not explicitly set
		clusterName:         "my-eks-cluster", // Fallback matches Istio!
		aggregatorService:   "aggregator",
		namespace:           "cza",
		serverInfoURL:       server.URL + "/server_info",
		clustersURL:         server.URL + "/clusters",
	}

	accessor := &mockAccessor{}
	err := c.checkSidecarMode(context.Background(), http.DefaultClient, accessor)
	require.NoError(t, err)

	check := accessor.GetCheck(DiagnosticIstioXClusterLB)
	require.NotNil(t, check)
	assert.True(t, check.Passing, "should pass when clusterName fallback matches Istio cluster ID")
	assert.Empty(t, check.Error)
}

func TestCheckSidecarMode_ExplicitClusterIDOverridesClusterName(t *testing.T) {
	// Test: explicit clusterID takes precedence over clusterName
	cleanup := setupTestRetries()
	defer cleanup()

	serverInfo := `{"node":{"metadata":{"CLUSTER_ID":"istio-cluster"}}}`
	clusters := `outbound|8080||aggregator.cza.svc.cluster.local::10.0.1.5:8080::region::us-east-1`

	server := mockEnvoyServer(serverInfo, clusters, http.StatusOK, http.StatusOK)
	defer server.Close()

	c := &checker{
		logger:              testLogger(),
		configuredClusterID: "istio-cluster",  // Explicitly set, takes precedence
		clusterName:         "different-name", // Would cause mismatch if used
		aggregatorService:   "aggregator",
		namespace:           "cza",
		serverInfoURL:       server.URL + "/server_info",
		clustersURL:         server.URL + "/clusters",
	}

	accessor := &mockAccessor{}
	err := c.checkSidecarMode(context.Background(), http.DefaultClient, accessor)
	require.NoError(t, err)

	check := accessor.GetCheck(DiagnosticIstioXClusterLB)
	require.NotNil(t, check)
	assert.True(t, check.Passing, "should pass when explicit clusterID matches Istio (ignoring clusterName)")
	assert.Empty(t, check.Error)
}
