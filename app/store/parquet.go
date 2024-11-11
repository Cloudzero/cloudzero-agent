package store

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/cloudzero/cirrus-remote-write/app/config"
	"github.com/cloudzero/cirrus-remote-write/app/types"
	"github.com/go-obvious/timestamp"
	"github.com/google/uuid"
	"github.com/parquet-go/parquet-go"
)

const (
	// DEFAULT_ROW_LIMIT results in approximately 25MB files
	// BEFORE  001 ls -lah
	// -rw-r--r--  1 joe.barnett  staff    94M Nov 10 13:02 metrics_1731254556187_1731254557963.parquet
	// AFTER 001 ls -lah
	// total 52792
	// -rw-r--r--  1 joe.barnett  staff    25M Nov 10 13:03 metrics_1731254556187_1731254557963.parquet.tgz
	DEFAULT_ROW_LIMIT = 1_000_000
)

type ParquetStore struct {
	dirPath        string
	activeFilename string
	rowLimit       int
	rowCount       int
	file           *os.File
	writer         *parquet.GenericWriter[types.Metric]
	startTime      int64
	mu             sync.Mutex
}

// NewParquetStore initializes a ParquetStore with a directory path and row limit
func NewParquetStore(settings config.Database) (*ParquetStore, error) {
	if settings.MaxRecords <= 0 {
		settings.MaxRecords = DEFAULT_ROW_LIMIT
	}
	if _, err := os.Stat(settings.StoragePath); os.IsNotExist(err) {
		if err := os.MkdirAll(settings.StoragePath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory: %w", err)
		}
	}

	store := &ParquetStore{
		dirPath:        settings.StoragePath,
		rowLimit:       settings.MaxRecords,
		activeFilename: fmt.Sprintf("active_%s.parquet", uuid.New().String()[:8]),
	}

	if err := store.newFileWriter(); err != nil {
		return nil, err
	}
	return store, nil
}

// newFileWriter creates a new Parquet writer with `active.parquet` as the active file
func (p *ParquetStore) newFileWriter() error {
	activeFilePath := filepath.Join(p.dirPath, p.activeFilename)

	file, err := os.Create(activeFilePath)
	if err != nil {
		return fmt.Errorf("failed to create active parquet file: %w", err)
	}
	writer := parquet.NewGenericWriter[types.Metric](file)

	p.file = file
	p.writer = writer
	p.rowCount = 0
	p.startTime = timestamp.Milli() // Capture the start time
	return nil
}

// Put appends metrics to the Parquet file, creating a new file if the row limit is reached
func (p *ParquetStore) Put(ctx context.Context, metrics ...types.Metric) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	_, err := p.writer.Write(metrics)
	if err != nil {
		return fmt.Errorf("failed to write metrics: %w", err)
	}
	p.rowCount += len(metrics)

	// If row count exceeds the limit, flush and create a new active file
	if p.rowCount >= p.rowLimit {
		if err := p.Flush(); err != nil {
			return err
		}
		if err := p.newFileWriter(); err != nil {
			return err
		}
	}
	return nil
}

// Flush finalizes the current writer, writes all buffered data to disk, and renames the file
func (p *ParquetStore) Flush() error {
	// Ensure Flush is protected by the mutex lock in Put
	if p.writer == nil {
		return nil
	}

	// Close the writer to flush data
	if err := p.writer.Close(); err != nil {
		return fmt.Errorf("failed to close parquet writer: %w", err)
	}

	// Close the file
	if err := p.file.Close(); err != nil {
		return fmt.Errorf("failed to close parquet file: %w", err)
	}

	// Capture stop time
	stopTime := timestamp.Milli()

	// Rename the active file with start and stop timestamps
	timestampedFilePath := filepath.Join(
		p.dirPath,
		fmt.Sprintf("metrics_%d_%d.parquet", p.startTime, stopTime),
	)
	err := os.Rename(filepath.Join(p.dirPath, p.activeFilename), timestampedFilePath)
	if err != nil {
		return fmt.Errorf("failed to rename active parquet file: %w", err)
	}

	// Reset writer and file pointers
	p.writer = nil
	p.file = nil
	p.rowCount = 0 // Reset row count after flush
	return nil
}

// Pending returns the count of buffered rows not yet written to disk
func (p *ParquetStore) Pending() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.rowCount
}

func (p *ParquetStore) GetFiles() ([]string, error) {
	pattern := filepath.Join(p.dirPath, "metrics_*_*.parquet")
	return filepath.Glob(pattern)
}

// All retrieves all metrics from uncompacted .parquet files, excluding the active and compressed files.
// It reads the data into memory and returns a MetricRange.
func (p *ParquetStore) All(ctx context.Context, file string) (types.MetricRange, error) {
	p.mu.Lock() // Don't allow a flush to occur while reading
	defer p.mu.Unlock()

	metrics, err := p.readParquetFile(file)
	if err != nil {
		return types.MetricRange{}, fmt.Errorf("failed to read parquet file %s: %w", file, err)
	}

	return types.MetricRange{
		Metrics: metrics,
		Next:    nil, // No pagination implemented
	}, nil
}

// readParquetFile reads all metrics from a single .parquet file and returns them as a slice.
func (p *ParquetStore) readParquetFile(parquetFilePath string) ([]types.Metric, error) {
	if _, err := os.Stat(parquetFilePath); os.IsNotExist(err) {
		return []types.Metric{}, nil // No file to read
	}

	// Open the parquet file
	file, err := os.Open(parquetFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open parquet file: %w", err)
	}
	defer file.Close()

	// Create a parquet reader
	reader := parquet.NewGenericReader[types.Metric](file)
	defer reader.Close()

	var metrics []types.Metric
	batchSize := 1000
	buffer := make([]types.Metric, batchSize)

	for {
		n, err := reader.Read(buffer)
		if err != nil {
			if err == io.EOF {
				metrics = append(metrics, buffer[:n]...)
				break
			}
			return nil, fmt.Errorf("error reading from parquet file: %w", err)
		}

		// Append the read metrics to the slice
		metrics = append(metrics, buffer[:n]...)
	}

	return metrics, nil
}
