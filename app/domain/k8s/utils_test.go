// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"errors"
	"os"
	"strings"
	"testing"

	"k8s.io/client-go/rest"
)

func TestNewConfigProvider(t *testing.T) {
	provider := NewConfigProvider()
	if provider == nil {
		t.Fatal("expected non-nil ConfigProvider")
	}

	// Test that it implements the interface
	var _ ConfigProvider = provider
}

func TestConfigProvider_GetConfig_Interface(t *testing.T) {
	// Test that the interface works regardless of environment
	provider := NewConfigProvider()

	// This will either succeed (in-cluster) or fail (out-of-cluster)
	// We just verify the interface contract is met
	config, err := provider.GetConfig()

	if err == nil {
		// In-cluster scenario - config should be valid
		if config == nil {
			t.Fatal("expected non-nil config when in-cluster")
		}
	} else {
		// Out-of-cluster scenario - should get some error
		if config != nil {
			t.Fatal("expected nil config when error occurs")
		}
		if err.Error() == "" {
			t.Error("expected non-empty error message")
		}
	}
}

func TestConfigProvider_GetConfig_EnvVarOverride(t *testing.T) {
	// Test that KUBE_CONFIG_LOCATION environment variable is respected
	// This test only makes sense when we're out-of-cluster
	originalEnv := os.Getenv(EnvKubeConfigLocation)
	defer os.Setenv(EnvKubeConfigLocation, originalEnv)

	// Set a custom kubeconfig location that definitely doesn't exist
	customPath := "/definitely/does/not/exist/kubeconfig"
	os.Setenv(EnvKubeConfigLocation, customPath)

	provider := NewConfigProvider()

	// Try to get config - this will either succeed (in-cluster) or fail (out-of-cluster)
	config, err := provider.GetConfig()

	if err == nil {
		// In-cluster scenario - this is fine, test passes
		if config == nil {
			t.Fatal("expected non-nil config when in-cluster")
		}
		t.Log("running in-cluster, env var override not tested")
	} else {
		// Out-of-cluster scenario - should fail with our custom path
		if config != nil {
			t.Fatal("expected nil config when error occurs")
		}

		// The error should mention our custom path
		if !strings.Contains(err.Error(), customPath) {
			t.Errorf("expected error to mention custom path '%s', got '%s'", customPath, err.Error())
		}
	}
}

func TestGetConfig(t *testing.T) {
	// Test that the package-level GetConfig function works
	// This will either succeed (in-cluster) or fail (out-of-cluster)
	config, err := GetConfig()

	if err == nil {
		// In-cluster scenario - config should be valid
		if config == nil {
			t.Fatal("expected non-nil config when in-cluster")
		}
	} else {
		// Out-of-cluster scenario - should get some error
		if config != nil {
			t.Fatal("expected nil config when error occurs")
		}
		if err.Error() == "" {
			t.Error("expected non-empty error message")
		}
	}
}

func TestGetClient(t *testing.T) {
	// Test that GetClient properly handles GetConfig results
	client, err := GetClient()

	if err == nil {
		// In-cluster scenario - client should be valid
		if client == nil {
			t.Fatal("expected non-nil client when in-cluster")
		}
	} else {
		// Out-of-cluster scenario - should get some error
		if client != nil {
			t.Fatal("expected nil client when error occurs")
		}

		// Should contain the GetConfig error message
		expectedPrefix := "failed to get the k8s rest config:"
		if !strings.Contains(err.Error(), expectedPrefix) {
			t.Errorf("expected error to contain '%s', got '%s'", expectedPrefix, err.Error())
		}
	}
}

func TestConfigProvider_GetConfig_ErrNotInClusterHandling(t *testing.T) {
	// This test specifically tests the ErrNotInCluster error handling logic
	// We can't easily mock rest.InClusterConfig, but we can test the interface contract

	provider := NewConfigProvider()

	// The GetConfig method should always return either a valid config or an error
	// We don't assume which one based on the environment
	config, err := provider.GetConfig()

	if err == nil {
		// Success case - verify config is valid
		if config == nil {
			t.Fatal("expected non-nil config on success")
		}
	} else {
		// Error case - verify error is meaningful
		if config != nil {
			t.Fatal("expected nil config on error")
		}
		if err.Error() == "" {
			t.Error("expected non-empty error message")
		}

		// If it's ErrNotInCluster, that's expected behavior
		if errors.Is(err, rest.ErrNotInCluster) {
			t.Log("running out-of-cluster, ErrNotInCluster is expected")
		}
	}
}
