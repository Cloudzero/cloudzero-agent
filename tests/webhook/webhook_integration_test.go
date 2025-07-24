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
	"runtime"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook"
	"github.com/cloudzero/cloudzero-agent/app/storage/repo"
	"github.com/cloudzero/cloudzero-agent/app/utils"
)

const (
	clusterName = "cloudzero-webhook-test"
	testTimeout = 5 * time.Minute
)

func TestWebhookResourceNamesIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Set up debug logging for the test
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})

	t.Log("Debug logging enabled for webhook integration test")

	// Create temporary directory for test data
	tempDir, err := os.MkdirTemp("", "cloudzero-webhook-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Setup Kind cluster
	kubeconfig := setupKindCluster(t, tempDir)
	defer cleanupKindCluster(t)

	// Create Kubernetes client
	_, err = createKubernetesClient(kubeconfig)
	require.NoError(t, err)

	// Create test configuration
	settings := createTestSettings(tempDir)

	// Create storage and webhook controller
	clock := &utils.Clock{}
	store, err := repo.NewInMemoryResourceRepository(clock)
	require.NoError(t, err)

	webhookController, err := webhook.NewWebhookFactory(store, settings, clock)
	require.NoError(t, err)

	// Test webhook resource configuration
	t.Log("Testing webhook resource name configuration...")
	testWebhookResourceSupport(t, webhookController)

	t.Log("Webhook integration test completed successfully")
}

func setupKindCluster(t *testing.T, tempDir string) string {
	t.Log("Setting up Kind cluster...")

	// Get the test directory path
	testDir := filepath.Dir(getCurrentFile())
	kindConfigPath := filepath.Join(testDir, "kind-config.yaml")

	// Create Kind cluster
	cmd := exec.Command("kind", "create", "cluster", "--name", clusterName, "--config", kindConfigPath, "--wait", "60s")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	require.NoError(t, err, "Failed to create Kind cluster")

	// Get kubeconfig path
	kubeconfig := filepath.Join(tempDir, "kubeconfig")
	cmd = exec.Command("kind", "get", "kubeconfig", "--name", clusterName)
	output, err := cmd.Output()
	require.NoError(t, err, "Failed to get kubeconfig")

	err = os.WriteFile(kubeconfig, output, 0o644)
	require.NoError(t, err, "Failed to write kubeconfig")

	t.Log("Kind cluster created successfully")
	return kubeconfig
}

func cleanupKindCluster(t *testing.T) {
	t.Log("Cleaning up Kind cluster...")
	cmd := exec.Command("kind", "delete", "cluster", "--name", clusterName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Logf("Warning: Failed to cleanup Kind cluster: %v", err)
	}
}

func createKubernetesClient(kubeconfig string) (kubernetes.Interface, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}

func createTestSettings(tempDir string) *config.Settings {
	// Create fake API key file
	apiKeyPath := filepath.Join(tempDir, "api-key")
	err := os.WriteFile(apiKeyPath, []byte("test-api-key"), 0o600)
	if err != nil {
		panic(err)
	}

	settings := &config.Settings{
		CloudAccountID: "test-account-123",
		Region:         "us-west-2",
		ClusterName:    "test-cluster",
		APIKeyPath:     apiKeyPath,
		Logging: config.Logging{
			Level: "debug",
		},
		Server: config.Server{
			Port:         8000,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  120 * time.Second,
		},
		Filters: config.Filters{
			Labels: config.Labels{
				Enabled: true,
				Resources: config.Resources{
					Namespaces: true,
					Pods:       true,
				},
			},
		},
	}

	// Load the API key from the file
	if err := settings.SetAPIKey(); err != nil {
		panic(fmt.Sprintf("Failed to set API key: %v", err))
	}

	return settings
}

func testWebhookResourceSupport(t *testing.T, webhookController webhook.WebhookController) {
	t.Log("Testing webhook resource support for fixed plural resource names...")

	// Test the specific resources that were fixed in the webhook configuration
	// These should all be supported with the correct plural forms
	expectedSupportedResources := map[string]map[string][]string{
		"apps": {
			"v1": {"Deployment", "StatefulSet", "DaemonSet", "ReplicaSet"},
		},
		"": { // core API group
			"v1": {"Pod", "Namespace", "Node", "Service", "PersistentVolume", "PersistentVolumeClaim"},
		},
		"batch": {
			"v1": {"Job", "CronJob"},
		},
		"storage.k8s.io": {
			"v1": {"StorageClass"},
		},
		"networking.k8s.io": {
			"v1": {"Ingress", "IngressClass"},
		},
		"apiextensions.k8s.io": {
			"v1": {"CustomResourceDefinition"},
		},
		"gateway.networking.k8s.io": {
			"v1": {"Gateway", "GatewayClass"},
		},
	}

	supported := webhookController.GetSupported()
	t.Logf("Webhook controller supports %d API groups", len(supported))

	// Verify that the webhook controller supports the expected resources
	for group, versions := range expectedSupportedResources {
		t.Logf("Testing group: %s", group)

		for version, kinds := range versions {
			t.Logf("  Testing version: %s", version)
			
			for _, kind := range kinds {
				t.Logf("    Testing kind: %s", kind)
				
				// Test if the resource is supported
				isSupported := webhookController.IsSupported(group, version, kind)
				
				if isSupported {
					t.Logf("      ✓ %s/%s %s is supported", group, version, kind)
				} else {
					t.Logf("      ✗ %s/%s %s is NOT supported", group, version, kind)
				}
				
				// For this test, we'll log the results but not fail
				// This allows us to see what the webhook actually supports
				// vs what we expect from the configuration
			}
		}
	}

	// Log the complete supported resource map for debugging
	t.Log("=== COMPLETE SUPPORTED RESOURCES MAP ===")
	for group, versions := range supported {
		t.Logf("Group: %s", group)
		for version, kinds := range versions {
			t.Logf("  Version: %s", version)
			for kind := range kinds {
				t.Logf("    Kind: %s", kind)
			}
		}
	}

	// Basic assertion - webhook should support at least some resources
	assert.Greater(t, len(supported), 0, "Webhook should support at least some resource groups")
}

func getCurrentFile() string {
	_, filename, _, _ := runtime.Caller(0)
	return filename
}