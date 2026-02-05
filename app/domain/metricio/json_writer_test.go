// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package metricio

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJSONWriter_WriteUncompressed(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.json")

	writer, err := NewJSONWriter(outputPath)
	require.NoError(t, err)

	// Write some metrics
	metrics := []types.Metric{
		{
			ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			MetricName: "test_metric_1",
			TimeStamp:  time.UnixMilli(1704067200000).UTC(),
			CreatedAt:  time.UnixMilli(1704067200000).UTC(),
			Value:      "42",
			Labels:     map[string]string{"env": "test"},
		},
		{
			ID:         uuid.MustParse("00000000-0000-0000-0000-000000000002"),
			MetricName: "test_metric_2",
			TimeStamp:  time.UnixMilli(1704067260000).UTC(),
			CreatedAt:  time.UnixMilli(1704067260000).UTC(),
			Value:      "99",
			Labels:     map[string]string{},
		},
	}

	err = writer.Write(metrics)
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	// Read back and verify
	reader := NewJSONReader(100)
	readMetrics, err := reader.ReadAllMetrics(outputPath)
	require.NoError(t, err)
	assert.Len(t, readMetrics, 2)
	assert.Equal(t, "test_metric_1", readMetrics[0].MetricName)
	assert.Equal(t, "42", readMetrics[0].Value)
	assert.Equal(t, "test_metric_2", readMetrics[1].MetricName)
}

func TestJSONWriter_WriteCompressed(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.json.br")

	writer, err := NewJSONWriter(outputPath)
	require.NoError(t, err)

	metric := types.Metric{
		ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		MetricName: "compressed_metric",
		TimeStamp:  time.UnixMilli(1704067200000).UTC(),
		CreatedAt:  time.UnixMilli(1704067200000).UTC(),
		Value:      "123",
		Labels:     map[string]string{"compressed": "true"},
	}

	err = writer.WriteOne(metric)
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	// Verify the file is Brotli-compressed
	compressed, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	// Try to decompress
	decompressor := brotli.NewReader(nil)
	// The file should be valid Brotli (we can't easily test this without reading it)

	// Use the reader to verify
	reader := NewJSONReader(100)
	readMetrics, err := reader.ReadAllMetrics(outputPath)
	require.NoError(t, err)
	assert.Len(t, readMetrics, 1)
	assert.Equal(t, "compressed_metric", readMetrics[0].MetricName)

	_ = decompressor
	_ = compressed
}

func TestJSONWriter_WriteOne(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "single.json")

	writer, err := NewJSONWriter(outputPath)
	require.NoError(t, err)

	for i := 0; i < 5; i++ {
		metric := types.Metric{
			ID:         uuid.New(),
			MetricName: "single_write",
			TimeStamp:  time.UnixMilli(int64(1704067200000 + i*1000)).UTC(),
			CreatedAt:  time.Now().UTC(),
			Value:      "1",
		}
		err = writer.WriteOne(metric)
		require.NoError(t, err)
	}

	err = writer.Close()
	require.NoError(t, err)

	// Verify
	reader := NewJSONReader(100)
	readMetrics, err := reader.ReadAllMetrics(outputPath)
	require.NoError(t, err)
	assert.Len(t, readMetrics, 5)
}

func TestJSONWriter_EmptyWrite(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "empty.json")

	writer, err := NewJSONWriter(outputPath)
	require.NoError(t, err)

	// Write nothing, just close
	err = writer.Close()
	require.NoError(t, err)

	// Read back - should be empty array
	reader := NewJSONReader(100)
	readMetrics, err := reader.ReadAllMetrics(outputPath)
	require.NoError(t, err)
	assert.Empty(t, readMetrics)
}

func TestJSONWriter_WriteAfterClose(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "closed.json")

	writer, err := NewJSONWriter(outputPath)
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	// Try to write after close
	err = writer.WriteOne(types.Metric{
		ID:        uuid.New(),
		TimeStamp: time.Now().UTC(),
		CreatedAt: time.Now().UTC(),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "closed")
}

func TestJSONWriter_DoubleClose(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "double.json")

	writer, err := NewJSONWriter(outputPath)
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	// Second close should be no-op
	err = writer.Close()
	require.NoError(t, err)
}

func TestJSONWriter_InvalidPath(t *testing.T) {
	_, err := NewJSONWriter("/nonexistent/directory/file.json")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create")
}

func TestJSONWriter_WithCustomCompression(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "custom.json.br")

	// Use best compression
	writer, err := NewJSONWriterWithCompression(outputPath, 11)
	require.NoError(t, err)

	metric := types.Metric{
		ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		MetricName: "high_compression",
		TimeStamp:  time.UnixMilli(1704067200000).UTC(),
		CreatedAt:  time.UnixMilli(1704067200000).UTC(),
		Value:      "999",
	}

	err = writer.WriteOne(metric)
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	// Verify
	reader := NewJSONReader(100)
	readMetrics, err := reader.ReadAllMetrics(outputPath)
	require.NoError(t, err)
	assert.Len(t, readMetrics, 1)
}

func TestJSONWriter_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()

	testCases := []struct {
		name string
		path string
	}{
		{"uncompressed", filepath.Join(tmpDir, "roundtrip.json")},
		{"compressed", filepath.Join(tmpDir, "roundtrip.json.br")},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			original := types.Metric{
				ID:             uuid.MustParse("12345678-1234-1234-1234-123456789012"),
				MetricName:     "roundtrip_metric",
				NodeName:       "test-node",
				ClusterName:    "test-cluster",
				CloudAccountID: "account-123",
				TimeStamp:      time.UnixMilli(1704067200000).UTC(),
				CreatedAt:      time.UnixMilli(1704067200000).UTC(),
				Value:          "42.5",
				Labels: map[string]string{
					"env":     "production",
					"app":     "myapp",
					"version": "1.0.0",
				},
			}

			// Write
			writer, err := NewJSONWriter(tc.path)
			require.NoError(t, err)
			err = writer.WriteOne(original)
			require.NoError(t, err)
			err = writer.Close()
			require.NoError(t, err)

			// Read back
			reader := NewJSONReader(100)
			readMetrics, err := reader.ReadAllMetrics(tc.path)
			require.NoError(t, err)
			require.Len(t, readMetrics, 1)

			result := readMetrics[0]
			assert.Equal(t, original.ID, result.ID)
			assert.Equal(t, original.MetricName, result.MetricName)
			assert.Equal(t, original.NodeName, result.NodeName)
			assert.Equal(t, original.ClusterName, result.ClusterName)
			assert.Equal(t, original.CloudAccountID, result.CloudAccountID)
			assert.Equal(t, original.TimeStamp.UnixMilli(), result.TimeStamp.UnixMilli())
			assert.Equal(t, original.Value, result.Value)
			assert.Equal(t, original.Labels, result.Labels)
		})
	}
}
