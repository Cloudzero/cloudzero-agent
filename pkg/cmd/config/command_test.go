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

func TestGenerate(t *testing.T) {
	// Create a fake clientset with some services and a ConfigMap
	clientset := fake.NewSimpleClientset(
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kube-state-metrics",
				Namespace: "default",
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{
					{Port: 8080},
				},
			},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "node-exporter",
				Namespace: "default",
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{
					{Port: 9100},
				},
			},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-configmap",
				Namespace: "default",
			},
			Data: map[string]string{
				"prometheus.kube_state_metrics_service_endpoint":       "old-url",
				"prometheus.prometheus_node_exporter_service_endpoint": "old-url",
			},
		},
	)

	ctx, _ := context.WithCancel(context.Background())

	// Fetch service URLs
	kubeStateMetricsURL, nodeExporterURL, err := k8s.GetServiceURLs(ctx, clientset)
	assert.NoError(t, err)

	// Fetch ConfigMap
	configMap, err := k8s.GetConfigMap(ctx, clientset, "default", "test-configmap")
	assert.NoError(t, err)

	// Update the ConfigMap data
	configMap.Data["prometheus.kube_state_metrics_service_endpoint"] = kubeStateMetricsURL
	configMap.Data["prometheus.prometheus_node_exporter_service_endpoint"] = nodeExporterURL

	err = k8s.UpdateConfigMap(ctx, clientset, "default", configMap)
	assert.NoError(t, err)

	// Verify the ConfigMap was updated
	updatedConfigMap, err := k8s.GetConfigMap(ctx, clientset, "default", "test-configmap")
	assert.NoError(t, err)
	assert.Equal(t, kubeStateMetricsURL, updatedConfigMap.Data["prometheus.kube_state_metrics_service_endpoint"])
	assert.Equal(t, nodeExporterURL, updatedConfigMap.Data["prometheus.prometheus_node_exporter_service_endpoint"])

	values := map[string]interface{}{
		"ChartVerson":         "1.0.0",
		"AgentVersion":        "1.0.0",
		"AccountID":           "123456789",
		"ClusterName":         "test-cluster",
		"Region":              "us-west-2",
		"CloudzeroHost":       "https://cloudzero.com",
		"KubeStateMetricsURL": kubeStateMetricsURL,
		"PromNodeExporterURL": nodeExporterURL,
	}

	outputFile := "test_output.yml"

	err = config.Generate(values, outputFile)
	assert.NoError(t, err)

	// Verify that the output file exists
	_, err = os.Stat(outputFile)
	assert.NoError(t, err)

	// Read the contents of the output file
	content, err := os.ReadFile(outputFile)
	assert.NoError(t, err)
	assert.NotEmpty(t, content)

	// TODO: Add assertions to validate the content of the output file

	// Clean up the output file
	err = os.Remove(outputFile)
	assert.NoError(t, err)
}
