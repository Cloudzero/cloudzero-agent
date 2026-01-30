// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package metricio

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/andybalholm/brotli"
	"github.com/cloudzero/cloudzero-agent/app/types"
)

const (
	// DefaultCompressionLevel is the default Brotli compression level.
	DefaultCompressionLevel = 8

	// NoCompression disables compression when passed to NewJSONWriterToWriter.
	NoCompression = -1
)

// JSONWriter writes metrics to JSON or Brotli-compressed JSON.
type JSONWriter struct {
	dest       io.Writer
	compressor *brotli.Writer
	encoder    *json.Encoder
	writer     io.Writer
	first      bool
	mu         sync.Mutex
	closed     bool
}

// NewJSONWriterToWriter creates a new JSONWriter that writes to the given io.Writer.
// If compressionLevel >= 0, Brotli compression is applied at that level.
// If compressionLevel < 0 (e.g., NoCompression), no compression is used.
func NewJSONWriterToWriter(dest io.Writer, compressionLevel int) (*JSONWriter, error) {
	w := &JSONWriter{
		dest:  dest,
		first: true,
	}

	if compressionLevel >= 0 {
		w.compressor = brotli.NewWriterLevel(dest, compressionLevel)
		w.writer = w.compressor
	} else {
		w.writer = dest
	}

	// Write opening bracket
	if _, err := w.writer.Write([]byte("[\n")); err != nil {
		return nil, fmt.Errorf("failed to write opening bracket: %w", err)
	}

	w.encoder = json.NewEncoder(w.writer)

	return w, nil
}

// NewJSONWriter creates a new JSONWriter that writes to the specified path.
// If the path ends with .br, Brotli compression is used at the default level.
func NewJSONWriter(path string) (*JSONWriter, error) {
	compressed := strings.HasSuffix(path, ".br")
	if compressed {
		return NewJSONWriterWithCompression(path, DefaultCompressionLevel)
	}
	return NewJSONWriterWithCompression(path, NoCompression)
}

// NewJSONWriterWithCompression creates a new JSONWriter that writes to the specified path
// with a specific compression level. Use NoCompression (-1) to disable compression.
func NewJSONWriterWithCompression(path string, compressionLevel int) (*JSONWriter, error) {
	file, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("failed to create output file %s: %w", path, err)
	}

	w, err := NewJSONWriterToWriter(file, compressionLevel)
	if err != nil {
		file.Close()
		return nil, err
	}

	return w, nil
}

// Write writes a batch of metrics to the JSON output.
func (w *JSONWriter) Write(metrics []types.Metric) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return errors.New("writer is closed")
	}

	for _, metric := range metrics {
		if !w.first {
			if _, err := w.writer.Write([]byte(",\n")); err != nil {
				return fmt.Errorf("failed to write separator: %w", err)
			}
		}
		w.first = false

		if err := w.encoder.Encode(metric); err != nil {
			return fmt.Errorf("failed to encode metric: %w", err)
		}
	}

	return nil
}

// WriteOne writes a single metric to the JSON output.
func (w *JSONWriter) WriteOne(metric types.Metric) error {
	return w.Write([]types.Metric{metric})
}

// Close closes the writer, flushing any remaining data.
// If the underlying writer implements io.Closer, it will be closed.
func (w *JSONWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return nil
	}
	w.closed = true

	// Write closing bracket
	if _, err := w.writer.Write([]byte("\n]")); err != nil {
		return fmt.Errorf("failed to write closing bracket: %w", err)
	}

	// Close compressor if used
	if w.compressor != nil {
		if err := w.compressor.Close(); err != nil {
			return fmt.Errorf("failed to close compressor: %w", err)
		}
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
