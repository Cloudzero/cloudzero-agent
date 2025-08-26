// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// EnvKubeConfigLocation Location of a valid kube config
const EnvKubeConfigLocation = "KUBE_CONFIG_LOCATION"

// ConfigProvider defines the interface for Kubernetes configuration
type ConfigProvider interface {
	// GetConfig returns a Kubernetes rest.Config
	GetConfig() (*rest.Config, error)
}

// configProvider implements ConfigProvider using the standard approach
type configProvider struct{}

// NewConfigProvider creates a new ConfigProvider instance
func NewConfigProvider() ConfigProvider {
	return &configProvider{}
}

// GetConfig returns a k8s config based on the environment
// detecting if we are on the prometheus pod or running
// on a machine with a kubeconfig file
//
// # If no config is found, an error will be thrown
//
// Optionally, you can set the env variable `KUBE_CONFIG_LOCATION` to set
// where the function looks for a valid kube config file
func (p *configProvider) GetConfig() (*rest.Config, error) {
	// Try in-cluster config first
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	// Check if the error is something other than ErrNotInCluster
	if !errors.Is(err, rest.ErrNotInCluster) {
		return nil, fmt.Errorf("failed to get Kubernetes config: %w", err)
	}

	// Not in cluster, fall back to kubeconfig file
	// default location
	configLoc := filepath.Join(homedir.HomeDir(), ".kube", "config")

	// if an env variable was passed explicitly for the k8s config location, use that
	if loc, exists := os.LookupEnv(EnvKubeConfigLocation); exists {
		configLoc = loc
	}

	return clientcmd.BuildConfigFromFlags("", configLoc)
}

// GetConfig returns a k8s config based on the environment
// detecting if we are on the prometheus pod or running
// on a machine with a kubeconfig file
//
// # If no config is found, an error will be thrown
//
// Optionally, you can set the env variable `KUBE_CONFIG_LOCATION` to set
// where the function looks for a valid kube config file
func GetConfig() (*rest.Config, error) {
	provider := NewConfigProvider()
	return provider.GetConfig()
}

// GetClient creates a new k8s client to use
func GetClient() (*kubernetes.Clientset, error) {
	cfg, err := GetConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get the k8s rest config: %w", err)
	}

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create a k8s client: %w", err)
	}

	return client, nil
}
