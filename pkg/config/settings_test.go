// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSettings(t *testing.T) {
	t.Run("valid config file", func(t *testing.T) {
		// Create a temporary config file
		configContent := `
api_key_path: "/path/to/api_key"
server:
  port: 8080
certificate:
  cert_file: "/path/to/cert"
  key_file: "/path/to/key"
logging:
  level: "info"
database:
  host: "localhost"
  port: 5432
filters:
  labels:
    patterns:
      - "label1"
  annotations:
    patterns:
      - "annotation1"
remote_write:
  max_bytes_per_send: 10000000
  send_interval: 60s
`
		configContentExtra := `
cloud_account_id: "123456789012"
region: "us-west-2"
cluster_name: "test-cluster"
host: "api.cloudzero.com"
`
		tmpFile, err := os.CreateTemp("", "config-*.yaml")
		require.NoError(t, err)
		tmpFileExtra, err := os.CreateTemp("", "config-extra-*.yaml")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())
		defer os.Remove(tmpFileExtra.Name())

		_, err = tmpFile.Write([]byte(configContent))
		require.NoError(t, err)
		_, err = tmpFileExtra.Write([]byte(configContentExtra))
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())

		// Mock the API key file
		apiKeyContent := "test-api-key"
		apiKeyFile, err := os.CreateTemp("", "api_key-*.txt")
		require.NoError(t, err)
		defer os.Remove(apiKeyFile.Name())

		_, err = apiKeyFile.Write([]byte(apiKeyContent))
		require.NoError(t, err)
		require.NoError(t, apiKeyFile.Close())

		// Update the API key path in the config
		configContent = strings.Replace(configContent, "/path/to/api_key", apiKeyFile.Name(), 1)
		err = os.WriteFile(tmpFile.Name(), []byte(configContent), 0o644)
		require.NoError(t, err)
		configFiles := Files{tmpFile.Name(), tmpFileExtra.Name()}
		settings, err := NewSettings(configFiles...)
		require.NoError(t, err)
		assert.NotNil(t, settings)
		assert.Equal(t, "123456789012", settings.CloudAccountID)
		assert.Equal(t, "us-west-2", settings.Region)
		assert.Equal(t, "test-cluster", settings.ClusterName)
		assert.Equal(t, "api.cloudzero.com", settings.Host)
		assert.Equal(t, apiKeyContent, settings.GetAPIKey())
		assert.Equal(t, "https://api.cloudzero.com/v1/container-metrics?cloud_account_id=123456789012&cluster_name=test-cluster&region=us-west-2", settings.RemoteWrite.Host)
		assert.Equal(t, 10000000, settings.RemoteWrite.MaxBytesPerSend)
		assert.Equal(t, 60*time.Second, settings.RemoteWrite.SendInterval)
	})

	t.Run("missing config file", func(t *testing.T) {
		settings, err := NewSettings("nonexistent.yaml")
		assert.Error(t, err)
		assert.Nil(t, settings)
	})

	t.Run("invalid config file", func(t *testing.T) {
		// Create an invalid config file
		tmpFile, err := os.CreateTemp("", "invalid-config-*.yaml")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.Write([]byte("invalid content"))
		require.NoError(t, err)
		require.NoError(t, tmpFile.Close())

		settings, err := NewSettings(tmpFile.Name())
		assert.Error(t, err)
		assert.Nil(t, settings)
	})
}

func TestCleanString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "string with leading and trailing spaces",
			input:    "  test-string  ",
			expected: "test-string",
		},
		{
			name:     "string with leading and trailing quotes",
			input:    "\"test-string\"",
			expected: "test-string",
		},
		{
			name:     "string with leading and trailing single quotes",
			input:    "'test-string'",
			expected: "test-string",
		},
		{
			name:     "string with mixed leading and trailing quotes",
			input:    "'test-string\"",
			expected: "test-string",
		},
		{
			name:     "string with no leading or trailing spaces or quotes",
			input:    "test-string",
			expected: "test-string",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "string with only spaces",
			input:    "   ",
			expected: "",
		},
		{
			name:     "string with only quotes",
			input:    "\"\"",
			expected: "",
		},
		{
			name:     "string with only single quotes",
			input:    "''",
			expected: "",
		},
		{
			name: "string with newlines, and other special characters",
			input: `
			"foobar"
`,
			expected: "foobar",
		},
		{
			name:     "string with backslashes",
			input:    "\\\"test-string\\",
			expected: "test-string",
		},
		{
			name:     "string with special characters in the middle",
			input:    "^ test-$triñg-123 $",
			expected: "test-$triñg-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cleanString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
