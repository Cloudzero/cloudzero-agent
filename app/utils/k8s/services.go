// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package k8s contains helpers for working with the Kubernetes Client.
package k8s

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	EnvVarHostname   = "HOSTNAME"
	PodNamePartial   = "insights-controller-server"
	queriesPerSecond = 50
	maxBurst         = 100
)

// UpdateConfigMap updates the specified ConfigMap
func UpdateConfigMap(ctx context.Context, clientset kubernetes.Interface, namespace, name string, data map[string]string) error {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: data,
	}

	_, err := clientset.CoreV1().ConfigMaps(namespace).Update(ctx, configMap, metav1.UpdateOptions{})
	if err == nil {
		return nil
	}

	// If the ConfigMap does not exist, create it
	if k8serrors.IsNotFound(err) {
		_, createErr := clientset.CoreV1().ConfigMaps(namespace).Create(ctx, configMap, metav1.CreateOptions{})
		if createErr != nil {
			return errors.Wrap(createErr, "creating configmap")
		}
		return nil
	}

	return errors.Wrap(err, "updating configmap")
}

// NewClient creates a new Kubernetes client using the provided kubeconfig file path.
// It returns a kubernetes.Interface which can be used to interact with the Kubernetes API.
// The function sets the QPS (Queries Per Second) and Burst rate for the client to ensure efficient communication with the cluster.
// If there is an error building the kubeconfig or creating the clientset, it returns an error.
func NewClient(kubeconfigPath string) (kubernetes.Interface, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, errors.Wrap(err, "building kubeconfig")
	}
	config.QPS = queriesPerSecond
	config.Burst = maxBurst

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "building clientset")
	}
	return clientset, nil
}

// GetKubeStateMetricsURL fetches the URL for the Kube State Metrics service across all namespaces
func GetKubeStateMetricsURL(ctx context.Context, clientset kubernetes.Interface) (string, error) {
	// First, try to find the service by name
	services, err := clientset.CoreV1().Services("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", errors.Wrap(err, "listing services")
	}

	var kubeStateMetricsURL string

	for _, service := range services.Items {
		if strings.Contains(service.Name, "kube-state-metrics") {
			kubeStateMetricsURL = fmt.Sprintf("%s.%s.svc.cluster.local:%d", service.Name, service.Namespace, service.Spec.Ports[0].Port)
			return kubeStateMetricsURL, nil
		}
	}

	// If not found by name, check by labels
	for _, service := range services.Items {
		// Check for Helm-specific labels
		if service.Labels["app.kubernetes.io/name"] == "kube-state-metrics" &&
			service.Labels["helm.sh/chart"] != "" { // Ensure the service is managed by Helm
			kubeStateMetricsURL = fmt.Sprintf("%s.%s.svc.cluster.local:%d", service.Name, service.Namespace, service.Spec.Ports[0].Port)
			return kubeStateMetricsURL, nil
		}
	}

	if kubeStateMetricsURL == "" {
		return "", errors.New("kube-state-metrics service not found")
	}

	return kubeStateMetricsURL, nil
}
