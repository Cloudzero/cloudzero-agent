package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/cloudzero/cirrus-remote-write/app/types"
	"github.com/go-obvious/timestamp"
	"github.com/parquet-go/parquet-go"
)

type ParquetStore struct {
	dirPath  string
	rowLimit int
	rowCount int
	file     *os.File
	writer   *parquet.GenericWriter[types.Metric]
	mu       sync.Mutex
}

// NewParquetStore initializes a ParquetStore with a directory path and row limit
func NewParquetStore(dirPath string, rowLimit int) (*ParquetStore, error) {
	store := &ParquetStore{
		dirPath:  dirPath,
		rowLimit: rowLimit,
	}
	if err := store.newFileWriter(); err != nil {
		return nil, err
	}
	return store, nil
}

// newFileWriter creates a new Parquet writer with `active.parquet` as the active file
func (p *ParquetStore) newFileWriter() error {
	activeFilePath := filepath.Join(p.dirPath, "active.parquet")

	file, err := os.Create(activeFilePath)
	if err != nil {
		return fmt.Errorf("failed to create active parquet file: %w", err)
	}
	writer := parquet.NewGenericWriter[types.Metric](file)

	p.file = file
	p.writer = writer
	p.rowCount = 0
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
	if err := p.writer.Close(); err != nil {
		return fmt.Errorf("failed to flush parquet writer: %w", err)
	}
	if err := p.file.Close(); err != nil {
		return fmt.Errorf("failed to close parquet file: %w", err)
	}

	// Rename the active file with a timestamp to mark it as complete
	timestampedFilePath := filepath.Join(p.dirPath, fmt.Sprintf("metrics_%d.parquet", timestamp.Milli()))
	err := os.Rename(filepath.Join(p.dirPath, "active.parquet"), timestampedFilePath)
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
