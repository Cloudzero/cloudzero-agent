// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build integration
// +build integration

package webhook

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"

	"github.com/cloudzero/cloudzero-agent/tests/kind"
)

const (
	chartClusterName = "cloudzero-webhook-chart-test"
	chartTestTimeout = 15 * time.Minute
	webhookNamespace = "cz-webhook-test"
	releaseName      = "webhook-chart-test"
)

// Helm chart configuration constants
const (
	testCloudAccountID = "test-account-123"
	testClusterName    = "webhook-chart-test"
	testRegion         = "us-west-2"
	testSecretName     = "api-token"
	testHost           = "dev-api.cloudzero.com"
	helmTimeout        = "10m"
)

// Service configuration constants
const (
	webhookServicePort = 443
	serviceSuffix      = "-cloudzero-agent-webhook-server-svc"
)

// Test resource constants
const (
	testDeploymentName = "webhook-test-deployment"
	testServiceName    = "webhook-test-service"
	testNamespaceName  = "webhook-test-namespace"
	defaultNamespace   = "default"
	webhookWaitTime    = 45 * time.Second
)

// kubectlCommand creates a kubectl command with the given kubeconfig and arguments
func kubectlCommand(kubeconfig string, args ...string) *exec.Cmd {
	cmdArgs := append([]string{"--kubeconfig", kubeconfig}, args...)
	cmd := exec.Command("kubectl", cmdArgs...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", kubeconfig))
	return cmd
}

// helmCommand creates a helm command with the given kubeconfig and arguments
func helmCommand(kubeconfig string, args ...string) *exec.Cmd {
	cmd := exec.Command("helm", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", kubeconfig))
	return cmd
}

// buildWebhookServiceURL constructs the webhook service URL for metrics endpoint
func buildWebhookServiceURL(releaseName, namespace string) string {
	return fmt.Sprintf("https://%s%s.%s.svc:%d/metrics",
		releaseName, serviceSuffix, namespace, webhookServicePort)
}

// applyManifestFromFile applies a Kubernetes manifest file using kubectl
func applyManifestFromFile(t *testing.T, kubeconfig, manifestPath string) {
	t.Logf("Applying manifest: %s", filepath.Base(manifestPath))
	cmd := kubectlCommand(kubeconfig, "apply", "-f", manifestPath)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Kubectl apply output: %s", string(output))
		t.Fatalf("Failed to apply manifest %s: %v", manifestPath, err)
	}
	t.Logf("✅ Applied %s successfully", filepath.Base(manifestPath))
}

func TestWebhookChartIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chart integration test in short mode")
	}

	// Require API key for chart deployment
	apiKey := os.Getenv("CLOUDZERO_DEV_API_KEY")
	if apiKey == "" {
		t.Skip("CLOUDZERO_DEV_API_KEY required for chart integration test")
	}

	// Set up debug logging
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	t.Log("Starting webhook Helm chart integration test")

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "cloudzero-webhook-chart-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Setup Kind cluster
	kubeconfig := setupChartKindCluster(t, tempDir)
	defer kind.CleanupCluster(t, chartClusterName)

	// Deploy webhook chart
	t.Log("Deploying webhook Helm chart...")
	deployWebhookChart(t, kubeconfig, apiKey)
	defer uninstallWebhookChart(t, kubeconfig)

	// Test webhook invocations by deploying resources
	t.Log("Testing webhook invocations by creating resources...")
	deployWorkloadsFromManifests(t, kubeconfig)

	// Get webhook metrics to verify invocations
	t.Log("Getting webhook metrics to verify invocations...")
	metrics := getWebhookMetrics(t, kubeconfig)
	if metrics != "" {
		t.Logf("✅ Successfully retrieved webhook metrics (%d bytes)", len(metrics))
		validateWebhookInvocations(t, metrics)
	} else {
		t.Log("❌ Could not retrieve webhook metrics")
	}

	t.Log("✅ Webhook chart integration test completed successfully")
}

func setupChartKindCluster(t *testing.T, tempDir string) string {
	t.Log("Setting up Kind cluster for chart deployment...")

	return kind.SetupClusterWithEmbeddedConfig(t, chartClusterName, tempDir)
}

func deployWebhookChart(t *testing.T, kubeconfig, apiKey string) {
	t.Log("Deploying CloudZero webhook chart...")

	// Find project root
	currentDir, err := os.Getwd()
	require.NoError(t, err)

	projectRoot := currentDir
	for !fileExists(filepath.Join(projectRoot, "helm", "Chart.yaml")) {
		parent := filepath.Dir(projectRoot)
		if parent == projectRoot {
			t.Fatal("Could not find helm directory")
		}
		projectRoot = parent
	}
	helmDir := filepath.Join(projectRoot, "helm")

	t.Logf("Using Helm chart from: %s", helmDir)

	// Create namespace
	cmd := kubectlCommand(kubeconfig, "create", "namespace", webhookNamespace)
	cmd.Run() // Ignore error if namespace exists

	// Create API key secret
	cmd = kubectlCommand(kubeconfig, "create", "secret", "generic", testSecretName,
		"--namespace", webhookNamespace,
		fmt.Sprintf("--from-literal=value=%s", apiKey))
	output, err := cmd.CombinedOutput()
	if err != nil && !strings.Contains(string(output), "already exists") {
		t.Fatalf("Failed to create API key secret: %v, output: %s", err, string(output))
	}

	// Update Helm dependencies
	cmd = helmCommand(kubeconfig, "dependency", "update", helmDir)
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Logf("Helm dependency update output: %s", string(output))
		t.Fatalf("Failed to update Helm dependencies: %v", err)
	}

	// Get values override file path
	testDir := filepath.Dir(kind.GetCurrentFile())
	valuesPath := filepath.Join(testDir, "testdata", "values-override.yaml")
	if !fileExists(valuesPath) {
		t.Fatalf("Values override file not found: %s", valuesPath)
	}

	// Install Helm chart with webhook enabled and minimal replicas for faster testing
	cmd = helmCommand(kubeconfig, "upgrade", "--install", releaseName, helmDir,
		"--namespace", webhookNamespace,
		"--values", valuesPath,
		"--timeout", helmTimeout,
		"--wait",
		"--debug",
	)

	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Logf("Helm install output: %s", string(output))
		t.Fatalf("Failed to install Helm chart: %v", err)
	}

	t.Log("✅ Helm chart installed successfully")
}

func uninstallWebhookChart(t *testing.T, kubeconfig string) {
	t.Log("Uninstalling webhook Helm chart...")

	cmd := helmCommand(kubeconfig, "uninstall", releaseName, "--namespace", webhookNamespace)
	if err := cmd.Run(); err != nil {
		t.Logf("Warning: Failed to uninstall Helm chart: %v", err)
	}
}

func deployWorkloadsFromManifests(t *testing.T, kubeconfig string) {
	// Find the testdata directory
	currentDir, err := os.Getwd()
	require.NoError(t, err)

	testdataDir := filepath.Join(currentDir, "testdata")
	if !fileExists(testdataDir) {
		t.Fatalf("testdata directory not found at %s", testdataDir)
	}

	// Apply test manifests to trigger webhook invocations
	manifests := []string{
		"test-deployment.yaml", // Should trigger webhook for "deployments"
		"test-service.yaml",    // Should trigger webhook for "services"
		"test-namespace.yaml",  // Should trigger webhook for "namespaces"
	}

	for _, manifest := range manifests {
		manifestPath := filepath.Join(testdataDir, manifest)
		if !fileExists(manifestPath) {
			t.Fatalf("Test manifest not found: %s", manifestPath)
		}
		applyManifestFromFile(t, kubeconfig, manifestPath)
	}

	// Wait for webhook to process resources
	t.Log("Waiting for webhook to process resources...")
	time.Sleep(webhookWaitTime)
}

func getWebhookMetrics(t *testing.T, kubeconfig string) string {
	t.Log("Accessing webhook metrics using kubectl run with curl container...")

	// Build the service URL for the webhook metrics endpoint
	serviceURL := buildWebhookServiceURL(releaseName, webhookNamespace)
	t.Logf("Target service URL: %s", serviceURL)

	// Create file path for debugging metrics output
	metricsFile := filepath.Join(os.TempDir(), "webhook-metrics-debug.log")
	t.Logf("📄 Metrics will be saved to: %s", metricsFile)

	// Debug: List all services in the webhook namespace
	t.Log("🔍 Debugging: Listing services in webhook namespace...")
	listCmd := kubectlCommand(kubeconfig, "get", "services", "-n", webhookNamespace, "-o", "wide")
	if output, err := listCmd.CombinedOutput(); err != nil {
		t.Logf("Failed to list services: %v", err)
	} else {
		t.Logf("Services in namespace %s:\n%s", webhookNamespace, string(output))
	}

	// Debug: Describe the specific webhook service
	expectedServiceName := releaseName + serviceSuffix
	t.Logf("🔍 Debugging: Describing service '%s'...", expectedServiceName)
	describeCmd := kubectlCommand(kubeconfig, "describe", "service", expectedServiceName, "-n", webhookNamespace)
	if output, err := describeCmd.CombinedOutput(); err != nil {
		t.Logf("Failed to describe webhook service: %v", err)
	} else {
		t.Logf("Webhook service details:\n%s", string(output))
	}

	// Debug: Check service endpoints
	t.Log("🔍 Debugging: Checking service endpoints...")
	endpointsCmd := kubectlCommand(kubeconfig, "get", "endpoints", expectedServiceName, "-n", webhookNamespace, "-o", "yaml")
	if output, err := endpointsCmd.CombinedOutput(); err != nil {
		t.Logf("Failed to get service endpoints: %v", err)
	} else {
		t.Logf("Service endpoints:\n%s", string(output))
	}

	// Debug: Get service IP for connectivity test
	t.Log("🔍 Debugging: Getting service cluster IP...")
	getIPCmd := kubectlCommand(kubeconfig, "get", "service", expectedServiceName, "-n", webhookNamespace,
		"-o", "jsonpath={.spec.clusterIP}")
	serviceIP := ""
	if output, err := getIPCmd.Output(); err != nil {
		t.Logf("Failed to get service IP: %v", err)
	} else {
		serviceIP = strings.TrimSpace(string(output))
		t.Logf("Service cluster IP: %s", serviceIP)
	}

	// Test basic network connectivity if we have the service IP
	if serviceIP != "" && serviceIP != "<none>" {
		t.Logf("🔍 Debugging: Testing connectivity to %s:443...", serviceIP)
		pingCmd := kubectlCommand(kubeconfig, "run", "connectivity-test",
			"--image=busybox", "--rm", "-it", "--restart=Never",
			"--", "nc", "-zv", serviceIP, "443")
		if output, err := pingCmd.CombinedOutput(); err != nil {
			t.Logf("Connectivity test failed: %v, output: %s", err, string(output))
		} else {
			t.Logf("Connectivity test output: %s", string(output))
		}
	}

	// Run curl in a pod to fetch metrics with verbose output
	t.Logf("🔍 Running curl with verbose output to: %s", serviceURL)
	curlCmd := kubectlCommand(kubeconfig, "run", "curl-metrics",
		"--image=curlimages/curl", "--rm", "-it", "--restart=Never",
		"--", "curl", "-v", "-s", "-k", "--connect-timeout", "10", "--max-time", "30", serviceURL)

	output, err := curlCmd.CombinedOutput()
	curlOutput := string(output)

	if err != nil {
		t.Logf("❌ Curl command failed: %v", err)
		t.Logf("Curl output:\n%s", curlOutput)
		t.Errorf("Failed to fetch webhook metrics - this should not pass the test")
		return ""
	}

	if len(curlOutput) > 0 {
		t.Logf("✅ Successfully fetched metrics (%d bytes)", len(curlOutput))

		// Write full metrics to file for debugging
		if err := os.WriteFile(metricsFile, []byte(curlOutput), 0o644); err != nil {
			t.Logf("⚠️  Failed to write metrics to file: %v", err)
		} else {
			t.Logf("📄 Full metrics written to: %s", metricsFile)
		}

		// Only show first 200 chars to avoid spam in logs
		if len(curlOutput) > 200 {
			t.Logf("Metrics sample: %s...", curlOutput[:200])
		} else {
			t.Logf("Metrics content: %s", curlOutput)
		}
	} else {
		t.Errorf("❌ Curl succeeded but returned empty content - webhook metrics endpoint not working")
		return ""
	}

	return curlOutput
}

func validateWebhookInvocations(t *testing.T, metrics string) {
	t.Log("Validating webhook was invoked for test resources...")

	if metrics == "" {
		t.Log("❌ Could not retrieve webhook metrics - unable to validate invocations")
		return
	}

	// Look for webhook event metrics for our test resources (metrics use singular kind names)
	expectedResources := []string{"deployment", "service", "namespace"}
	foundInvocations := 0

	for _, resource := range expectedResources {
		// Look for czo_webhook_types_total metrics with create operations for our resources
		pattern := fmt.Sprintf(`czo_webhook_types_total\{.*kind_resource="%s".*operation="create".*\}\s+([1-9]\d*)`, resource)
		matched, err := regexp.MatchString(pattern, metrics)
		if err != nil {
			t.Logf("Error matching pattern for %s: %v", resource, err)
			continue
		}

		if matched {
			t.Logf("✅ Webhook was invoked for %s (create)", resource)
			foundInvocations++
		} else {
			t.Logf("❌ No webhook invocation detected for %s (create)", resource)
		}
	}

	// Overall validation
	if foundInvocations > 0 {
		t.Logf("✅ SUCCESS: Webhook was invoked for %d/%d expected resources", foundInvocations, len(expectedResources))
		t.Log("✅ This confirms the webhook is working - Kubernetes is calling our webhook for resource operations!")
	} else {
		t.Log("❌ FAILURE: No webhook invocations detected")
		t.Log("This could indicate:")
		t.Log("  - Webhook is not properly configured")
		t.Log("  - ValidatingWebhookConfiguration is not working")
		t.Log("  - Label patterns don't match the test resource labels")
		t.Log("  - Metrics are not being updated correctly")
		t.Errorf("Webhook integration test failed - no webhook invocations detected for any test resources")
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
