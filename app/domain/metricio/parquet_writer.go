// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package metricio

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/parquet-go/parquet-go"
)

// ParquetWriter writes metrics to a Snappy-compressed Parquet file.
type ParquetWriter struct {
	dest   io.Writer
	writer *parquet.GenericWriter[types.ParquetMetric]
	mu     sync.Mutex
	closed bool
}

// NewParquetWriterToWriter creates a new ParquetWriter that writes to the given io.Writer.
func NewParquetWriterToWriter(dest io.Writer) (*ParquetWriter, error) {
	writer := parquet.NewGenericWriter[types.ParquetMetric](dest, parquet.Compression(&parquet.Snappy))

	return &ParquetWriter{
		dest:   dest,
		writer: writer,
	}, nil
}

// NewParquetWriter creates a new ParquetWriter that writes to the specified path.
func NewParquetWriter(path string) (*ParquetWriter, error) {
	file, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("failed to create output file %s: %w", path, err)
	}

	w, err := NewParquetWriterToWriter(file)
	if err != nil {
		file.Close()
		return nil, err
	}

	return w, nil
}

// Write writes a batch of metrics to the Parquet file.
// Metrics are converted to ParquetMetric format before writing.
func (w *ParquetWriter) Write(metrics []types.Metric) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return errors.New("writer is closed")
	}

	parquetMetrics := make([]types.ParquetMetric, len(metrics))
	for i, m := range metrics {
		parquetMetrics[i] = m.Parquet()
	}

	_, err := w.writer.Write(parquetMetrics)
	if err != nil {
		return fmt.Errorf("failed to write metrics to Parquet: %w", err)
	}

	return nil
}

// WriteOne writes a single metric to the Parquet file.
func (w *ParquetWriter) WriteOne(metric types.Metric) error {
	return w.Write([]types.Metric{metric})
}

// WriteParquetMetrics writes pre-converted ParquetMetric records directly.
func (w *ParquetWriter) WriteParquetMetrics(metrics []types.ParquetMetric) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return errors.New("writer is closed")
	}

	_, err := w.writer.Write(metrics)
	if err != nil {
		return fmt.Errorf("failed to write metrics to Parquet: %w", err)
	}

	return nil
}

// Close closes the writer, flushing any remaining data.
// If the underlying writer implements io.Closer, it will be closed.
func (w *ParquetWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return nil
	}
	w.closed = true

	// Close the Parquet writer (flushes buffered data)
	if err := w.writer.Close(); err != nil {
		return fmt.Errorf("failed to close Parquet writer: %w", err)
	}

	// Sync if the destination supports it (e.g., *os.File)
	if syncer, ok := w.dest.(interface{ Sync() error }); ok {
		if err := syncer.Sync(); err != nil {
			return fmt.Errorf("failed to sync: %w", err)
		}
	}

	// Close if the destination supports it
	if closer, ok := w.dest.(io.Closer); ok {
		if err := closer.Close(); err != nil {
			return fmt.Errorf("failed to close: %w", err)
		}
	}

	return nil
}
