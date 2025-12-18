// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package istio provides diagnostics for detecting Istio service mesh configuration,
// cross-cluster load balancing, and validating cluster ID settings.
package istio

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	config "github.com/cloudzero/cloudzero-agent/app/config/validator"
	"github.com/cloudzero/cloudzero-agent/app/domain/diagnostic"
	logging "github.com/cloudzero/cloudzero-agent/app/logging/validator"
	"github.com/cloudzero/cloudzero-agent/app/types/status"
	"github.com/sirupsen/logrus"
)

const (
	DiagnosticIstioXClusterLB = config.DiagnosticIstioXClusterLB

	// envoyAdminClustersURL is the Envoy admin API endpoint for cluster information
	envoyAdminClustersURL = "http://localhost:15000/clusters"

	// envoyAdminServerInfoURL is the Envoy admin API endpoint for server info (includes cluster ID)
	envoyAdminServerInfoURL = "http://localhost:15000/server_info"

	// logRetryAttempt is the format string for retry attempt logging
	logRetryAttempt = "Attempt %d: %v"

	// envIstioAmbientRedirection is the env var set via Downward API for ambient mode detection
	envIstioAmbientRedirection = "ISTIO_AMBIENT_REDIRECTION"

	// istioAmbientRedirectionEnabled is the value indicating ambient mode is active
	istioAmbientRedirectionEnabled = "enabled"

	// envIstioTopologyCluster is the env var set via Downward API for topology label validation
	envIstioTopologyCluster = "ISTIO_TOPOLOGY_CLUSTER"
)

// IstioMode represents the detected Istio service mesh mode
type IstioMode string

const (
	// IstioModeNone indicates no Istio service mesh detected
	IstioModeNone IstioMode = "none"
	// IstioModeSidecar indicates traditional sidecar proxy mode
	IstioModeSidecar IstioMode = "sidecar"
	// IstioModeAmbient indicates ambient mode (sidecarless)
	IstioModeAmbient IstioMode = "ambient"
)

var (
	// Exported for testing
	MaxRetry      = 3
	RetryInterval = 5 * time.Second
)

// checker implements the diagnostic.Provider interface for Istio cross-cluster LB detection
type checker struct {
	cfg                 *config.Settings
	logger              *logrus.Entry
	configuredClusterID string // From Helm values (integrations.istio.clusterID) - explicit only
	clusterName         string // From deployment.cluster_name - used for fallback
	topologyCluster     string // From Downward API label (topology.istio.io/cluster)
	aggregatorService   string // Service name to look for in Envoy clusters
	namespace           string // Namespace where aggregator runs

	// URLs for Envoy admin API endpoints (configurable for testing)
	serverInfoURL string
	clustersURL   string
}

// NewProvider creates a new Istio cross-cluster LB diagnostic provider
var NewProvider = func(ctx context.Context, cfg *config.Settings) diagnostic.Provider {
	return &checker{
		cfg: cfg,
		logger: logging.NewLogger().
			WithContext(ctx).WithField(logging.OpField, "istio-xcluster"),
		configuredClusterID: cfg.Integrations.Istio.ClusterID,
		clusterName:         cfg.Deployment.ClusterName,
		topologyCluster:     os.Getenv(envIstioTopologyCluster),
		aggregatorService:   cfg.Services.CollectorService,
		namespace:           cfg.Services.Namespace,
		serverInfoURL:       envoyAdminServerInfoURL,
		clustersURL:         envoyAdminClustersURL,
	}
}

// effectiveClusterID returns the cluster ID that Helm uses for DestinationRule.
// This matches the Helm template: {{ .Values.integrations.istio.clusterID | default .Values.clusterName }}
func (c *checker) effectiveClusterID() string {
	if c.configuredClusterID != "" {
		return c.configuredClusterID
	}
	return c.clusterName
}

// detectIstioMode determines the Istio service mesh mode by checking:
// 1. Sidecar mode: localhost:15000 (Envoy admin API) is reachable
// 2. Ambient mode: ISTIO_AMBIENT_REDIRECTION env var is "enabled" (set via Downward API)
// 3. None: Neither indicator present
func (c *checker) detectIstioMode(ctx context.Context, client *http.Client) IstioMode {
	// Try sidecar detection first - if we can reach localhost:15000, we have a sidecar
	if _, err := c.getIstioClusterID(ctx, client); err == nil {
		return IstioModeSidecar
	}

	// Check for ambient mode via Downward API env var
	if os.Getenv(envIstioAmbientRedirection) == istioAmbientRedirectionEnabled {
		return IstioModeAmbient
	}

	return IstioModeNone
}

// Check performs the Istio cross-cluster load balancing detection and validation.
// It detects Istio mode (Sidecar, Ambient, or None) and validates configuration:
//
// - None: PASS - not running in an Istio mesh
// - Sidecar: Full validation via localhost:15000 (cluster ID match, cross-cluster LB detection)
// - Ambient: PASS - trust configured cluster ID (no local proxy to validate against)
//
// In Ambient mode, traffic fencing relies on the DestinationRule configured at deploy time.
// Runtime validation is not possible because there's no local sidecar to query.
func (c *checker) Check(ctx context.Context, client *http.Client, accessor status.Accessor) error {
	c.logger.Infof("Configured cluster ID: '%s', cluster name: '%s', effective: '%s'",
		c.configuredClusterID, c.clusterName, c.effectiveClusterID())

	// Detect Istio mode
	mode := c.detectIstioMode(ctx, client)
	c.logger.Infof("Detected Istio mode: %s", mode)

	switch mode {
	case IstioModeNone:
		c.logger.Info("Not running in Istio mesh")
		accessor.AddCheck(&status.StatusCheck{
			Name:    DiagnosticIstioXClusterLB,
			Passing: true,
		})
		return nil

	case IstioModeAmbient:
		return c.checkAmbientMode(ctx, accessor)

	case IstioModeSidecar:
		return c.checkSidecarMode(ctx, client, accessor)
	}

	// Should not reach here, but handle gracefully
	accessor.AddCheck(&status.StatusCheck{
		Name:    DiagnosticIstioXClusterLB,
		Passing: true,
	})
	return nil
}

// checkAmbientMode handles validation for Istio Ambient mode.
// In ambient mode, there's no local sidecar to query, so we require explicit clusterID
// and trust it for traffic fencing via DestinationRule.
func (c *checker) checkAmbientMode(_ context.Context, accessor status.Accessor) error {
	c.logger.Info("Running in Istio Ambient mode (sidecarless)")

	// Ambient mode REQUIRES explicit clusterID (no fallback to clusterName)
	// because we cannot validate it at runtime
	if c.configuredClusterID == "" {
		c.logger.Warn("Ambient mode detected but explicit cluster ID not configured - " +
			"in multi-cluster deployments, traffic may be routed to other clusters")
		accessor.AddCheck(&status.StatusCheck{
			Name:    DiagnosticIstioXClusterLB,
			Passing: false,
			Error: "Istio Ambient mode detected but integrations.istio.clusterID not configured. " +
				"In Ambient mode, explicit cluster ID is required for traffic fencing. " +
				"Set integrations.istio.clusterID to your Istio cluster ID.",
		})
		return nil
	}

	// If topology label is available, validate it matches configured cluster ID
	if c.topologyCluster != "" && c.topologyCluster != c.configuredClusterID {
		c.logger.Warnf("Configured cluster ID '%s' does not match pod topology label '%s'",
			c.configuredClusterID, c.topologyCluster)
		accessor.AddCheck(&status.StatusCheck{
			Name:    DiagnosticIstioXClusterLB,
			Passing: false,
			Error: fmt.Sprintf("Configured cluster ID '%s' does not match pod topology label '%s'. "+
				"Update integrations.istio.clusterID to match your Istio cluster ID.",
				c.configuredClusterID, c.topologyCluster),
		})
		return nil
	}

	// Trust the configured cluster ID since we can't validate against sidecar
	c.logger.Infof("Ambient mode: trusting configured cluster ID '%s' for traffic fencing", c.configuredClusterID)
	accessor.AddCheck(&status.StatusCheck{
		Name:    DiagnosticIstioXClusterLB,
		Passing: true,
	})
	return nil
}

// checkSidecarMode handles validation for Istio Sidecar mode.
// This is the traditional mode with per-pod Envoy proxies that we can query.
// In sidecar mode, we validate the effective cluster ID (clusterID || clusterName)
// against what Istio knows, since that's what Helm uses for the DestinationRule.
func (c *checker) checkSidecarMode(ctx context.Context, client *http.Client, accessor status.Accessor) error {
	// Get the Istio cluster ID from the sidecar
	istioClusterID, err := c.getIstioClusterID(ctx, client)
	if err != nil {
		// Unexpected - we already verified sidecar was reachable in detectIstioMode
		c.logger.Warnf("Could not query Envoy sidecar: %v", err)
		accessor.AddCheck(&status.StatusCheck{
			Name:    DiagnosticIstioXClusterLB,
			Passing: true,
		})
		return nil
	}

	c.logger.Infof("Sidecar mode: Istio cluster ID from Envoy: '%s'", istioClusterID)

	// Compute effective cluster ID (what Helm uses for DestinationRule)
	// Note: clusterName is always set (required field, auto-detected by scout)
	// so effective is never empty
	effective := c.effectiveClusterID()
	c.logger.Infof("Effective cluster ID (for DestinationRule): '%s'", effective)

	// Validate effective cluster ID matches Istio's cluster ID
	// This ensures the DestinationRule will work correctly
	if istioClusterID != "" && effective != istioClusterID {
		c.logger.Warnf("Effective cluster ID '%s' does not match Istio cluster ID '%s'",
			effective, istioClusterID)
		accessor.AddCheck(&status.StatusCheck{
			Name:    DiagnosticIstioXClusterLB,
			Passing: false,
			Error: fmt.Sprintf("Effective cluster ID '%s' does not match Istio cluster ID '%s'. "+
				"The DestinationRule will not route traffic correctly. "+
				"Set integrations.istio.clusterID to '%s' to match your Istio configuration.",
				effective, istioClusterID, istioClusterID),
		})
		return nil
	}

	// Also check topology label if available
	if c.topologyCluster != "" && c.topologyCluster != effective {
		c.logger.Warnf("Effective cluster ID '%s' does not match pod topology label '%s'",
			effective, c.topologyCluster)
		accessor.AddCheck(&status.StatusCheck{
			Name:    DiagnosticIstioXClusterLB,
			Passing: false,
			Error: fmt.Sprintf("Effective cluster ID '%s' does not match pod topology label '%s'. "+
				"Set integrations.istio.clusterID to '%s' to match your Istio configuration.",
				effective, c.topologyCluster, c.topologyCluster),
		})
		return nil
	}

	// Query Envoy sidecar to detect cross-cluster load balancing
	xclusterDetected, err := c.detectCrossClusterLB(ctx, client)
	if err != nil {
		c.logger.Warnf("Could not query Envoy clusters endpoint: %v", err)
		accessor.AddCheck(&status.StatusCheck{
			Name:    DiagnosticIstioXClusterLB,
			Passing: true,
		})
		return nil
	}

	if xclusterDetected {
		c.logger.Info("Cross-cluster load balancing detected, cluster-local routing configured correctly")
	} else {
		c.logger.Info("No cross-cluster load balancing detected for aggregator service")
	}

	accessor.AddCheck(&status.StatusCheck{
		Name:    DiagnosticIstioXClusterLB,
		Passing: true,
	})
	return nil
}

// serverInfoResponse represents the structure of the Envoy /server_info response
type serverInfoResponse struct {
	Node struct {
		Metadata map[string]interface{} `json:"metadata"`
	} `json:"node"`
}

// getIstioClusterID queries the Envoy sidecar's server_info endpoint to get the Istio cluster ID
func (c *checker) getIstioClusterID(ctx context.Context, client *http.Client) (string, error) {
	var lastErr error

	for attempt := 1; attempt <= MaxRetry; attempt++ {
		resp, err := client.Get(c.serverInfoURL)
		if err != nil {
			lastErr = fmt.Errorf("failed to query Envoy server_info: %w", err)
			c.logger.Debugf(logRetryAttempt, attempt, lastErr)
			time.Sleep(RetryInterval)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("envoy server_info returned status %d", resp.StatusCode)
			c.logger.Debugf(logRetryAttempt, attempt, lastErr)
			time.Sleep(RetryInterval)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("failed to read Envoy server_info response: %w", err)
			c.logger.Debugf(logRetryAttempt, attempt, lastErr)
			time.Sleep(RetryInterval)
			continue
		}

		var info serverInfoResponse
		if err := json.Unmarshal(body, &info); err != nil {
			lastErr = fmt.Errorf("failed to parse Envoy server_info JSON: %w", err)
			c.logger.Debugf(logRetryAttempt, attempt, lastErr)
			time.Sleep(RetryInterval)
			continue
		}

		// The cluster ID is in node.metadata.CLUSTER_ID
		if clusterID, ok := info.Node.Metadata["CLUSTER_ID"].(string); ok && clusterID != "" {
			return clusterID, nil
		}

		// If CLUSTER_ID not found, return empty but no error (sidecar is reachable)
		c.logger.Debug("CLUSTER_ID not found in Envoy server_info metadata")
		return "", nil
	}

	return "", lastErr
}

// detectCrossClusterLB queries the Envoy sidecar's admin API to check if the aggregator
// service has endpoints in multiple clusters (indicating cross-cluster load balancing).
func (c *checker) detectCrossClusterLB(ctx context.Context, client *http.Client) (bool, error) {
	var lastErr error

	for attempt := 1; attempt <= MaxRetry; attempt++ {
		resp, err := client.Get(c.clustersURL)
		if err != nil {
			lastErr = fmt.Errorf("failed to query Envoy admin API: %w", err)
			c.logger.Debugf(logRetryAttempt, attempt, lastErr)
			time.Sleep(RetryInterval)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("envoy admin API returned status %d", resp.StatusCode)
			c.logger.Debugf(logRetryAttempt, attempt, lastErr)
			time.Sleep(RetryInterval)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("failed to read Envoy response: %w", err)
			c.logger.Debugf(logRetryAttempt, attempt, lastErr)
			time.Sleep(RetryInterval)
			continue
		}

		return c.parseEnvoyResponse(string(body)), nil
	}

	return false, lastErr
}

// parseEnvoyResponse parses the Envoy clusters response and checks for multiple
// distinct localities for the aggregator service, which indicates cross-cluster LB.
//
// The text format (default) looks like:
// outbound|8080||aggregator.cza.svc.cluster.local::10.0.1.5:8080::region::us-east-1
// outbound|8080||aggregator.cza.svc.cluster.local::10.0.1.5:8080::zone::us-east-1a
//
// We look for the aggregator service and check if it has endpoints with different regions.
func (c *checker) parseEnvoyResponse(body string) bool {
	// Build the service FQDN pattern to look for
	// Format: outbound|<port>||<service>.<namespace>.svc.cluster.local
	servicePattern := fmt.Sprintf("%s.%s.svc.cluster.local", c.aggregatorService, c.namespace)

	c.logger.Debugf("Looking for service pattern: %s", servicePattern)

	// Track unique regions/zones for the aggregator service
	regions := make(map[string]struct{})

	scanner := bufio.NewScanner(strings.NewReader(body))
	for scanner.Scan() {
		line := scanner.Text()

		// Skip lines that don't match our service
		if !strings.Contains(line, servicePattern) {
			continue
		}

		// Look for region information in the line
		// Format: ...::region::us-east-1
		if strings.Contains(line, "::region::") {
			parts := strings.Split(line, "::region::")
			if len(parts) >= 2 {
				// Extract the region value (everything up to the next ::)
				regionPart := strings.Split(parts[1], "::")[0]
				regionPart = strings.TrimSpace(regionPart)
				if regionPart != "" {
					c.logger.Debugf("Found region '%s' for service %s", regionPart, servicePattern)
					regions[regionPart] = struct{}{}
				}
			}
		}
	}

	// If we have more than one distinct region, cross-cluster LB is active
	if len(regions) > 1 {
		regionList := make([]string, 0, len(regions))
		for r := range regions {
			regionList = append(regionList, r)
		}
		c.logger.Infof("Cross-cluster LB detected: aggregator service has endpoints in %d regions: %v",
			len(regions), regionList)
		return true
	}

	if len(regions) == 1 {
		for r := range regions {
			c.logger.Debugf("Aggregator service endpoints are all in region: %s", r)
		}
	} else {
		c.logger.Debug("No region information found for aggregator service endpoints")
	}

	return false
}
