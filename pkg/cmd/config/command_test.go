package config_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cloudzero/cloudzero-agent-validator/pkg/cmd/config"
)

func TestGenerate(t *testing.T) {
	values := map[string]interface{}{
		"ChartVerson":         "1.0.0",
		"AgentVersion":        "1.0.0",
		"AccountID":           "123456789",
		"ClusterName":         "test-cluster",
		"Region":              "us-west-2",
		"CloudzeroHost":       "https://cloudzero.com",
		"KubeStateMetricsURL": "http://kube-state-metrics.your-namespace.svc.cluster.local:8080",
		"PromNodeExporterURL": "http://node-exporter.your-namespace.svc.cluster.local:9100",
	}

	outputFile := "test_output.yml"

	err := config.Generate(values, outputFile)
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
