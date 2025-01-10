// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/cloudzero/cloudzero-agent-validator/pkg/cmd/config"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/k8s"
)

func TestGenerateByName(t *testing.T) {
	// Define the namespace to be used in the test
	namespace := "test-namespace"

	// Create a fake clientset with a service named "kube-state-metrics"
	clientset := fake.NewSimpleClientset(
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kube-state-metrics",
				Namespace: namespace,
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{
					{Port: 8080},
				},
			},
		},
	)

	ctx, _ := context.WithCancel(context.Background())

	// Fetch the Kube State Metrics URL
	kubeStateMetricsURL, err := k8s.GetKubeStateMetricsURL(ctx, clientset)
	assert.NoError(t, err)

	// Define the scrape config data
	scrapeConfigData := config.ScrapeConfigData{
		Targets:        []string{kubeStateMetricsURL},
		ClusterName:    "test-cluster",
		CloudAccountID: "123456789",
		Region:         "us-west-2",
		Host:           "test-host",
		SecretPath:     "/etc/config/prometheus/secrets/",
	}

	// Generate the configuration content
	configContent, err := config.Generate(scrapeConfigData)
	assert.NoError(t, err)
	assert.NotEmpty(t, configContent)

	// Validate the dynamically populated values
	assert.Contains(t, configContent, kubeStateMetricsURL)
	assert.Contains(t, configContent, "cluster_name=test-cluster")
	assert.Contains(t, configContent, "cloud_account_id=123456789")
	assert.Contains(t, configContent, "region=us-west-2")
	assert.Contains(t, configContent, "test-host")
	assert.Contains(t, configContent, "/etc/config/prometheus/secrets/")

	// Define the ConfigMap data
	configMapData := map[string]string{
		"prometheus.yml": configContent,
	}

	// Update the ConfigMap
	err = k8s.UpdateConfigMap(ctx, clientset, namespace, "test-configmap", configMapData)
	assert.NoError(t, err)

	// Verify the ConfigMap was updated
	updatedConfigMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, "test-configmap", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, configContent, updatedConfigMap.Data["prometheus.yml"])

	// Clean up the output file if it exists
	outputFile := "test_output.yml"
	if _, err := os.Stat(outputFile); err == nil {
		err = os.Remove(outputFile)
		assert.NoError(t, err)
	}
}

func TestGenerateByLabel(t *testing.T) {
	// Define the namespace to be used in the test
	namespace := "test-namespace"

	// Create a fake clientset with a service having Helm-specific labels
	clientset := fake.NewSimpleClientset(
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "custom-service-name",
				Namespace: namespace,
				Labels: map[string]string{
					"app.kubernetes.io/name": "kube-state-metrics",
					"helm.sh/chart":          "kube-state-metrics-2.11.1",
				},
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{
					{Port: 8080},
				},
			},
		},
	)

	ctx, _ := context.WithCancel(context.Background())

	// Fetch the Kube State Metrics URL
	kubeStateMetricsURL, err := k8s.GetKubeStateMetricsURL(ctx, clientset)
	assert.NoError(t, err)

	// Define the scrape config data
	scrapeConfigData := config.ScrapeConfigData{
		Targets:        []string{kubeStateMetricsURL},
		ClusterName:    "test-cluster",
		CloudAccountID: "123456789",
		Region:         "us-west-2",
		Host:           "test-host",
		SecretPath:     "/etc/config/prometheus/secrets/",
	}

	// Generate the configuration content
	configContent, err := config.Generate(scrapeConfigData)
	assert.NoError(t, err)
	assert.NotEmpty(t, configContent)

	// Validate the dynamically populated values
	assert.Contains(t, configContent, kubeStateMetricsURL)
	assert.Contains(t, configContent, "cluster_name=test-cluster")
	assert.Contains(t, configContent, "cloud_account_id=123456789")
	assert.Contains(t, configContent, "region=us-west-2")
	assert.Contains(t, configContent, "test-host")
	assert.Contains(t, configContent, "/etc/config/prometheus/secrets/")

	// Define the ConfigMap data
	configMapData := map[string]string{
		"prometheus.yml": configContent,
	}

	// Update the ConfigMap
	err = k8s.UpdateConfigMap(ctx, clientset, namespace, "test-configmap", configMapData)
	assert.NoError(t, err)

	// Verify the ConfigMap was updated
	updatedConfigMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, "test-configmap", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, configContent, updatedConfigMap.Data["prometheus.yml"])

	// Clean up the output file if it exists
	outputFile := "test_output.yml"
	if _, err := os.Stat(outputFile); err == nil {
		err = os.Remove(outputFile)
		assert.NoError(t, err)
	}
}
