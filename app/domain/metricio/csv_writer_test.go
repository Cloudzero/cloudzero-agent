// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package metricio

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/google/uuid"
	"github.com/prometheus/prometheus/prompb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCSVWriter_Write(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "output.csv")

	writer, err := NewCSVWriter(outputPath)
	require.NoError(t, err)

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
	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	reader := csv.NewReader(bytes.NewReader(content))
	records, err := reader.ReadAll()
	require.NoError(t, err)

	// Header + 2 data rows
	assert.Len(t, records, 3)
	assert.Equal(t, []string{"USAGE_DATE", "VALUE", "LABELS"}, records[0])
	assert.Equal(t, "42", records[1][1])
	assert.Equal(t, "99", records[2][1])
}

func TestCSVWriter_WriteOne(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "single.csv")

	writer, err := NewCSVWriter(outputPath)
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
	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	reader := csv.NewReader(bytes.NewReader(content))
	records, err := reader.ReadAll()
	require.NoError(t, err)

	// Header + 5 data rows
	assert.Len(t, records, 6)
}

func TestCSVWriter_EmptyWrite(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "empty.csv")

	writer, err := NewCSVWriter(outputPath)
	require.NoError(t, err)

	// Write nothing, just close
	err = writer.Close()
	require.NoError(t, err)

	// Read back - should have only header
	content, err := os.ReadFile(outputPath)
	require.NoError(t, err)

	reader := csv.NewReader(bytes.NewReader(content))
	records, err := reader.ReadAll()
	require.NoError(t, err)

	assert.Len(t, records, 1)
	assert.Equal(t, []string{"USAGE_DATE", "VALUE", "LABELS"}, records[0])
}

func TestCSVWriter_WriteAfterClose(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "closed.csv")

	writer, err := NewCSVWriter(outputPath)
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

func TestCSVWriter_DoubleClose(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "double.csv")

	writer, err := NewCSVWriter(outputPath)
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	// Second close should be no-op
	err = writer.Close()
	require.NoError(t, err)
}

func TestCSVWriter_InvalidPath(t *testing.T) {
	_, err := NewCSVWriter("/nonexistent/directory/file.csv")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create")
}

func TestCSVWriter_ToWriter(t *testing.T) {
	var buf bytes.Buffer

	writer, err := NewCSVWriterToWriter(&buf)
	require.NoError(t, err)

	metric := types.Metric{
		ID:         uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		MetricName: "buffer_metric",
		NodeName:   "test-node",
		TimeStamp:  time.UnixMilli(1704067200000).UTC(),
		CreatedAt:  time.UnixMilli(1704067200000).UTC(),
		Value:      "123",
		Labels:     map[string]string{"key": "value"},
	}

	err = writer.WriteOne(metric)
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	// Verify buffer contents
	reader := csv.NewReader(&buf)
	records, err := reader.ReadAll()
	require.NoError(t, err)

	assert.Len(t, records, 2)
	assert.Equal(t, []string{"USAGE_DATE", "VALUE", "LABELS"}, records[0])
	assert.Equal(t, "123", records[1][1])

	// Verify labels JSON contains all labels including __name__ and node
	var labels map[string]string
	err = json.Unmarshal([]byte(records[1][2]), &labels)
	require.NoError(t, err)
	assert.Equal(t, "buffer_metric", labels["__name__"])
	assert.Equal(t, "test-node", labels["node"])
	assert.Equal(t, "value", labels["key"])
}

func TestCSVWriter_TimestampFormat(t *testing.T) {
	var buf bytes.Buffer

	writer, err := NewCSVWriterToWriter(&buf)
	require.NoError(t, err)

	// Use a specific timestamp
	ts := time.Date(2024, 1, 15, 10, 30, 45, 123000000, time.UTC)
	metric := types.Metric{
		ID:        uuid.New(),
		TimeStamp: ts,
		CreatedAt: ts,
		Value:     "1",
	}

	err = writer.WriteOne(metric)
	require.NoError(t, err)

	err = writer.Close()
	require.NoError(t, err)

	reader := csv.NewReader(&buf)
	records, err := reader.ReadAll()
	require.NoError(t, err)

	// Verify timestamp format: "2006-01-02 15:04:05.000 Z"
	assert.Equal(t, "2024-01-15 10:30:45.123 Z", records[1][0])
}

func TestCSVWriter_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "roundtrip.csv")

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
	writer, err := NewCSVWriter(outputPath)
	require.NoError(t, err)
	err = writer.WriteOne(original)
	require.NoError(t, err)
	err = writer.Close()
	require.NoError(t, err)

	// Read back using CSVReader
	reader := NewCSVReader(100)
	var readMetrics []types.Metric
	err = reader.ReadCSVFile(outputPath, func(ts []prompb.TimeSeries) error {
		for _, series := range ts {
			metrics := TimeSeriesToMetrics(series)
			readMetrics = append(readMetrics, metrics...)
		}
		return nil
	})
	require.NoError(t, err)
	require.Len(t, readMetrics, 1)

	result := readMetrics[0]
	// MetricName, NodeName preserved via __name__ and node labels
	assert.Equal(t, original.MetricName, result.MetricName)
	assert.Equal(t, original.NodeName, result.NodeName)
	// Note: ClusterName and CloudAccountID are NOT included in FullLabels()
	// and therefore don't round-trip through CSV format
	assert.Equal(t, original.TimeStamp.UnixMilli(), result.TimeStamp.UnixMilli())
	assert.Equal(t, original.Value, result.Value)
	assert.Equal(t, original.Labels["env"], result.Labels["env"])
	assert.Equal(t, original.Labels["app"], result.Labels["app"])
	assert.Equal(t, original.Labels["version"], result.Labels["version"])
}
