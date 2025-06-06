// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"fmt"
	"os"
)

// EnvK8sPodName Kubernetes pod name the application is running inside
const EnvK8sPodName = "K8S_POD_NAME"

// GetPodName gets the current kubernetes pod name this app is running in
//
// This is parsed from the `K8S_POD_NAME` environment variable.
//
// This will return an error if the environment variable is not set or the
// value is empty.
func GetPodName() (string, error) {
	podName, exists := os.LookupEnv(EnvK8sPodName)
	if !exists {
		return "", fmt.Errorf("the env variable `%s` was not set", EnvK8sPodName)
	}
	if podName == "" {
		return "", fmt.Errorf("the environment variable `%s` was empty", EnvK8sPodName)
	}

	return podName, nil
}
