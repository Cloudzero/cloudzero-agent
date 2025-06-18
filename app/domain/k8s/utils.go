// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudzero/cloudzero-agent/app/domain/diagnostic/common"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// EnvKubeConfigLocation Location of a valid kube config
const EnvKubeConfigLocation = "KUBE_CONFIG_LOCATION"

// GetConfig returns a k8s config based on the environment
// detecting if we are on the prometheus pod or running
// on a machine with a kubeconfig file
//
// # If no config is found, an error will be thrown
//
// Optionally, you can set the env variable `KUBE_CONFIG_LOCATION` to set
// where the function looks for a valid kube config file
func GetConfig() (*rest.Config, error) {
	if common.InPod() {
		return rest.InClusterConfig()
	}

	// default location
	configLoc := filepath.Join(homedir.HomeDir(), ".kube", "config")

	// if an env variable was passed explicitely for the k8s config location, use that
	if loc, exists := os.LookupEnv(EnvKubeConfigLocation); exists {
		configLoc = loc
	}

	if _, err := os.Stat(configLoc); err != nil {
		return nil, fmt.Errorf("there is no k8s config file found at: '%s'", configLoc)
	}

	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
	return clientcmd.BuildConfigFromFlags("", kubeconfig)
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
