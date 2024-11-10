package store_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudzero/cirrus-remote-write/app/store"
	"github.com/cloudzero/cirrus-remote-write/app/types"
	"github.com/stretchr/testify/assert"
)

func TestParquetStore_PutAndPending(t *testing.T) {
	dirPath := t.TempDir()
	rowLimit := 10

	ps, err := store.NewParquetStore(dirPath, rowLimit)
	assert.NoError(t, err)
	defer ps.Flush()

	// Add metrics less than the row limit
	metric := types.NewMetric("test_metric", time.Now().Unix(), map[string]string{"label": "test"}, "123.45")
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

func TestParquetStore_Flush(t *testing.T) {
	dirPath := t.TempDir()
	rowLimit := 5

	ps, err := store.NewParquetStore(dirPath, rowLimit)
	assert.NoError(t, err)

	// Add metrics and verify they are pending
	metric := types.NewMetric("test_metric", time.Now().Unix(), map[string]string{"label": "test"}, "123.45")
	err = ps.Put(context.Background(), metric, metric)
	assert.NoError(t, err)
	assert.Equal(t, 2, ps.Pending())

	// Call Flush to write all pending data to disk
	err = ps.Flush()
	assert.NoError(t, err)

	// Verify that all pending data has been written
	assert.Equal(t, 0, ps.Pending())

	// Check that at least one file has been created
	files, err := filepath.Glob(filepath.Join(dirPath, "metrics_*.parquet"))
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(files), 1)
}

func TestParquetStore_AutoRotateFileOnRowLimit(t *testing.T) {
	dirPath := t.TempDir()
	rowLimit := 3

	ps, err := store.NewParquetStore(dirPath, rowLimit)
	assert.NoError(t, err)
	defer ps.Flush()

	// Add metrics exactly to the row limit to trigger rotation
	metric := types.NewMetric("test_metric", time.Now().Unix(), map[string]string{"label": "test"}, "123.45")
	err = ps.Put(context.Background(), metric, metric, metric)
	assert.NoError(t, err)

	// Verify Pending is reset after rotation and row limit exceeded
	assert.Equal(t, 0, ps.Pending())

	// Add another metric after rotation to confirm it's buffered in a new file
	err = ps.Put(context.Background(), metric)
	assert.NoError(t, err)
	assert.Equal(t, 1, ps.Pending())

	// Check that two files have been created due to auto-rotation
	files, err := filepath.Glob(filepath.Join(dirPath, "metrics_*.parquet"))
	assert.NoError(t, err)
	assert.Equal(t, 2, len(files))
}
