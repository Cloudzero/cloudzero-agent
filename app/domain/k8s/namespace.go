// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package k8s gives a unified interface for k8s information to be retrieved.
package k8s

import (
	"fmt"
	"os"
)

// EnvK8sNamespace Kubernetes namespace that the application is running inside
const EnvK8sNamespace = "K8S_NAMESPACE"

// GetNamespace gets the current kubernetes namespace if running in a pod.
//
// This is parsed from the `K8S_NAMESPACE` environment variable.
//
// This will return an error if the environment variable is not set or the
// value is empty.
func GetNamespace() (string, error) {
	ns, exists := os.LookupEnv(EnvK8sNamespace)
	if !exists {
		return "", fmt.Errorf("the env variable `%s` was not set", EnvK8sNamespace)
	}
	if ns == "" {
		return "", fmt.Errorf("the environment variable `%s` was empty", EnvK8sNamespace)
	}

	return ns, nil
}
