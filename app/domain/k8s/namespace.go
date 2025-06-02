// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"fmt"
	"os"
)

// Kubernetes namespace that the application is running inside
const ENV_K8S_NAMESPACE = "K8S_NAMESPACE"

// GetNamespace gets the current kubernetes namespace if running in a pod.
//
// This is parsed from the `K8S_NAMESPACE` environment variable.
//
// This will return an error if the environment variable is not set or the
// value is empty.
func GetNamespace() (string, error) {
	ns, exists := os.LookupEnv(ENV_K8S_NAMESPACE)
	if !exists {
		return "", fmt.Errorf("the env variable `%s` was not set", ENV_K8S_NAMESPACE)
	}
	if ns == "" {
		return "", fmt.Errorf("the environment variable `%s` was empty", ENV_K8S_NAMESPACE)
	}

	return ns, nil
}
