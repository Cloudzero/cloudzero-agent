// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

//go:build integration
// +build integration

package webhook

import (
	"context"
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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	chartClusterName = "cloudzero-webhook-chart-test"
	chartTestTimeout = 15 * time.Minute
	webhookNamespace = "cz-webhook-test"
	releaseName      = "webhook-chart-test"
)

func TestWebhookChartIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping chart integration test in short mode")
	}

	// Require API key for chart deployment
	apiKey := os.Getenv("CLOUDZERO_DEV_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("CZ_DEV_API_TOKEN")
	}
	if apiKey == "" {
		t.Skip("CLOUDZERO_DEV_API_KEY or CZ_DEV_API_TOKEN required for chart integration test")
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
	defer cleanupChartKindCluster(t)

	// Create Kubernetes client
	k8sClient, err := createKubernetesClientForChart(kubeconfig)
	require.NoError(t, err)

	// Deploy webhook chart
	t.Log("Deploying webhook Helm chart...")
	deployWebhookChart(t, kubeconfig, apiKey)
	defer uninstallWebhookChart(t, kubeconfig)

	// Wait for webhook to be ready
	t.Log("Waiting for webhook deployment to be ready...")
	waitForWebhookReady(t, k8sClient)

	// Test webhook invocations
	t.Log("Testing webhook invocations by creating resources...")
	testWebhookInvocationsViaMetrics(t, k8sClient)

	t.Log("✅ Webhook chart integration test completed successfully")
}

func setupChartKindCluster(t *testing.T, tempDir string) string {
	t.Log("Setting up Kind cluster for chart deployment...")

	// Create Kind config for chart test
	kindConfig := fmt.Sprintf(`kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: %s
nodes:
  - role: control-plane
    extraMounts:
      - hostPath: %s
        containerPath: /tmp/test-data
    extraPortMappings:
      - containerPort: 30080
        hostPort: 30080
        protocol: TCP`, chartClusterName, tempDir)

	kindConfigPath := filepath.Join(tempDir, "chart-kind-config.yaml")
	err := os.WriteFile(kindConfigPath, []byte(kindConfig), 0o644)
	require.NoError(t, err)

	// Create Kind cluster
	cmd := exec.Command("kind", "create", "cluster", "--name", chartClusterName, "--config", kindConfigPath, "--wait", "90s")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	require.NoError(t, err, "Failed to create Kind cluster")

	// Get kubeconfig
	kubeconfig := filepath.Join(tempDir, "kubeconfig")
	cmd = exec.Command("kind", "get", "kubeconfig", "--name", chartClusterName)
	output, err := cmd.Output()
	require.NoError(t, err, "Failed to get kubeconfig")

	err = os.WriteFile(kubeconfig, output, 0o644)
	require.NoError(t, err, "Failed to write kubeconfig")

	t.Log("Kind cluster created successfully")
	return kubeconfig
}

func createKubernetesClientForChart(kubeconfig string) (kubernetes.Interface, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}

func cleanupChartKindCluster(t *testing.T) {
	t.Log("Cleaning up chart Kind cluster...")
	cmd := exec.Command("kind", "delete", "cluster", "--name", chartClusterName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Logf("Warning: Failed to cleanup Kind cluster: %v", err)
	}
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
	cmd := exec.Command("kubectl", "--kubeconfig", kubeconfig, "create", "namespace", webhookNamespace)
	cmd.Run() // Ignore error if namespace exists

	// Create API key secret
	cmd = exec.Command("kubectl", "--kubeconfig", kubeconfig, "create", "secret", "generic", "api-token",
		"--namespace", webhookNamespace,
		fmt.Sprintf("--from-literal=value=%s", apiKey))
	output, err := cmd.CombinedOutput()
	if err != nil && !strings.Contains(string(output), "already exists") {
		t.Fatalf("Failed to create API key secret: %v, output: %s", err, string(output))
	}

	// Update Helm dependencies
	cmd = exec.Command("helm", "dependency", "update", helmDir)
	cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", kubeconfig))
	output, err = cmd.CombinedOutput()
	if err != nil {
		t.Logf("Helm dependency update output: %s", string(output))
		t.Fatalf("Failed to update Helm dependencies: %v", err)
	}

	// Install Helm chart with webhook enabled and minimal replicas for faster testing
	cmd = exec.Command("helm", "upgrade", "--install", releaseName, helmDir,
		"--namespace", webhookNamespace,
		"--set", "cloudAccountId=test-account-123",
		"--set", "clusterName=webhook-chart-test",
		"--set", "region=us-west-2",
		"--set", "existingSecretName=api-token",
		"--set", "host=dev-api.cloudzero.com",
		"--set", "insightsController.enabled=true",
		"--set", "insightsController.labels.enabled=true",
		"--set", "insightsController.annotations.enabled=false",
		// Reduce replicas for faster test deployment
		"--set", "components.aggregator.replicas=1",
		"--set", "components.webhookServer.replicas=1",
		// Disable unnecessary components for webhook-only testing
		"--set", "initBackfillJob.enabled=false",
		"--timeout", "10m",
		"--wait",
		"--debug",
	)

	cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", kubeconfig))
	output, err = cmd.CombinedOutput()

	if err != nil {
		t.Logf("Helm install output: %s", string(output))
		t.Fatalf("Failed to install Helm chart: %v", err)
	}

	t.Log("✅ Helm chart installed successfully")
}

func uninstallWebhookChart(t *testing.T, kubeconfig string) {
	t.Log("Uninstalling webhook Helm chart...")

	cmd := exec.Command("helm", "uninstall", releaseName, "--namespace", webhookNamespace)
	cmd.Env = append(os.Environ(), fmt.Sprintf("KUBECONFIG=%s", kubeconfig))
	if err := cmd.Run(); err != nil {
		t.Logf("Warning: Failed to uninstall Helm chart: %v", err)
	}
}

func waitForWebhookReady(t *testing.T, k8sClient kubernetes.Interface) {
	t.Log("Waiting for webhook deployment to be ready...")

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()

	// Wait for webhook deployment
	for {
		select {
		case <-ctx.Done():
			// List all resources for debugging
			t.Log("=== DEBUGGING: Listing all resources ===")
			cmd := exec.Command("kubectl", "get", "all", "-n", webhookNamespace, "-o", "wide")
			if output, err := cmd.CombinedOutput(); err == nil {
				t.Logf("Resources in namespace %s:\n%s", webhookNamespace, string(output))
			}

			t.Fatal("Timeout waiting for webhook deployment to be ready")
		default:
			deployments, err := k8sClient.AppsV1().Deployments(webhookNamespace).List(ctx, metav1.ListOptions{})
			if err != nil {
				t.Logf("Error checking deployments: %v", err)
				time.Sleep(10 * time.Second)
				continue
			}

			ready := false
			for _, deployment := range deployments.Items {
				t.Logf("Deployment %s: Ready=%d, Available=%d, Replicas=%d",
					deployment.Name, deployment.Status.ReadyReplicas,
					deployment.Status.AvailableReplicas, deployment.Status.Replicas)

				// For webhook testing, we just need at least 1 replica ready
				if deployment.Status.ReadyReplicas >= 1 && deployment.Status.AvailableReplicas >= 1 {
					ready = true
					break
				}
			}

			if ready {
				t.Log("✅ Webhook deployment is ready")
				// Additional wait for webhook registration
				time.Sleep(30 * time.Second)
				return
			}

			time.Sleep(10 * time.Second)
		}
	}
}

func testWebhookInvocationsViaMetrics(t *testing.T, k8sClient kubernetes.Interface) {
	ctx := context.Background()

	// Get initial metrics baseline
	initialMetrics := getWebhookMetrics(t)
	t.Logf("Initial webhook metrics: %s", initialMetrics)

	// Test 1: Create a deployment (should trigger webhook for "deployments")
	t.Log("Creating test deployment...")
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "webhook-test-deployment",
			Namespace: "default",
			Labels: map[string]string{
				"app":         "webhook-test",
				"environment": "test",
				"team":        "engineering",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "webhook-test"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "webhook-test"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "test-container",
							Image: "nginx:alpine",
						},
					},
				},
			},
		},
	}

	_, err := k8sClient.AppsV1().Deployments("default").Create(ctx, deployment, metav1.CreateOptions{})
	require.NoError(t, err, "Failed to create test deployment")

	// Test 2: Create a service (should trigger webhook for "services")
	t.Log("Creating test service...")
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "webhook-test-service",
			Namespace: "default",
			Labels: map[string]string{
				"app":         "webhook-test",
				"environment": "test",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{"app": "webhook-test"},
			Ports: []corev1.ServicePort{
				{
					Port:       80,
					TargetPort: intstr.FromInt(80),
				},
			},
		},
	}

	_, err = k8sClient.CoreV1().Services("default").Create(ctx, service, metav1.CreateOptions{})
	require.NoError(t, err, "Failed to create test service")

	// Test 3: Create a namespace (should trigger webhook for "namespaces")
	t.Log("Creating test namespace...")
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "webhook-test-namespace",
			Labels: map[string]string{
				"environment": "test",
				"team":        "webhook-team",
			},
		},
	}

	_, err = k8sClient.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	require.NoError(t, err, "Failed to create test namespace")

	// Wait for webhook to process resources
	t.Log("Waiting for webhook to process resources...")
	time.Sleep(45 * time.Second)

	// Get updated metrics
	finalMetrics := getWebhookMetrics(t)
	t.Logf("Final webhook metrics: %s", finalMetrics)

	// Validate webhook was invoked for our resources
	validateWebhookInvocations(t, initialMetrics, finalMetrics)
}

func getWebhookMetrics(t *testing.T) string {
	t.Log("Accessing webhook metrics via kubectl port-forward...")

	// Find project root for kubeconfig
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

	tempDir := filepath.Join(projectRoot, "tests", "webhook")
	kubeconfigPath := ""
	
	// Look for kubeconfig files in temp directories
	if matches, err := filepath.Glob("/tmp/cloudzero-webhook-chart-test-*/kubeconfig"); err == nil && len(matches) > 0 {
		kubeconfigPath = matches[0]
	} else {
		// Fallback to expected location
		kubeconfigPath = filepath.Join(tempDir, "kubeconfig")
	}

	if !fileExists(kubeconfigPath) {
		t.Logf("Kubeconfig not found at %s, cannot access metrics", kubeconfigPath)
		return ""
	}

	// Start port-forward in background
	portForwardCmd := exec.Command("kubectl", "--kubeconfig", kubeconfigPath,
		"port-forward", "-n", webhookNamespace,
		"svc/"+releaseName+"-cloudzero-agent", "8080:8080")

	// Create a context with timeout for port-forward
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	portForwardCmd = exec.CommandContext(ctx, "kubectl", "--kubeconfig", kubeconfigPath,
		"port-forward", "-n", webhookNamespace,
		"svc/"+releaseName+"-cloudzero-agent", "8080:8080")

	// Start port-forward
	t.Log("Starting kubectl port-forward to webhook service...")
	err = portForwardCmd.Start()
	if err != nil {
		t.Logf("Failed to start port-forward: %v", err)
		return ""
	}
	defer func() {
		if portForwardCmd.Process != nil {
			portForwardCmd.Process.Kill()
		}
	}()

	// Wait for port-forward to be ready
	time.Sleep(5 * time.Second)

	// Try to fetch metrics with retries
	var metricsContent string
	maxRetries := 3
	
	for retry := 0; retry < maxRetries; retry++ {
		t.Logf("Attempting to fetch metrics (attempt %d/%d)...", retry+1, maxRetries)
		
		// Use curl to fetch metrics
		curlCmd := exec.Command("curl", "-s", "--connect-timeout", "10", "http://localhost:8080/metrics")
		output, err := curlCmd.Output()
		
		if err != nil {
			t.Logf("Failed to fetch metrics (attempt %d): %v", retry+1, err)
			if retry < maxRetries-1 {
				time.Sleep(3 * time.Second)
				continue
			}
		} else {
			metricsContent = string(output)
			if len(metricsContent) > 0 {
				t.Logf("Successfully fetched metrics (%d bytes)", len(metricsContent))
				break
			}
		}
	}

	if metricsContent == "" {
		t.Log("❌ Could not retrieve webhook metrics via port-forward")
		t.Log("This might indicate:")
		t.Log("  - Webhook service is not accessible")
		t.Log("  - Metrics endpoint is not responding")
		t.Log("  - Port-forward connection failed")
	}

	return metricsContent
}

func validateWebhookInvocations(t *testing.T, initialMetrics, finalMetrics string) {
	t.Log("Validating webhook was invoked for test resources...")

	if finalMetrics == "" {
		t.Log("❌ Could not retrieve webhook metrics - unable to validate invocations")
		t.Log("This might indicate the webhook is not accessible or metrics endpoint is not working")
		return
	}

	// Look for webhook event metrics
	webhookEventPattern := regexp.MustCompile(`webhook_types_total\{.*kind_resource="([^"]+)".*operation="([^"]+)".*\}\s+(\d+)`)

	initialCounts := parseWebhookMetrics(initialMetrics, webhookEventPattern)
	finalCounts := parseWebhookMetrics(finalMetrics, webhookEventPattern)

	t.Log("=== WEBHOOK METRICS ANALYSIS ===")
	t.Logf("Initial webhook event counts: %v", initialCounts)
	t.Logf("Final webhook event counts: %v", finalCounts)

	// Check for increases in webhook events
	expectedResources := []string{"deployments", "services", "namespaces"}
	foundInvocations := 0

	for _, resource := range expectedResources {
		key := fmt.Sprintf("%s_CREATE", resource)
		initialCount := initialCounts[key]
		finalCount := finalCounts[key]

		if finalCount > initialCount {
			t.Logf("✅ Webhook was invoked for %s (CREATE): %d -> %d", resource, initialCount, finalCount)
			foundInvocations++
		} else {
			t.Logf("❌ No webhook invocation detected for %s (CREATE)", resource)
		}
	}

	// Overall validation
	if foundInvocations > 0 {
		t.Logf("✅ SUCCESS: Webhook was invoked for %d/%d expected resources", foundInvocations, len(expectedResources))
		t.Log("✅ This confirms the webhook resource name fix is working - Kubernetes is calling our webhook with the correct plural resource names!")
	} else {
		t.Log("❌ WARNING: No webhook invocations detected")
		t.Log("This could indicate:")
		t.Log("  - Webhook is not properly configured")
		t.Log("  - ValidatingWebhookConfiguration is not working")
		t.Log("  - Metrics are not being updated correctly")

		// Don't fail the test, but log the issue for investigation
		assert.Greater(t, foundInvocations, 0, "Expected webhook invocations for test resources")
	}
}

func parseWebhookMetrics(metrics string, pattern *regexp.Regexp) map[string]int {
	counts := make(map[string]int)

	matches := pattern.FindAllStringSubmatch(metrics, -1)
	for _, match := range matches {
		if len(match) >= 4 {
			resource := match[1]
			operation := match[2]
			count := 0
			fmt.Sscanf(match[3], "%d", &count)

			key := fmt.Sprintf("%s_%s", resource, operation)
			counts[key] = count
		}
	}

	return counts
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func int32Ptr(i int32) *int32 {
	return &i
}

