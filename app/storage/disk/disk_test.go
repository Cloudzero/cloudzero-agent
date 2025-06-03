// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package disk_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	config "github.com/cloudzero/cloudzero-agent/app/config/gator"
	"github.com/cloudzero/cloudzero-agent/app/storage/disk"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/cloudzero/cloudzero-agent/app/types/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiskStore_PutAndPending(t *testing.T) {
	dirPath := t.TempDir()
	rowLimit := 10

	ps, err := disk.NewDiskStore(config.Database{StoragePath: dirPath, MaxRecords: rowLimit}, disk.WithContentIdentifier(disk.CostContentIdentifier))
	assert.NoError(t, err)
	defer ps.Flush()

	initialTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	mockClock := mocks.NewMockClock(initialTime)

	// Add metrics less than the row limit
	metric := types.Metric{
		ID:             uuid.New(),
		ClusterName:    "cluster",
		CloudAccountID: "cloudaccount",
		MetricName:     "test_metric",
		NodeName:       "node1",
		CreatedAt:      mockClock.GetCurrentTime(),
		TimeStamp:      mockClock.GetCurrentTime(),
		Labels:         map[string]string{"label": "test"},
		Value:          "123.45",
	}
	err = ps.Put(context.Background(), metric, metric, metric)
	assert.NoError(t, err)

	// Verify Pending returns the correct buffered count
	assert.Equal(t, 3, ps.Pending())

	// Add more metrics but still below row limit
	err = ps.Put(context.Background(), metric, metric)
	assert.NoError(t, err)

	// Confirm Pending count reflects all metrics added
	assert.Equal(t, 5, ps.Pending())
}

func TestDiskStore_Flush(t *testing.T) {
	dirPath := t.TempDir()
	rowLimit := 5

	ps, err := disk.NewDiskStore(config.Database{StoragePath: dirPath, MaxRecords: rowLimit}, disk.WithContentIdentifier(disk.CostContentIdentifier))
	assert.NoError(t, err)

	initialTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	mockClock := mocks.NewMockClock(initialTime)

	// Add metrics and verify they are pending
	metric := types.Metric{
		ID:             uuid.New(),
		ClusterName:    "cluster",
		CloudAccountID: "cloudaccount",
		MetricName:     "test_metric",
		NodeName:       "node1",
		CreatedAt:      mockClock.GetCurrentTime(),
		TimeStamp:      mockClock.GetCurrentTime(),
		Labels:         map[string]string{"label": "test"},
		Value:          "123.45",
	}
	err = ps.Put(context.Background(), metric, metric)
	assert.NoError(t, err)
	assert.Equal(t, 2, ps.Pending())

	// Call Flush to write all pending data to disk
	err = ps.Flush()
	assert.NoError(t, err)

	// Verify that all pending data has been written
	assert.Equal(t, 0, ps.Pending())
}

func TestDiskStore_FlushTimeout(t *testing.T) {
	dirPath := t.TempDir()
	rowLimit := 5

	ps, err := disk.NewDiskStore(config.Database{StoragePath: dirPath, MaxRecords: rowLimit}, disk.WithMaxInterval(50*time.Millisecond))
	assert.NoError(t, err)

	initialTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	mockClock := mocks.NewMockClock(initialTime)

	// Add metrics and verify they are pending
	metric := types.Metric{
		ID:             uuid.New(),
		ClusterName:    "cluster",
		CloudAccountID: "cloudaccount",
		MetricName:     "test_metric",
		NodeName:       "node1",
		CreatedAt:      mockClock.GetCurrentTime(),
		TimeStamp:      mockClock.GetCurrentTime(),
		Labels:         map[string]string{"label": "test"},
		Value:          "123.45",
	}
	err = ps.Put(context.Background(), metric, metric)
	assert.NoError(t, err)

	// Wait for the flush to complete
	time.Sleep(100 * time.Millisecond)

	// Verify that all pending data has been written
	assert.Equal(t, 0, ps.Pending())
}

func TestDiskStore_Compact(t *testing.T) {
	// create a unique directory for each test
	dirPath, err := os.MkdirTemp(t.TempDir(), "TestDiskStore_Compact_")
	assert.NoError(t, err)
	ctx := context.Background()
	rowLimit := 100
	fileCount := 3
	recordCount := rowLimit * fileCount

	ps, err := disk.NewDiskStore(config.Database{StoragePath: dirPath, MaxRecords: rowLimit}, disk.WithContentIdentifier(disk.CostContentIdentifier))
	assert.NoError(t, err)
	defer ps.Flush()

	initialTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	mockClock := mocks.NewMockClock(initialTime)

	for i := 0; i < recordCount; i++ {
		id := fmt.Sprintf("test_metric_%d", i)
		value := fmt.Sprintf("%d", i)
		metric := types.Metric{
			ID:             uuid.New(),
			ClusterName:    "cluster",
			CloudAccountID: "cloudaccount",
			MetricName:     id,
			NodeName:       "node1",
			CreatedAt:      mockClock.GetCurrentTime(),
			TimeStamp:      mockClock.GetCurrentTime(),
			Labels:         map[string]string{"label": id},
			Value:          value,
		}
		err := ps.Put(ctx, metric)
		assert.NoError(t, err)
	}
	// give a moment to allow OS async operations to complete
	time.Sleep(1 * time.Second)

	discovered, err := ps.GetFiles()
	assert.NoError(t, err)
	assert.Equal(t, fileCount, len(discovered))

	for _, file := range discovered {
		metrics, err := ps.All(ctx, file)
		assert.NoError(t, err)
		assert.Len(t, metrics.Metrics, rowLimit)
	}
}

func TestDiskStore_GetFiles(t *testing.T) {
	// create a unique directory for each test
	dirPath, err := os.MkdirTemp(t.TempDir(), "TestDiskStore_MatchingFiles_")
	assert.NoError(t, err)
	ctx := context.Background()
	rowLimit := 100
	fileCount := 3
	recordCount := rowLimit * fileCount

	ps, err := disk.NewDiskStore(config.Database{StoragePath: dirPath, MaxRecords: rowLimit}, disk.WithContentIdentifier(disk.CostContentIdentifier))
	assert.NoError(t, err)
	defer ps.Flush()

	initialTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	mockClock := mocks.NewMockClock(initialTime)

	addRecords := func() {
		for i := 0; i < recordCount; i++ {
			id := fmt.Sprintf("test_metric_%d", i)
			value := fmt.Sprintf("%d", i)
			metric := types.Metric{
				ID:             uuid.New(),
				ClusterName:    "cluster",
				CloudAccountID: "cloudaccount",
				MetricName:     id,
				NodeName:       "node1",
				CreatedAt:      mockClock.GetCurrentTime(),
				TimeStamp:      mockClock.GetCurrentTime(),
				Labels:         map[string]string{"label": id},
				Value:          value,
			}
			err := ps.Put(ctx, metric)
			assert.NoError(t, err)
		}

		// give a moment to allow OS async operations to complete
		time.Sleep(1 * time.Second)
	}

	addRecords()

	// the `GetMatchingFiles` must respect the split between directories
	t.Run("TestDiskStore_GetFiles_EnsureSubdirectorySplit", func(t *testing.T) {
		files, err := ps.GetFiles()
		require.NoError(t, err)

		// move the files to a different directory
		err = os.Mkdir(filepath.Join(dirPath, "uploaded"), 0o755)
		require.NoError(t, err)
		for _, file := range files {
			newPath := filepath.Join(filepath.Dir(file), "uploaded", filepath.Base(file))
			err = os.Rename(file, newPath)
			require.NoError(t, err)
		}

		// ensure the root is empty
		res, err := ps.GetFiles()
		require.NoError(t, err)
		require.Empty(t, res)

		// ensure the new directory is not empty
		res, err = ps.GetFiles("uploaded")
		require.NoError(t, err)
		require.Equal(t, 3, len(res))

		// add more metrics
		addRecords()

		// ensure the root is not empty
		res, err = ps.GetFiles()
		require.NoError(t, err)
		require.Equal(t, 3, len(res))
	})
}

func TestDiskStore_GetUsage(t *testing.T) {
	tmpDir := t.TempDir()
	d, err := disk.NewDiskStore(config.Database{StoragePath: tmpDir, MaxRecords: 100}, disk.WithContentIdentifier(disk.CostContentIdentifier))
	require.NoError(t, err)
	defer d.Flush()

	_, err = d.GetUsage(0)
	require.NoError(t, err)
}

func TestDiskStore_Find(t *testing.T) {
	rowLimit := 10

	// note gotest will automatically clean up the temp directory after the test
	// no need to do it manually
	dir, err := os.MkdirTemp(t.TempDir(), "TestDiskStore_Find_")
	assert.NoError(t, err)

	// Create two more directories under dir
	subDir1 := filepath.Join(dir, "subdir1")
	err = os.Mkdir(subDir1, 0o755)
	assert.NoError(t, err)
	subDir2 := filepath.Join(dir, "subdir2")
	err = os.Mkdir(subDir2, 0o755)
	assert.NoError(t, err)

	// Write some files in the main directory and subdirectories
	for i := 0; i < rowLimit; i++ {
		filePath := filepath.Join(dir, fmt.Sprintf("metrics_%d.json.br", i))
		err = os.WriteFile(filePath, []byte(fmt.Sprintf(`{"metric": "test_metric_%d"}`, i)), 0o644)
		assert.NoError(t, err)

		filePathSub1 := filepath.Join(subDir1, fmt.Sprintf("metrics_sd1_%d.json.br", i))
		err = os.WriteFile(filePathSub1, []byte(fmt.Sprintf(`{"metric": "test_metric_sub1_%d"}`, i)), 0o644)
		assert.NoError(t, err)

		filePathSub2 := filepath.Join(subDir2, fmt.Sprintf("metrics_sd2_%d.json.br", i))
		err = os.WriteFile(filePathSub2, []byte(fmt.Sprintf(`{"metric": "test_metric_sub2_%d"}`, i)), 0o644)
		assert.NoError(t, err)
	}

	ps, err := disk.NewDiskStore(config.Database{StoragePath: dir, MaxRecords: rowLimit}, disk.WithContentIdentifier(disk.CostContentIdentifier))
	require.NoError(t, err)
	defer ps.Flush()

	ctx := context.Background()

	// Find All Files (no filter)
	files, err := ps.Find(ctx, "", "")
	require.NoError(t, err)
	require.Len(t, files, rowLimit*3+1) // 10 files in each of the 3 directories PLUS the file created by NewDiskStor

	// Find All Files (no extension only filter)
	files, err = ps.Find(ctx, "", ".json.br")
	require.NoError(t, err)
	require.Len(t, files, rowLimit*3) // still all files since we didn't filter by name
	require.Equal(t, rowLimit*3, len(files))

	// Find All Files (name filter only)
	files, err = ps.Find(ctx, "metrics_", "")
	require.NoError(t, err)
	require.Len(t, files, rowLimit*3) // still all files since we didn't filter by name
	require.Equal(t, rowLimit*3, len(files))

	// Find All Files (name filter and extension filter)
	files, err = ps.Find(ctx, "metrics_", ".json.br")
	require.NoError(t, err)
	require.Len(t, files, rowLimit*3) // still all files since we didn't filter by name
	require.Equal(t, rowLimit*3, len(files))

	// Find All Files (name filter and extension filter - sub-dir 1)
	files, err = ps.Find(ctx, "metrics_sd1_", ".json.br")
	require.NoError(t, err)
	require.Len(t, files, rowLimit) // only files matching the name and extension filter in sub-dir 1
	require.Equal(t, rowLimit, len(files))

	// Find Only 1 File (name filter and extension filter - sub-dir 1)
	files, err = ps.Find(ctx, "metrics_sd1_0", ".json.br")
	require.NoError(t, err)
	require.Len(t, files, 1) // only one file matching the name and extension filter in sub-dir 1
	require.Equal(t, 1, len(files))

	// Find All Files (name filter and extension filter - sub-dir 2)
	files, err = ps.Find(ctx, "metrics_sd2_", ".json.br")
	require.NoError(t, err)
	require.Len(t, files, rowLimit) // only files matching the name and extension filter in sub-dir 2
	require.Equal(t, rowLimit, len(files))

	// Find non-existent file
	files, err = ps.Find(ctx, "non_existent_file", ".json.br")
	require.NoError(t, err)
	require.Len(t, files, 0) // no files should match
}
