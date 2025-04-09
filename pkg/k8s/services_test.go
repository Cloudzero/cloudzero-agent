// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package k8s_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/cloudzero/cloudzero-agent/pkg/k8s"
)

func TestGetKubeStateMetricsURLByName(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	ctx := context.TODO()
	namespace := "test-namespace"

	// Create a fake service in the test namespace with the name "kube-state-metrics"
	_, err := clientset.CoreV1().Services(namespace).Create(ctx, &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kube-state-metrics",
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port: 8080,
				},
			},
		},
	}, metav1.CreateOptions{})
	assert.NoError(t, err)

	kubeStateMetricsURL, err := k8s.GetKubeStateMetricsURL(ctx, clientset)
	assert.NoError(t, err)
	assert.Contains(t, kubeStateMetricsURL, "kube-state-metrics")
}

func TestGetKubeStateMetricsURLByLabel(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	ctx := context.TODO()
	namespace := "test-namespace"

	// Create a fake service in the test namespace with Helm-specific labels
	_, err := clientset.CoreV1().Services(namespace).Create(ctx, &corev1.Service{
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
				{
					Port: 8080,
				},
			},
		},
	}, metav1.CreateOptions{})
	assert.NoError(t, err)

	kubeStateMetricsURL, err := k8s.GetKubeStateMetricsURL(ctx, clientset)
	assert.NoError(t, err)
	assert.Contains(t, kubeStateMetricsURL, "custom-service-name")
}

func TestUpdateConfigMap(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	ctx := context.TODO()
	namespace := "test-namespace"

	// Create a ConfigMap
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-configmap",
			Namespace: namespace,
		},
		Data: map[string]string{
			"key1": "value1",
		},
	}
	_, err := clientset.CoreV1().ConfigMaps(namespace).Create(ctx, configMap, metav1.CreateOptions{})
	assert.NoError(t, err)

	// Update the ConfigMap
	newData := map[string]string{
		"key1": "new-value1",
		"key2": "value2",
	}
	err = k8s.UpdateConfigMap(ctx, clientset, namespace, "test-configmap", newData)
	assert.NoError(t, err)

	// Verify the ConfigMap was updated
	updatedConfigMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, "test-configmap", metav1.GetOptions{})
	assert.NoError(t, err)
	assert.Equal(t, newData, updatedConfigMap.Data)
}

func TestBuildKubeClient(t *testing.T) {
	// Create a temporary kubeconfig file
	kubeconfigContent := `
apiVersion: v1
clusters:
- cluster:
    server: https://127.0.0.1:6443
  name: test-cluster
contexts:
- context:
    cluster: test-cluster
    user: test-user
  name: test-context
current-context: test-context
kind: Config
preferences: {}
users:
- name: test-user
  user:
    token: test-token
`
	tmpfile, err := os.CreateTemp("", "kubeconfig")
	assert.NoError(t, err)
	defer os.Remove(tmpfile.Name()) // clean up

	_, err = tmpfile.Write([]byte(kubeconfigContent))
	assert.NoError(t, err)
	err = tmpfile.Close()
	assert.NoError(t, err)

	// Build the kube client using the temporary kubeconfig file
	clientset, err := k8s.BuildKubeClient(tmpfile.Name())
	assert.NoError(t, err)
	assert.NotNil(t, clientset)
}
