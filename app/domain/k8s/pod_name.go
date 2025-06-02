// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"fmt"
	"os"
)

// Kubernetes pod name the application is running inside
const ENV_K8S_POD_NAME = "K8S_POD_NAME"

// GetPodName gets the current kubernetes pod name this app is running in
//
// This is parsed from the `K8S_POD_NAME` environment variable.
//
// This will return an error if the environment variable is not set or the
// value is empty.
func GetPodName() (string, error) {
	podName, exists := os.LookupEnv(ENV_K8S_POD_NAME)
	if !exists {
		return "", fmt.Errorf("the env variable `%s` was not set", ENV_K8S_POD_NAME)
	}
	if podName == "" {
		return "", fmt.Errorf("the environment variable `%s` was empty", ENV_K8S_POD_NAME)
	}

	return podName, nil
}
