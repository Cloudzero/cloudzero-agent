package store_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/cloudzero/cirrus-remote-write/app/config"
	"github.com/cloudzero/cirrus-remote-write/app/store"
	"github.com/cloudzero/cirrus-remote-write/app/types"
	"github.com/stretchr/testify/assert"
)

func TestParquetStore_PutAndPending(t *testing.T) {
	dirPath := t.TempDir()
	rowLimit := 10

	ps, err := store.NewParquetStore(config.Database{StoragePath: dirPath, MaxRecords: rowLimit})
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

	ps, err := store.NewParquetStore(config.Database{StoragePath: dirPath, MaxRecords: rowLimit})
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
}

func TestParquetStore_Compact(t *testing.T) {
	// create a unique directory for each test
	dirPath, err := os.MkdirTemp(t.TempDir(), "TestParquetStore_Compact_")
	ctx := context.Background()
	rowLimit := 100
	fileCount := 3
	recordCount := rowLimit * fileCount

	ps, err := store.NewParquetStore(config.Database{StoragePath: dirPath, MaxRecords: rowLimit})
	assert.NoError(t, err)
	defer ps.Flush()

	for i := 0; i < recordCount; i++ {
		id := fmt.Sprintf("test_metric_%d", i)
		value := fmt.Sprintf("%d", i)
		metric := types.NewMetric(
			id,
			time.Now().Unix(),
			map[string]string{"label": id},
			value,
		)
		err := ps.Put(ctx, metric)
		assert.NoError(t, err)
	}
	// give a moment to allow OS async operatoins to complete
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
