// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package metricio

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/cloudzero/cloudzero-agent/app/types"
)

// CSVWriter writes metrics to CSV in the Snowflake export format.
// The format uses columns: USAGE_DATE, VALUE, LABELS (with labels as JSON).
type CSVWriter struct {
	dest   io.Writer
	writer *csv.Writer
	mu     sync.Mutex
	closed bool
}

// NewCSVWriterToWriter creates a new CSVWriter that writes to the given io.Writer.
func NewCSVWriterToWriter(dest io.Writer) (*CSVWriter, error) {
	writer := csv.NewWriter(dest)

	// Write header
	if err := writer.Write([]string{"USAGE_DATE", "VALUE", "LABELS"}); err != nil {
		return nil, fmt.Errorf("failed to write CSV header: %w", err)
	}

	return &CSVWriter{
		dest:   dest,
		writer: writer,
	}, nil
}

// NewCSVWriter creates a new CSVWriter that writes to the specified path.
func NewCSVWriter(path string) (*CSVWriter, error) {
	file, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("failed to create CSV file %s: %w", path, err)
	}

	w, err := NewCSVWriterToWriter(file)
	if err != nil {
		file.Close()
		return nil, err
	}

	return w, nil
}

// Write writes a batch of metrics to the CSV output.
func (w *CSVWriter) Write(metrics []types.Metric) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return errors.New("writer is closed")
	}

	for _, metric := range metrics {
		if err := w.writeMetric(metric); err != nil {
			return err
		}
	}

	return nil
}

// WriteOne writes a single metric to the CSV output.
func (w *CSVWriter) WriteOne(metric types.Metric) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return errors.New("writer is closed")
	}

	return w.writeMetric(metric)
}

func (w *CSVWriter) writeMetric(metric types.Metric) error {
	// Format timestamp as "2006-01-02 15:04:05.999 Z"
	timestamp := metric.TimeStamp.UTC().Format("2006-01-02 15:04:05.000") + " Z"

	// Serialize labels to JSON
	labels := metric.FullLabels()
	labelsJSON, err := json.Marshal(labels)
	if err != nil {
		return fmt.Errorf("failed to marshal labels: %w", err)
	}

	record := []string{
		timestamp,
		metric.Value,
		string(labelsJSON),
	}

	if err := w.writer.Write(record); err != nil {
		return fmt.Errorf("failed to write CSV record: %w", err)
	}

	return nil
}

// Close closes the writer, flushing any remaining data.
// If the underlying writer implements io.Closer, it will be closed.
func (w *CSVWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return nil
	}
	w.closed = true

	// Flush any buffered data
	w.writer.Flush()
	if err := w.writer.Error(); err != nil {
		return fmt.Errorf("failed to flush CSV writer: %w", err)
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
