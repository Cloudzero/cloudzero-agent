// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package store_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudzero/cloudzero-insights-controller/app/config"
	"github.com/cloudzero/cloudzero-insights-controller/app/store"
	"github.com/cloudzero/cloudzero-insights-controller/app/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiskStore_PutAndPending(t *testing.T) {
	dirPath := t.TempDir()
	rowLimit := 10

	ps, err := store.NewDiskStore(config.Database{StoragePath: dirPath, MaxRecords: rowLimit})
	assert.NoError(t, err)
	defer ps.Flush()

	// Add metrics less than the row limit
	metric := types.NewMetric("cloudaccount", "cluster", "test_metric", "node1", time.Now().Unix(), map[string]string{"label": "test"}, "123.45")
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

	ps, err := store.NewDiskStore(config.Database{StoragePath: dirPath, MaxRecords: rowLimit})
	assert.NoError(t, err)

	// Add metrics and verify they are pending
	metric := types.NewMetric("cloudaccount", "cluster", "test_metric", "node1", time.Now().Unix(), map[string]string{"label": "test"}, "123.45")
	err = ps.Put(context.Background(), metric, metric)
	assert.NoError(t, err)
	assert.Equal(t, 2, ps.Pending())

	// Call Flush to write all pending data to disk
	err = ps.Flush()
	assert.NoError(t, err)

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

	ps, err := store.NewDiskStore(config.Database{StoragePath: dirPath, MaxRecords: rowLimit})
	assert.NoError(t, err)
	defer ps.Flush()

	for i := 0; i < recordCount; i++ {
		id := fmt.Sprintf("test_metric_%d", i)
		value := fmt.Sprintf("%d", i)
		metric := types.NewMetric(
			"cloudaccount", "cluster",
			id,
			"node1",
			time.Now().Unix(),
			map[string]string{"label": id},
			value,
		)
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

func TestDiskStore_MatchingFiles(t *testing.T) {
	// create a unique directory for each test
	dirPath, err := os.MkdirTemp(t.TempDir(), "TestDiskStore_MatchingFiles_")
	assert.NoError(t, err)
	ctx := context.Background()
	rowLimit := 100
	fileCount := 3
	recordCount := rowLimit * fileCount

	ps, err := store.NewDiskStore(config.Database{StoragePath: dirPath, MaxRecords: rowLimit})
	assert.NoError(t, err)
	defer ps.Flush()

	addRecords := func() {
		for i := 0; i < recordCount; i++ {
			id := fmt.Sprintf("test_metric_%d", i)
			value := fmt.Sprintf("%d", i)
			metric := types.NewMetric(
				"cloudaccount", "cluster",
				id,
				"node1",
				time.Now().Unix(),
				map[string]string{"label": id},
				value,
			)
			err := ps.Put(ctx, metric)
			assert.NoError(t, err)
		}

		// give a moment to allow OS async operations to complete
		time.Sleep(1 * time.Second)
	}

	addRecords()

	// `GetMatchingFiles` must not return any files when no targets are defined
	t.Run("TestDiskStore_MatchingFiles_EmptyTargetFiles", func(t *testing.T) {
		files, err := ps.GetFiles()
		require.NoError(t, err)
		require.Equal(t, 3, len(files))

		files, err = ps.GetMatching("", []string{})
		require.NoError(t, err)
		require.Empty(t, files)
	})

	// the `GetMatchingFiles` must respect the split between directories
	t.Run("TestDiskStore_MatchingFiles_EnsureSubdirectorySplit", func(t *testing.T) {
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
		res, err := ps.GetMatching("", files)
		require.NoError(t, err)
		require.Empty(t, res)

		// ensure the new directory is not empty
		res, err = ps.GetMatching("uploaded", files)
		require.NoError(t, err)
		require.Equal(t, 3, len(res))

		// add more metrics
		addRecords()

		// ensure the root is not empty
		res, err = ps.GetFiles()
		require.NoError(t, err)
		require.Equal(t, 3, len(res))
	})

	// `GetMatchingFiles` must ONLY use the filename as the id, and ignore the rest of the path
	t.Run("TestDiskStore_MatchingFiles_EnsureIgnoreFullPath", func(t *testing.T) {
		files, err := ps.GetFiles()
		require.NoError(t, err)

		// create different paths
		newIds := make([]string, len(files))
		for _, item := range files {
			newIds = append(newIds, filepath.Join("invalid", "unrelated", "dir", item))
		}

		// ensure that even with bad paths only the filename is used
		res, err := ps.GetMatching("", newIds)
		require.NoError(t, err)
		require.Equal(t, len(files), len(res))
	})
}

func TestDiskStore_GetUsage(t *testing.T) {
	tmpDir := t.TempDir()
	d, err := store.NewDiskStore(config.Database{StoragePath: tmpDir, MaxRecords: 100})
	require.NoError(t, err)
	defer d.Flush()

	_, err = d.GetUsage()
	require.NoError(t, err)
}
