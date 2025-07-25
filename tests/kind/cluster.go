// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package kind

import (
	_ "embed"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

//go:embed testdata/kind-config.yaml
var kindConfigTemplate string

// KindConfig holds the configuration parameters for the Kind cluster
type KindConfig struct {
	ClusterName string
	HostPath    string
}

// SetupClusterWithEmbeddedConfig creates a Kind cluster using the embedded configuration template
func SetupClusterWithEmbeddedConfig(t *testing.T, clusterName, tempDir string) string {
	t.Logf("Setting up Kind cluster %s...", clusterName)

	// Create Kind config from embedded template
	kindConfigPath := filepath.Join(tempDir, "kind-config.yaml")
	config := KindConfig{
		ClusterName: clusterName,
		HostPath:    tempDir,
	}

	tmpl, err := template.New("kind-config").Parse(kindConfigTemplate)
	require.NoError(t, err, "Failed to parse Kind config template")

	var configContent strings.Builder
	err = tmpl.Execute(&configContent, config)
	require.NoError(t, err, "Failed to execute Kind config template")

	err = os.WriteFile(kindConfigPath, []byte(configContent.String()), 0o644)
	require.NoError(t, err, "Failed to write Kind config file")

	return SetupCluster(t, clusterName, tempDir, kindConfigPath)
}

// SetupCluster creates a Kind cluster using the specified configuration file
func SetupCluster(t *testing.T, clusterName, tempDir, kindConfigPath string) string {
	t.Logf("Setting up Kind cluster %s...", clusterName)

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

// CleanupCluster deletes the Kind cluster
func CleanupCluster(t *testing.T, clusterName string) {
	t.Logf("Cleaning up Kind cluster %s...", clusterName)
	cmd := exec.Command("kind", "delete", "cluster", "--name", clusterName)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Logf("Warning: Failed to cleanup Kind cluster: %v", err)
	}
}

// CreateKubernetesClient creates a Kubernetes client from kubeconfig file
func CreateKubernetesClient(kubeconfig string) (kubernetes.Interface, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}

// GetCurrentFile returns the current file path (useful for finding test configuration files)
func GetCurrentFile() string {
	_, filename, _, _ := runtime.Caller(1)
	return filename
}
