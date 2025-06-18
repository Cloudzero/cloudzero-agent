// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// GetProviderID uses the detected k8s namespace and pod name to query
// the API to get the cloud provider id.
//
// This function requires setting both environment variables:
// `K8S_POD_NAME` and `K8S_NAMESPACE`
func GetProviderID(ctx context.Context) (string, error) {
	// ensure the correct env variables were injected
	ns, err := GetNamespace()
	if err != nil {
		return "", err
	}

	podName, err := GetPodName()
	if err != nil {
		return "", err
	}

	// create the k8s client
	cfg, err := GetConfig()
	if err != nil {
		return "", fmt.Errorf("failed to get the k8s client config: %w", err)
	}
	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return "", fmt.Errorf("failed to create a k8s client: %w", err)
	}

	// get the pod
	pod, err := client.CoreV1().Pods(ns).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to query the pod: %w", err)
	}

	// get the node
	node, err := client.CoreV1().Nodes().Get(ctx, pod.Spec.NodeName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get the node: %w", err)
	}

	return node.Spec.ProviderID, nil
}
