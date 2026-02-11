// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package metricio

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewParquetReader(t *testing.T) {
	tests := []struct {
		name      string
		batchSize int
	}{
		{"positive batch size", 100},
		{"zero batch size defaults", 0},
		{"negative batch size defaults", -10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := NewParquetReader(tt.batchSize)
			assert.NotNil(t, reader)
		})
	}
}

func TestParquetReader_ReadFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test parquet file
	outputPath := filepath.Join(tmpDir, "test.parquet")
	writer, err := NewParquetWriter(outputPath)
	require.NoError(t, err)

	metrics := []types.Metric{
		{
			ID:             uuid.New(),
			MetricName:     "metric1",
			NodeName:       "node1",
			ClusterName:    "cluster1",
			CloudAccountID: "account1",
			TimeStamp:      time.UnixMilli(1704067200000).UTC(),
			CreatedAt:      time.UnixMilli(1704067200000).UTC(),
			Value:          "10",
			Labels:         map[string]string{"env": "test"},
		},
		{
			ID:             uuid.New(),
			MetricName:     "metric2",
			NodeName:       "node2",
			ClusterName:    "cluster1",
			CloudAccountID: "account1",
			TimeStamp:      time.UnixMilli(1704067260000).UTC(),
			CreatedAt:      time.UnixMilli(1704067260000).UTC(),
			Value:          "20",
			Labels:         map[string]string{},
		},
	}

	err = writer.Write(metrics)
	require.NoError(t, err)
	err = writer.Close()
	require.NoError(t, err)

	// Read back
	reader := NewParquetReader(100)
	var readMetrics []types.ParquetMetric
	err = reader.ReadFile(outputPath, func(batch []types.ParquetMetric) error {
		readMetrics = append(readMetrics, batch...)
		return nil
	})

	require.NoError(t, err)
	assert.Len(t, readMetrics, 2)
	assert.Equal(t, "metric1", readMetrics[0].MetricName)
	assert.Equal(t, "metric2", readMetrics[1].MetricName)
}

func TestParquetReader_ReadFile_FileNotFound(t *testing.T) {
	reader := NewParquetReader(100)

	err := reader.ReadFile("/nonexistent/file.parquet", func(batch []types.ParquetMetric) error {
		return nil
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open")
}

func TestParquetReader_ReadFile_CallbackError(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a test parquet file
	outputPath := filepath.Join(tmpDir, "test.parquet")
	writer, err := NewParquetWriter(outputPath)
	require.NoError(t, err)
	err = writer.WriteOne(types.Metric{
		ID:        uuid.New(),
		TimeStamp: time.Now().UTC(),
		CreatedAt: time.Now().UTC(),
	})
	require.NoError(t, err)
	err = writer.Close()
	require.NoError(t, err)

	reader := NewParquetReader(100)
	err = reader.ReadFile(outputPath, func(batch []types.ParquetMetric) error {
		return assert.AnError
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "callback error")
}

func TestFindFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create directory structure
	subDir1 := filepath.Join(tmpDir, "sub1")
	subDir2 := filepath.Join(tmpDir, "sub2")
	subSubDir := filepath.Join(subDir1, "nested")

	err := os.MkdirAll(subDir1, 0o755)
	require.NoError(t, err)
	err = os.MkdirAll(subDir2, 0o755)
	require.NoError(t, err)
	err = os.MkdirAll(subSubDir, 0o755)
	require.NoError(t, err)

	// Create test parquet files
	parquetFiles := []string{
		filepath.Join(tmpDir, "root.parquet"),
		filepath.Join(subDir1, "sub1.parquet"),
		filepath.Join(subDir2, "sub2.parquet"),
		filepath.Join(subSubDir, "nested.parquet"),
	}

	for _, f := range parquetFiles {
		writer, err := NewParquetWriter(f)
		require.NoError(t, err)
		err = writer.Close()
		require.NoError(t, err)
	}

	// Also create non-parquet files (some supported, some not)
	err = os.WriteFile(filepath.Join(tmpDir, "other.txt"), []byte("test"), 0o644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "data.csv"), []byte("col1,col2\n"), 0o644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(subDir1, "metrics.json"), []byte("[]"), 0o644)
	require.NoError(t, err)

	t.Run("find parquet only", func(t *testing.T) {
		found, err := FindFiles(tmpDir, func(path string) bool {
			return strings.HasSuffix(strings.ToLower(path), ".parquet")
		})
		require.NoError(t, err)
		assert.Len(t, found, 4)

		foundMap := make(map[string]bool)
		for _, f := range found {
			foundMap[f] = true
		}
		for _, expected := range parquetFiles {
			assert.True(t, foundMap[expected], "expected file not found: %s", expected)
		}
	})

	t.Run("find all supported files", func(t *testing.T) {
		found, err := FindSupportedFiles(tmpDir)
		require.NoError(t, err)
		assert.Len(t, found, 6) // 4 parquet + 1 csv + 1 json
	})

	t.Run("find all files with nil filter", func(t *testing.T) {
		found, err := FindFiles(tmpDir, nil)
		require.NoError(t, err)
		assert.Len(t, found, 7) // 4 parquet + 1 csv + 1 json + 1 txt
	})
}

func TestFindFiles_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	found, err := FindSupportedFiles(tmpDir)
	require.NoError(t, err)
	assert.Empty(t, found)
}

func TestFindFiles_NonexistentDir(t *testing.T) {
	_, err := FindFiles("/nonexistent/directory", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read directory")
}

func TestIsSupportedFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"data.parquet", true},
		{"data.PARQUET", true},
		{"data.csv", true},
		{"data.CSV", true},
		{"data.json", true},
		{"data.JSON", true},
		{"data.json.br", true},
		{"data.JSON.BR", true},
		{"data.txt", false},
		{"data.parquet.gz", false},
		{"/path/to/metrics.parquet", true},
		{"/path/to/metrics.json.br", true},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			assert.Equal(t, tc.expected, IsSupportedFile(tc.path))
		})
	}
}

func TestParquetReader_ReadFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple parquet files
	paths := []string{
		filepath.Join(tmpDir, "file1.parquet"),
		filepath.Join(tmpDir, "file2.parquet"),
	}

	for i, p := range paths {
		writer, err := NewParquetWriter(p)
		require.NoError(t, err)
		err = writer.WriteOne(types.Metric{
			ID:         uuid.New(),
			MetricName: "metric_" + string(rune('A'+i)),
			TimeStamp:  time.Now().UTC(),
			CreatedAt:  time.Now().UTC(),
			Value:      "1",
		})
		require.NoError(t, err)
		err = writer.Close()
		require.NoError(t, err)
	}

	// Read all files
	reader := NewParquetReader(100)
	var results []struct {
		path    string
		metrics []types.ParquetMetric
	}

	err := reader.ReadFiles(paths, func(path string, metrics []types.ParquetMetric) error {
		results = append(results, struct {
			path    string
			metrics []types.ParquetMetric
		}{path, metrics})
		return nil
	})

	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestParquetReader_ReadFiles_EmptyList(t *testing.T) {
	reader := NewParquetReader(100)

	err := reader.ReadFiles([]string{}, func(path string, metrics []types.ParquetMetric) error {
		return nil
	})

	require.NoError(t, err)
}

func TestParquetReader_ReadFiles_CallbackError(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a parquet file
	path := filepath.Join(tmpDir, "test.parquet")
	writer, err := NewParquetWriter(path)
	require.NoError(t, err)
	err = writer.WriteOne(types.Metric{
		ID:        uuid.New(),
		TimeStamp: time.Now().UTC(),
		CreatedAt: time.Now().UTC(),
	})
	require.NoError(t, err)
	err = writer.Close()
	require.NoError(t, err)

	reader := NewParquetReader(100)
	err = reader.ReadFiles([]string{path}, func(path string, metrics []types.ParquetMetric) error {
		return assert.AnError
	})

	require.Error(t, err)
}
