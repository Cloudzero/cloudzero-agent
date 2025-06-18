// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestRunWithExtractorIntegration(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	// Create test configured values file
	configuredValues := map[string]interface{}{
		"replicas": 5,
		"image":    "nginx:latest",
		"config": map[string]interface{}{
			"database": map[string]interface{}{
				"host": "custom.db.com",
				"port": 5432,
			},
		},
		"kubeStateMetrics": map[string]interface{}{
			"enabled": true,
		},
	}

	configuredFile := filepath.Join(tempDir, "configured.yaml")
	writeTestYAML(t, configuredFile, configuredValues)

	// Create test default values file
	defaultValues := map[string]interface{}{
		"replicas": 3,
		"image":    "nginx:latest",
		"config": map[string]interface{}{
			"database": map[string]interface{}{
				"host": "localhost",
				"port": 5432,
			},
		},
		"kubeStateMetrics": map[string]interface{}{
			"enabled": false,
		},
	}

	defaultsFile := filepath.Join(tempDir, "defaults.yaml")
	writeTestYAML(t, defaultsFile, defaultValues)

	// Create output file
	outputFile := filepath.Join(tempDir, "output.yaml")
	output, err := os.Create(outputFile)
	require.NoError(t, err)

	// Create config and run
	config := Config{
		ConfiguredValuesPath: configuredFile,
		DefaultValuesPath:    defaultsFile,
		OutputPath:           output,
	}

	err = run(config)
	require.NoError(t, err)

	// Read and parse output file
	outputData, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	var result map[string]interface{}
	err = yaml.Unmarshal(outputData, &result)
	require.NoError(t, err)

	// Verify expected overrides (kubeStateMetrics should be excluded)
	expected := map[string]interface{}{
		"replicas": 5,
		"config": map[string]interface{}{
			"database": map[string]interface{}{
				"host": "custom.db.com",
			},
		},
	}

	assert.Equal(t, expected, result)
}

func TestRunWithEmbeddedDefaults(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	// Create simple configured values file
	configuredValues := map[string]interface{}{
		"replicas":  10,
		"customKey": "customValue",
	}

	configuredFile := filepath.Join(tempDir, "configured.yaml")
	writeTestYAML(t, configuredFile, configuredValues)

	// Create output file
	outputFile := filepath.Join(tempDir, "output.yaml")
	output, err := os.Create(outputFile)
	require.NoError(t, err)

	// Create config with no defaults file (should use embedded defaults)
	config := Config{
		ConfiguredValuesPath: configuredFile,
		DefaultValuesPath:    "",
		OutputPath:           output,
	}

	err = run(config)
	require.NoError(t, err)

	// Read and parse output file
	outputData, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	var result map[string]interface{}
	err = yaml.Unmarshal(outputData, &result)
	require.NoError(t, err)

	// Should have some overrides (exact content depends on embedded defaults)
	assert.NotEmpty(t, result)
}

// Helper function to write test YAML
func writeTestYAML(t *testing.T, path string, data interface{}) {
	file, err := os.Create(path)
	require.NoError(t, err)
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	defer encoder.Close()
	encoder.SetIndent(2)

	err = encoder.Encode(data)
	require.NoError(t, err)
}
