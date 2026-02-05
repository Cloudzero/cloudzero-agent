// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package metricio

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParquetWriter_Write(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.parquet")

	writer, err := NewParquetWriter(outputPath)
	require.NoError(t, err)

	// Write some metrics
	metrics := []types.Metric{
		{
			ID:             uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			MetricName:     "test_metric_1",
			NodeName:       "node-1",
			ClusterName:    "cluster-1",
			CloudAccountID: "account-1",
			TimeStamp:      time.UnixMilli(1704067200000).UTC(),
			CreatedAt:      time.UnixMilli(1704067200000).UTC(),
			Value:          "42",
			Labels:         map[string]string{"env": "test"},
		},
		{
			ID:             uuid.MustParse("00000000-0000-0000-0000-000000000002"),
			MetricName:     "test_metric_2",
			NodeName:       "node-2",
			ClusterName:    "cluster-1",
			CloudAccountID: "account-1",
			TimeStamp:      time.UnixMilli(1704067260000).UTC(),
			CreatedAt:      time.UnixMilli(1704067260000).UTC(),
			Value:          "99",
			Labels:         map[string]string{},
		},
	}

	err = writer.Write(metrics)
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	// Read back and verify using ParquetReader
	reader := NewParquetReader(100)
	var readMetrics []types.ParquetMetric
	err = reader.ReadFile(outputPath, func(batch []types.ParquetMetric) error {
		readMetrics = append(readMetrics, batch...)
		return nil
	})
	require.NoError(t, err)
	assert.Len(t, readMetrics, 2)
	assert.Equal(t, "test_metric_1", readMetrics[0].MetricName)
	assert.Equal(t, "42", readMetrics[0].Value)
	assert.Equal(t, "test_metric_2", readMetrics[1].MetricName)
}

func TestParquetWriter_WriteOne(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "single.parquet")

	writer, err := NewParquetWriter(outputPath)
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
	reader := NewParquetReader(100)
	var readMetrics []types.ParquetMetric
	err = reader.ReadFile(outputPath, func(batch []types.ParquetMetric) error {
		readMetrics = append(readMetrics, batch...)
		return nil
	})
	require.NoError(t, err)
	assert.Len(t, readMetrics, 5)
}

func TestParquetWriter_WriteParquetMetrics(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "direct.parquet")

	writer, err := NewParquetWriter(outputPath)
	require.NoError(t, err)

	// Write ParquetMetric directly (no conversion)
	parquetMetrics := []types.ParquetMetric{
		{
			MetricName:     "direct_metric",
			NodeName:       "node-1",
			ClusterName:    "cluster-1",
			CloudAccountID: "account-1",
			Year:           "2024",
			Month:          "01",
			Day:            "01",
			Hour:           "12",
			TimeStamp:      1704067200000,
			CreatedAt:      1704067200000,
			Value:          "100",
			Labels:         `{"direct":"true"}`,
		},
	}

	err = writer.WriteParquetMetrics(parquetMetrics)
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	// Verify
	reader := NewParquetReader(100)
	var readMetrics []types.ParquetMetric
	err = reader.ReadFile(outputPath, func(batch []types.ParquetMetric) error {
		readMetrics = append(readMetrics, batch...)
		return nil
	})
	require.NoError(t, err)
	assert.Len(t, readMetrics, 1)
	assert.Equal(t, "direct_metric", readMetrics[0].MetricName)
	assert.Equal(t, "2024", readMetrics[0].Year)
}

func TestParquetWriter_EmptyWrite(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "empty.parquet")

	writer, err := NewParquetWriter(outputPath)
	require.NoError(t, err)

	// Write nothing, just close
	err = writer.Close()
	require.NoError(t, err)

	// Read back - should be empty
	reader := NewParquetReader(100)
	var readMetrics []types.ParquetMetric
	err = reader.ReadFile(outputPath, func(batch []types.ParquetMetric) error {
		readMetrics = append(readMetrics, batch...)
		return nil
	})
	require.NoError(t, err)
	assert.Empty(t, readMetrics)
}

func TestParquetWriter_WriteAfterClose(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "closed.parquet")

	writer, err := NewParquetWriter(outputPath)
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

func TestParquetWriter_DoubleClose(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "double.parquet")

	writer, err := NewParquetWriter(outputPath)
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	// Second close should be no-op
	err = writer.Close()
	require.NoError(t, err)
}

func TestParquetWriter_InvalidPath(t *testing.T) {
	_, err := NewParquetWriter("/nonexistent/directory/file.parquet")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create")
}

func TestParquetWriter_LargeBatch(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "large.parquet")

	writer, err := NewParquetWriter(outputPath)
	require.NoError(t, err)

	// Write 10,000 metrics
	metrics := make([]types.Metric, 10000)
	for i := 0; i < 10000; i++ {
		metrics[i] = types.Metric{
			ID:         uuid.New(),
			MetricName: "large_batch_metric",
			TimeStamp:  time.UnixMilli(int64(1704067200000 + i*1000)).UTC(),
			CreatedAt:  time.Now().UTC(),
			Value:      "1",
			Labels:     map[string]string{"index": string(rune(i % 100))},
		}
	}

	err = writer.Write(metrics)
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	// Verify count
	reader := NewParquetReader(1000)
	var count int
	err = reader.ReadFile(outputPath, func(batch []types.ParquetMetric) error {
		count += len(batch)
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 10000, count)
}

func TestParquetWriter_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "roundtrip.parquet")

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
	writer, err := NewParquetWriter(outputPath)
	require.NoError(t, err)
	err = writer.WriteOne(original)
	require.NoError(t, err)
	err = writer.Close()
	require.NoError(t, err)

	// Read back
	reader := NewParquetReader(100)
	var readParquet []types.ParquetMetric
	err = reader.ReadFile(outputPath, func(batch []types.ParquetMetric) error {
		readParquet = append(readParquet, batch...)
		return nil
	})
	require.NoError(t, err)
	require.Len(t, readParquet, 1)

	// Convert back to Metric
	result := readParquet[0].Metric()

	assert.Equal(t, original.MetricName, result.MetricName)
	assert.Equal(t, original.NodeName, result.NodeName)
	assert.Equal(t, original.ClusterName, result.ClusterName)
	assert.Equal(t, original.CloudAccountID, result.CloudAccountID)
	assert.Equal(t, original.TimeStamp.UnixMilli(), result.TimeStamp.UnixMilli())
	assert.Equal(t, original.Value, result.Value)
	assert.Equal(t, original.Labels["env"], result.Labels["env"])
	assert.Equal(t, original.Labels["app"], result.Labels["app"])
	assert.Equal(t, original.Labels["version"], result.Labels["version"])
}
