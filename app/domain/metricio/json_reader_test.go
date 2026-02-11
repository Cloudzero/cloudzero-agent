// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package metricio

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/andybalholm/brotli"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONReader_ReadFromReader(t *testing.T) {
	tests := []struct {
		name      string
		json      string
		batchSize int
		expected  int
		wantErr   bool
		errMsg    string
	}{
		{
			name:      "single metric",
			json:      `[{"id":"00000000-0000-0000-0000-000000000001","metric_name":"test","timestamp":"1704067200000","created_at":"1704067200000","value":"42"}]`,
			batchSize: 100,
			expected:  1,
		},
		{
			name:      "multiple metrics",
			json:      `[{"id":"00000000-0000-0000-0000-000000000001","metric_name":"m1","timestamp":"1000","created_at":"1000","value":"1"},{"id":"00000000-0000-0000-0000-000000000002","metric_name":"m2","timestamp":"2000","created_at":"2000","value":"2"},{"id":"00000000-0000-0000-0000-000000000003","metric_name":"m3","timestamp":"3000","created_at":"3000","value":"3"}]`,
			batchSize: 100,
			expected:  3,
		},
		{
			name:      "empty array",
			json:      `[]`,
			batchSize: 100,
			expected:  0,
		},
		{
			name:      "batching",
			json:      `[{"id":"00000000-0000-0000-0000-000000000001","metric_name":"m1","timestamp":"1000","created_at":"1000","value":"1"},{"id":"00000000-0000-0000-0000-000000000002","metric_name":"m2","timestamp":"2000","created_at":"2000","value":"2"},{"id":"00000000-0000-0000-0000-000000000003","metric_name":"m3","timestamp":"3000","created_at":"3000","value":"3"}]`,
			batchSize: 2,
			expected:  3,
		},
		{
			name:      "invalid json - missing bracket",
			json:      `{"metric_name":"test"}`,
			batchSize: 100,
			wantErr:   true,
			errMsg:    "expected '['",
		},
		{
			name:      "invalid json - malformed",
			json:      `[{"metric_name":}]`,
			batchSize: 100,
			wantErr:   true,
			errMsg:    "failed to decode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := NewJSONReader(tt.batchSize)

			var metrics []types.Metric
			var batchCount int

			err := reader.ReadFromReader(strings.NewReader(tt.json), func(batch []types.Metric) error {
				metrics = append(metrics, batch...)
				batchCount++
				return nil
			})

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			require.NoError(t, err)
			assert.Len(t, metrics, tt.expected)
		})
	}
}

func TestJSONReader_ReadJSONFile(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create test JSON file
	jsonPath := filepath.Join(tmpDir, "test.json")
	jsonContent := `[
		{"id":"00000000-0000-0000-0000-000000000001","metric_name":"file_metric1","timestamp":"1704067200000","created_at":"1704067200000","value":"100","labels":{"env":"test"}},
		{"id":"00000000-0000-0000-0000-000000000002","metric_name":"file_metric2","timestamp":"1704067260000","created_at":"1704067260000","value":"200","labels":{}}
	]`
	err := os.WriteFile(jsonPath, []byte(jsonContent), 0o644)
	require.NoError(t, err)

	reader := NewJSONReader(100)

	var metrics []types.Metric
	err = reader.ReadJSONFile(jsonPath, func(batch []types.Metric) error {
		metrics = append(metrics, batch...)
		return nil
	})

	require.NoError(t, err)
	assert.Len(t, metrics, 2)
	assert.Equal(t, "file_metric1", metrics[0].MetricName)
	assert.Equal(t, "file_metric2", metrics[1].MetricName)
	assert.Equal(t, "100", metrics[0].Value)
	assert.Equal(t, "test", metrics[0].Labels["env"])
}

func TestJSONReader_ReadBrotliCompressed(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create test data
	jsonContent := `[{"id":"00000000-0000-0000-0000-000000000001","metric_name":"compressed_metric","timestamp":"1704067200000","created_at":"1704067200000","value":"42","labels":{"compressed":"true"}}]`

	// Compress with Brotli
	var compressed bytes.Buffer
	writer := brotli.NewWriter(&compressed)
	_, err := writer.Write([]byte(jsonContent))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	// Write to file with .br extension
	brPath := filepath.Join(tmpDir, "test.json.br")
	err = os.WriteFile(brPath, compressed.Bytes(), 0o644)
	require.NoError(t, err)

	reader := NewJSONReader(100)

	var metrics []types.Metric
	err = reader.ReadJSONFile(brPath, func(batch []types.Metric) error {
		metrics = append(metrics, batch...)
		return nil
	})

	require.NoError(t, err)
	assert.Len(t, metrics, 1)
	assert.Equal(t, "compressed_metric", metrics[0].MetricName)
	assert.Equal(t, "42", metrics[0].Value)
	assert.Equal(t, "true", metrics[0].Labels["compressed"])
}

func TestJSONReader_ReadAllMetrics(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file
	jsonPath := filepath.Join(tmpDir, "test.json")
	jsonContent := `[
		{"id":"00000000-0000-0000-0000-000000000001","metric_name":"m1","timestamp":"1000","created_at":"1000","value":"1"},
		{"id":"00000000-0000-0000-0000-000000000002","metric_name":"m2","timestamp":"2000","created_at":"2000","value":"2"}
	]`
	err := os.WriteFile(jsonPath, []byte(jsonContent), 0o644)
	require.NoError(t, err)

	reader := NewJSONReader(100)

	metrics, err := reader.ReadAllMetrics(jsonPath)

	require.NoError(t, err)
	assert.Len(t, metrics, 2)
}

func TestJSONReader_FileNotFound(t *testing.T) {
	reader := NewJSONReader(100)

	err := reader.ReadJSONFile("/nonexistent/path/file.json", func(batch []types.Metric) error {
		return nil
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open")
}

func TestJSONReader_CallbackError(t *testing.T) {
	jsonContent := `[{"id":"00000000-0000-0000-0000-000000000001","metric_name":"test","timestamp":"1000","created_at":"1000","value":"1"}]`

	reader := NewJSONReader(100)

	err := reader.ReadFromReader(strings.NewReader(jsonContent), func(batch []types.Metric) error {
		return assert.AnError
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "callback error")
}

func TestNewJSONReader_DefaultBatchSize(t *testing.T) {
	reader := NewJSONReader(0)
	assert.NotNil(t, reader)
	// batchSize should be set to DefaultBatchSize
}

func TestNewJSONReader_NegativeBatchSize(t *testing.T) {
	reader := NewJSONReader(-5)
	assert.NotNil(t, reader)
	// batchSize should be set to DefaultBatchSize
}
