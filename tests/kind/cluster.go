// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package kind

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

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
