// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package metricio provides readers and writers for metric data in various formats
// including Parquet, CSV, JSON, and Prometheus remote_write.
package metricio

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/parquet-go/parquet-go"
)

const (
	// DefaultBatchSize is the default number of metrics to read at a time.
	DefaultBatchSize = 10000
)

// ParquetReader reads ParquetMetric records from Parquet files.
type ParquetReader struct {
	batchSize int
}

// NewParquetReader creates a new ParquetReader with the specified batch size.
func NewParquetReader(batchSize int) *ParquetReader {
	if batchSize <= 0 {
		batchSize = DefaultBatchSize
	}
	return &ParquetReader{batchSize: batchSize}
}

// IsSupportedFile returns true if the file has a supported extension for metric data.
// Supported formats: .csv, .parquet, .json, .json.br
func IsSupportedFile(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".csv") ||
		strings.HasSuffix(lower, ".parquet") ||
		strings.HasSuffix(lower, ".json") ||
		strings.HasSuffix(lower, ".json.br")
}

// FindFiles recursively finds all files in the given directory that match the filter.
// It follows symlinks to directories. If filter is nil, all files are included.
func FindFiles(dir string, filter func(path string) bool) ([]string, error) {
	var files []string

	var walkDir func(string) error
	walkDir = func(currentDir string) error {
		entries, err := os.ReadDir(currentDir)
		if err != nil {
			return fmt.Errorf("failed to read directory %s: %w", currentDir, err)
		}

		for _, entry := range entries {
			path := filepath.Join(currentDir, entry.Name())

			// Resolve symlinks
			info, err := os.Stat(path) // Stat follows symlinks, Lstat does not
			if err != nil {
				return fmt.Errorf("failed to stat %s: %w", path, err)
			}

			if info.IsDir() {
				if err := walkDir(path); err != nil {
					return err
				}
			} else if filter == nil || filter(path) {
				files = append(files, path)
			}
		}
		return nil
	}

	if err := walkDir(dir); err != nil {
		return nil, err
	}
	return files, nil
}

// FindSupportedFiles recursively finds all supported metric files in the given directory.
// Supported formats: .csv, .parquet, .json, .json.br
func FindSupportedFiles(dir string) ([]string, error) {
	return FindFiles(dir, IsSupportedFile)
}

// ReadFile reads all ParquetMetric records from a single Parquet file.
// It calls the callback function with batches of metrics.
func (r *ParquetReader) ReadFile(path string, callback func([]types.ParquetMetric) error) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", path, err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file %s: %w", path, err)
	}

	reader := parquet.NewGenericReader[types.ParquetMetric](file)
	defer reader.Close()

	batch := make([]types.ParquetMetric, r.batchSize)
	totalRows := int64(0)

	for {
		n, readErr := reader.Read(batch)
		if n > 0 {
			totalRows += int64(n)
			if callbackErr := callback(batch[:n]); callbackErr != nil {
				return fmt.Errorf("callback error at row %d: %w", totalRows, callbackErr)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return fmt.Errorf("failed to read from file %s at row %d (file size: %d): %w", path, totalRows, stat.Size(), readErr)
		}
	}

	return nil
}

// ReadFiles reads all ParquetMetric records from multiple Parquet files.
// It calls the callback function with batches of metrics from each file.
func (r *ParquetReader) ReadFiles(paths []string, callback func(path string, metrics []types.ParquetMetric) error) error {
	for _, path := range paths {
		err := r.ReadFile(path, func(metrics []types.ParquetMetric) error {
			return callback(path, metrics)
		})
		if err != nil {
			return err
		}
	}
	return nil
}
