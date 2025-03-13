// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/andybalholm/brotli"
	"github.com/go-obvious/timestamp"
	"github.com/google/uuid"
	"github.com/launchdarkly/go-jsonstream/v3/jwriter"
	"github.com/rs/zerolog/log"

	"github.com/cloudzero/cloudzero-insights-controller/app/config"
	"github.com/cloudzero/cloudzero-insights-controller/app/types"
)

const (
	directoryMode     = 0o755
	batchSize         = 1000
	fileReadBatchSize = 1000
	jsonBufferSize    = 1024
)

const (
	CostContentIdentifier          = "metrics"
	ObservabilityContentIdentifier = "observability"
)

type DiskStoreOpt = func(d *DiskStore) error

func WithContentIdentifier(identifier string) DiskStoreOpt {
	return func(d *DiskStore) error {
		d.contentIdentifier = identifier
		return nil
	}
}

// DiskStore is a data store intended to be backed by a disk. Currently, data is stored in Brotli-compressed JSON, but transcoded to Snappy-compressed Parquet
type DiskStore struct {
	dirPath           string
	id                string
	contentIdentifier string
	activeFilePath    string
	rowLimit          int
	rowCount          int
	file              *os.File
	compressionLevel  int
	compressor        *brotli.Writer
	writer            *jwriter.Writer
	arrayState        *jwriter.ArrayState
	startTime         int64
	mu                sync.Mutex

	// internal metadata for the state of the disk store
	stat *syscall.Statfs_t
}

// Just to make sure DiskStore implements the AppendableFiles interface
var _ types.ReadableStore = (*DiskStore)(nil)

// Just to make sure DiskStore implements the DiskMonitor interface
var _ types.StoreMonitor = (*DiskStore)(nil)

// NewDiskStore initializes a DiskStore with a directory path and row limit
func NewDiskStore(settings config.Database, opts ...DiskStoreOpt) (*DiskStore, error) {
	if settings.MaxRecords <= 0 {
		settings.MaxRecords = config.DefaultDatabaseMaxRecords
	}
	if settings.CompressionLevel <= 0 || settings.CompressionLevel > brotli.BestCompression {
		settings.CompressionLevel = config.DefaultDatabaseCompressionLevel
	}
	if _, err := os.Stat(settings.StoragePath); os.IsNotExist(err) {
		if err := os.MkdirAll(settings.StoragePath, directoryMode); err != nil {
			return nil, fmt.Errorf("failed to create directory: %w", err)
		}
	}

	store := &DiskStore{
		dirPath:          settings.StoragePath,
		rowLimit:         settings.MaxRecords,
		id:               uuid.New().String()[:8],
		compressionLevel: settings.CompressionLevel,
	}

	// apply the opts
	for _, opt := range opts {
		if err := opt(store); err != nil {
			return nil, fmt.Errorf("failed to apply the store option: %w", err)
		}
	}

	if err := store.newFileWriter(); err != nil {
		return nil, err
	}
	return store, nil
}

func (d *DiskStore) makeFileName() string {
	return fmt.Sprintf("%s.%d", d.id, timestamp.Milli())
}

// newFileWriter creates a new Parquet writer with `active.json.br` as the active file
func (d *DiskStore) newFileWriter() error {
	// Intentionally make a new file, to prevent from collision on rename
	// for any OS level buffering
	d.activeFilePath = filepath.Join(d.dirPath, d.makeFileName())

	file, err := os.Create(d.activeFilePath)
	if err != nil {
		return fmt.Errorf("failed to create active file: %w", err)
	}

	compressor := brotli.NewWriterLevel(file, d.compressionLevel)

	writer := jwriter.NewStreamingWriter(compressor, jsonBufferSize)
	arrayState := writer.Array()

	d.rowCount = 0
	d.startTime = timestamp.Milli() // Capture the start time
	d.file = file
	d.compressor = compressor
	d.writer = &writer
	d.arrayState = &arrayState
	return nil
}

// Put appends metrics to the JSON file, creating a new file if the row limit is reached
func (d *DiskStore) Put(ctx context.Context, metrics ...types.Metric) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, metric := range metrics {
		encodedMetric, err := json.Marshal(metric)
		if err != nil {
			return fmt.Errorf("failed to marshal metric: %w", err)
		}
		d.arrayState.Raw(encodedMetric)
	}
	d.rowCount += len(metrics)

	// If row count exceeds the limit, flush and create a new active file
	if d.rowCount >= d.rowLimit {
		if err := d.flushUnlocked(); err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to flush writer")
			return err
		}
		if err := d.newFileWriter(); err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to create new file writer")
			return err
		}
	}
	return nil
}

// Flush finalizes the current writer, writes all buffered data to disk, and renames the file
func (d *DiskStore) Flush() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.flushUnlocked(); err != nil {
		log.Ctx(context.TODO()).Error().Err(err).Msg("failed to flush writer")
		return err
	}
	if err := d.newFileWriter(); err != nil {
		log.Ctx(context.TODO()).Error().Err(err).Msg("failed to create new file writer")
		return err
	}
	return nil
}

// flushUnlocked finalizes the current writer, writes all buffered data to disk, and renames the file
func (d *DiskStore) flushUnlocked() error {
	if d.writer == nil {
		return nil
	}

	// End the JSON array
	d.arrayState.End()

	// Flush the JSON writer to ensure all data is written to the compressor
	if err := d.writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush JSON writer: %w", err)
	}

	// Close the compressor to flush data
	if err := d.compressor.Close(); err != nil {
		return fmt.Errorf("failed to close compressor: %w", err)
	}

	// Close the file
	if err := d.file.Close(); err != nil {
		return fmt.Errorf("failed to close JSON file: %w", err)
	}

	// Capture stop time
	stopTime := timestamp.Milli()

	// create filename
	filename := d.contentIdentifier
	if filename == "" {
		filename = "file"
	}
	filename += fmt.Sprintf("_%d_%d.json.br", d.startTime, stopTime)

	// Rename the active file with start and stop timestamps
	timestampedFilePath := filepath.Join(
		d.dirPath,
		filename,
	)
	err := os.Rename(d.activeFilePath, timestampedFilePath)
	if err != nil {
		return fmt.Errorf("failed to rename active parquet file: %w", err)
	}

	// Reset writer and file pointers
	d.writer = nil
	d.arrayState = nil
	d.file = nil
	d.rowCount = 0 // Reset row count after flush
	return nil
}

// Pending returns the count of buffered rows not yet written to disk
func (d *DiskStore) Pending() int {
	return d.rowCount
}

func (d *DiskStore) GetFiles(paths ...string) ([]string, error) {
	// set to root path
	allPaths := []string{d.dirPath}

	// add specified location
	allPaths = append(allPaths, paths...)

	base := d.contentIdentifier
	if base == "" {
		base = "*"
	}

	// add file filter
	allPaths = append(allPaths, base+"_*_*.json.br")

	// list with glob find
	pattern := filepath.Join(allPaths...)
	return filepath.Glob(pattern)
}

func (d *DiskStore) ListFiles(paths ...string) ([]os.DirEntry, error) {
	allPaths := []string{d.dirPath}
	allPaths = append(allPaths, paths...)
	return os.ReadDir(filepath.Join(allPaths...))
}

// Walk will run `process` to walk the file tree
func (d *DiskStore) Walk(loc string, process filepath.WalkFunc) error {
	// walk the specific location in the store
	if err := filepath.Walk(filepath.Join(d.dirPath, loc), process); err != nil {
		return fmt.Errorf("failed to walk the store: %w", err)
	}

	return nil
}

// All retrieves all metrics from uncompacted .json.br files, excluding the active and compressed files.
// It reads the data into memory and returns a MetricRange.
func (d *DiskStore) All(ctx context.Context, file string) (types.MetricRange, error) {
	metrics, err := d.readCompressedJSONFile(file)
	if err != nil {
		return types.MetricRange{}, fmt.Errorf("failed to read parquet file %s: %w", file, err)
	}

	return types.MetricRange{
		Metrics: metrics,
		Next:    nil, // No pagination implemented
	}, nil
}

// readCompressedJSONFile reads all metrics from a single .json.br file and returns them as a slice.
func (d *DiskStore) readCompressedJSONFile(filePath string) ([]types.Metric, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return []types.Metric{}, nil // No file to read
	}

	// Open the JSON file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open JSON file: %w", err)
	}
	defer file.Close()

	// Create Brotli decompressor
	decompressor := brotli.NewReader(file)
	// defer decompressor.Close() // Necessary for cbrotli, but not the Go-native version

	// Create a JSON decoder
	decoder := json.NewDecoder(decompressor)

	// Read metrics from the JSON file
	var metrics []types.Metric
	err = decoder.Decode(&metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JSON file: %w", err)
	}

	return metrics, nil
}

// GetUsage gathers disk usage stats using syscall.Statfs.
// paths will be used as `filepath.Join(paths...)`
func (d *DiskStore) GetUsage(paths ...string) (*types.StoreUsage, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err != nil {
		return nil, err
	}

	// basic stats
	total := stat.Blocks * uint64(stat.Bsize)     //nolint:gosec // not an issue in 1.24
	available := stat.Bavail * uint64(stat.Bsize) //nolint:gosec // not an issue in 1.24
	used := total - available
	var percentUsed float64
	if total > 0 {
		percentUsed = (float64(used) / float64(total)) * 100
	}

	// This is USUALLY true
	reserved := (stat.Bfree - stat.Bavail) * uint64(stat.Bsize) //nolint:gosec // not an issue in 1.24

	// set inode information
	inodeTotal := stat.Files
	inodeAvailable := stat.Ffree
	inodeUsed := inodeTotal - inodeAvailable

	// save the stat information if needed
	d.stat = &stat

	return &types.StoreUsage{
		Total:          total,
		Available:      available,
		Used:           used,
		PercentUsed:    percentUsed,
		BlockSize:      uint32(stat.Bsize), //nolint:gosec // not an issue in 1.24
		Reserved:       reserved,
		InodeTotal:     inodeTotal,
		InodeUsed:      inodeUsed,
		InodeAvailable: inodeAvailable,
	}, nil
}
