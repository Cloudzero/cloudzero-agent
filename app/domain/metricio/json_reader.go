// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package metricio

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/andybalholm/brotli"
	"github.com/cloudzero/cloudzero-agent/app/types"
)

// JSONReader reads metrics from JSON or Brotli-compressed JSON files.
type JSONReader struct {
	batchSize int
}

// NewJSONReader creates a new JSONReader with the specified batch size.
func NewJSONReader(batchSize int) *JSONReader {
	if batchSize <= 0 {
		batchSize = DefaultBatchSize
	}
	return &JSONReader{batchSize: batchSize}
}

// ReadJSONFile reads metrics from a JSON or JSON.br file.
// It detects compression from the file extension and calls the callback with batches of metrics.
func (r *JSONReader) ReadJSONFile(path string, callback func([]types.Metric) error) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open JSON file %s: %w", path, err)
	}
	defer file.Close()

	// Determine if file is Brotli-compressed based on extension
	var reader io.Reader = file
	if strings.HasSuffix(path, ".br") {
		reader = brotli.NewReader(file)
	}

	return r.decodeJSONStream(reader, callback)
}

// ReadFromReader reads metrics from an io.Reader containing JSON data.
// This is useful for reading from streams or testing.
func (r *JSONReader) ReadFromReader(reader io.Reader, callback func([]types.Metric) error) error {
	return r.decodeJSONStream(reader, callback)
}

func (r *JSONReader) decodeJSONStream(reader io.Reader, callback func([]types.Metric) error) error {
	decoder := json.NewDecoder(reader)

	// Read opening bracket
	token, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("failed to read first token from JSON: %w", err)
	}
	if token != json.Delim('[') {
		return fmt.Errorf("expected '[' at the beginning of the file, got %v", token)
	}

	batch := make([]types.Metric, 0, r.batchSize)

	// Read metrics
	for decoder.More() {
		var metric types.Metric
		if decodeErr := decoder.Decode(&metric); decodeErr != nil {
			return fmt.Errorf("failed to decode metric: %w", decodeErr)
		}
		batch = append(batch, metric)

		if len(batch) >= r.batchSize {
			if callbackErr := callback(batch); callbackErr != nil {
				return fmt.Errorf("callback error: %w", callbackErr)
			}
			batch = make([]types.Metric, 0, r.batchSize)
		}
	}

	// Read closing bracket
	token, err = decoder.Token()
	if err != nil {
		return fmt.Errorf("failed to read last token from JSON: %w", err)
	}
	if token != json.Delim(']') {
		return fmt.Errorf("expected ']' at the end of the file, got %v", token)
	}

	// Send remaining batch
	if len(batch) > 0 {
		if callbackErr := callback(batch); callbackErr != nil {
			return fmt.Errorf("callback error at final batch: %w", callbackErr)
		}
	}

	return nil
}

// ReadAllMetrics reads all metrics from a JSON or JSON.br file into memory.
// This is a convenience method for smaller files.
func (r *JSONReader) ReadAllMetrics(path string) ([]types.Metric, error) {
	var allMetrics []types.Metric
	err := r.ReadJSONFile(path, func(metrics []types.Metric) error {
		allMetrics = append(allMetrics, metrics...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return allMetrics, nil
}
