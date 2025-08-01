// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"fmt"

	"k8s.io/client-go/discovery"
)

// GetVersion returns the current kuberentes version when running inside
// of a pod
func GetVersion() (string, error) {
	cfg, err := GetConfig()
	if err != nil {
		return "", fmt.Errorf("failed to get the config: %w", err)
	}

	// TODO: Improve the HTTPMock to allow us to override the client
	// To Control the response

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return "", fmt.Errorf("failed to create the discovery client: %w", err)
	}

	information, err := discoveryClient.ServerVersion()
	if err != nil {
		return "", fmt.Errorf("failed to query the client version: %w", err)
	}

	return fmt.Sprintf("%s.%s", information.Major, information.Minor), nil
}
