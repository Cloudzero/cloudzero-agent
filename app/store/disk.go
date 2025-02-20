// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

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

// DiskStore is a data store intended to be backed by a disk. Currently, data is stored in Brotli-compressed JSON, but transcoded to Snappy-compressed Parquet
type DiskStore struct {
	dirPath          string
	id               string
	activeFilePath   string
	rowLimit         int
	rowCount         int
	file             *os.File
	compressionLevel int
	compressor       *brotli.Writer
	writer           *jwriter.Writer
	arrayState       *jwriter.ArrayState
	startTime        int64
	mu               sync.Mutex
}

// Just to make sure DiskStore implements the AppendableFiles interface
var _ types.AppendableFiles = (*DiskStore)(nil)

// NewDiskStore initializes a DiskStore with a directory path and row limit
func NewDiskStore(settings config.Database) (*DiskStore, error) {
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
		id:               uuid.New().String()[:8], //nolint:revive // we just want a random string
		compressionLevel: settings.CompressionLevel,
	}

	if err := store.newFileWriter(); err != nil {
		return nil, err
	}
	return store, nil
}

func (p *DiskStore) makeFileName() string {
	return fmt.Sprintf("%s.%d", p.id, timestamp.Milli())
}

// newFileWriter creates a new Parquet writer with `active.json.br` as the active file
func (p *DiskStore) newFileWriter() error {
	// Intentionally make a new file, to prevent from collision on rename
	// for any OS level buffering
	p.activeFilePath = filepath.Join(p.dirPath, p.makeFileName())

	file, err := os.Create(p.activeFilePath)
	if err != nil {
		return fmt.Errorf("failed to create active file: %w", err)
	}

	compressor := brotli.NewWriterLevel(file, p.compressionLevel)

	writer := jwriter.NewStreamingWriter(compressor, jsonBufferSize)
	arrayState := writer.Array()

	p.rowCount = 0
	p.startTime = timestamp.Milli() // Capture the start time
	p.file = file
	p.compressor = compressor
	p.writer = &writer
	p.arrayState = &arrayState
	return nil
}

// Put appends metrics to the JSON file, creating a new file if the row limit is reached
func (p *DiskStore) Put(ctx context.Context, metrics ...types.Metric) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, metric := range metrics {
		encodedMetric, err := json.Marshal(metric)
		if err != nil {
			return fmt.Errorf("failed to marshal metric: %w", err)
		}
		p.arrayState.Raw(encodedMetric)
	}
	p.rowCount += len(metrics)

	// If row count exceeds the limit, flush and create a new active file
	if p.rowCount >= p.rowLimit {
		if err := p.flushUnlocked(); err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to flush writer")
			return err
		}
		if err := p.newFileWriter(); err != nil {
			log.Ctx(ctx).Error().Err(err).Msg("failed to create new file writer")
			return err
		}
	}
	return nil
}

// Flush finalizes the current writer, writes all buffered data to disk, and renames the file
func (p *DiskStore) Flush() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.flushUnlocked(); err != nil {
		log.Ctx(context.TODO()).Error().Err(err).Msg("failed to flush writer")
		return err
	}
	if err := p.newFileWriter(); err != nil {
		log.Ctx(context.TODO()).Error().Err(err).Msg("failed to create new file writer")
		return err
	}
	return nil
}

// flushUnlocked finalizes the current writer, writes all buffered data to disk, and renames the file
func (p *DiskStore) flushUnlocked() error {
	if p.writer == nil {
		return nil
	}

	// End the JSON array
	p.arrayState.End()

	// Flush the JSON writer to ensure all data is written to the compressor
	if err := p.writer.Flush(); err != nil {
		return fmt.Errorf("failed to flush JSON writer: %w", err)
	}

	// Close the compressor to flush data
	if err := p.compressor.Close(); err != nil {
		return fmt.Errorf("failed to close compressor: %w", err)
	}

	// Close the file
	if err := p.file.Close(); err != nil {
		return fmt.Errorf("failed to close JSON file: %w", err)
	}

	// Capture stop time
	stopTime := timestamp.Milli()

	// Rename the active file with start and stop timestamps
	timestampedFilePath := filepath.Join(
		p.dirPath,
		fmt.Sprintf("metrics_%d_%d.json.br", p.startTime, stopTime),
	)
	err := os.Rename(p.activeFilePath, timestampedFilePath)
	if err != nil {
		return fmt.Errorf("failed to rename active parquet file: %w", err)
	}

	// Reset writer and file pointers
	p.writer = nil
	p.arrayState = nil
	p.file = nil
	p.rowCount = 0 // Reset row count after flush
	return nil
}

// Pending returns the count of buffered rows not yet written to disk
func (p *DiskStore) Pending() int {
	return p.rowCount
}

func (p *DiskStore) GetFiles() ([]string, error) {
	pattern := filepath.Join(p.dirPath, "metrics_*_*.json.br")
	return filepath.Glob(pattern)
}

// Gets a list of files that match a predefined list of target files from a specific
// subdirectory.
func (p *DiskStore) GetMatching(loc string, targets []string) ([]string, error) {
	// create a lookup table of the targets to search for
	targetMap := make(map[string]any, len(targets))
	for _, item := range targets {
		targetMap[filepath.Base(item)] = struct{}{}
	}

	// store list of all found paths that match the requested targets
	var matches []string

	// open a pointer to the directory requested
	handle, err := os.Open(filepath.Join(p.dirPath, loc))
	if err != nil {
		return nil, fmt.Errorf("failed to open the directory: %w", err)
	}
	defer handle.Close()

	// TODO -- could pontentially run in a go-routine if enough files
	// but may add overhead. Need more testing to see if this would be valuable
	for {
		// read in chunks
		files, err := handle.ReadDir(fileReadBatchSize)

		// if the directory is empty, skip
		if err == io.EOF {
			break
		}

		// check for actual error
		if err != nil {
			return nil, fmt.Errorf("failed to read the directory: %w", err)
		}

		// check for matches
		for _, file := range files {
			if _, exists := targetMap[file.Name()]; exists {
				matches = append(matches, file.Name())
			}
		}

		if len(files) == 0 {
			break
		}
	}

	return matches, nil
}

// All retrieves all metrics from uncompacted .json.br files, excluding the active and compressed files.
// It reads the data into memory and returns a MetricRange.
func (p *DiskStore) All(ctx context.Context, file string) (types.MetricRange, error) {
	metrics, err := p.readCompressedJSONFile(file)
	if err != nil {
		return types.MetricRange{}, fmt.Errorf("failed to read parquet file %s: %w", file, err)
	}

	return types.MetricRange{
		Metrics: metrics,
		Next:    nil, // No pagination implemented
	}, nil
}

// readCompressedJSONFile reads all metrics from a single .json.br file and returns them as a slice.
func (p *DiskStore) readCompressedJSONFile(filePath string) ([]types.Metric, error) {
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
