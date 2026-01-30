// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package metricio

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/prometheus/prompb"
)

// CAdvisorMetric represents a row from the cAdvisor CSV export.
type CAdvisorMetric struct {
	Timestamp time.Time
	Value     float64
	Labels    map[string]string
}

// CSVReader reads cAdvisor metrics from CSV files exported from Snowflake.
type CSVReader struct {
	batchSize int
}

// NewCSVReader creates a new CSVReader with the specified batch size.
func NewCSVReader(batchSize int) *CSVReader {
	if batchSize <= 0 {
		batchSize = DefaultBatchSize
	}
	return &CSVReader{batchSize: batchSize}
}

// ReadCSVFile reads cAdvisor metrics from a CSV file with columns: USAGE_DATE, VALUE, LABELS.
// It calls the callback function with batches of TimeSeries.
func (r *CSVReader) ReadCSVFile(path string, callback func([]prompb.TimeSeries) error) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open CSV file %s: %w", path, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1 // Allow variable number of fields due to JSON containing commas

	// Read header
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read CSV header: %w", err)
	}

	// Find column indices (case-insensitive matching)
	dateIdx, valueIdx, labelsIdx := -1, -1, -1
	for i, col := range header {
		switch strings.ToUpper(col) {
		case "USAGE_DATE":
			dateIdx = i
		case "VALUE":
			valueIdx = i
		case "LABELS":
			labelsIdx = i
		}
	}

	if dateIdx == -1 || valueIdx == -1 || labelsIdx == -1 {
		return fmt.Errorf("CSV must have USAGE_DATE, VALUE, and LABELS columns, got: %v", header)
	}

	batch := make([]prompb.TimeSeries, 0, r.batchSize)
	rowNum := 0

	for {
		record, err := reader.Read()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			// Try to continue on parse errors
			rowNum++
			continue
		}
		rowNum++

		if len(record) <= labelsIdx {
			continue
		}

		// Parse timestamp: "2026-01-14 18:14:42.948 Z"
		ts, err := time.Parse("2006-01-02 15:04:05.999 Z", record[dateIdx])
		if err != nil {
			// Try alternative format without Z
			ts, err = time.Parse("2006-01-02 15:04:05.999", record[dateIdx])
			if err != nil {
				continue
			}
		}

		// Parse value
		value, err := strconv.ParseFloat(record[valueIdx], 64)
		if err != nil {
			continue
		}

		// Parse labels JSON
		var labels map[string]string
		if err := json.Unmarshal([]byte(record[labelsIdx]), &labels); err != nil {
			continue
		}

		// Convert to TimeSeries
		tsSeries := convertCAdvisorToTimeSeries(ts, value, labels)
		batch = append(batch, tsSeries)

		if len(batch) >= r.batchSize {
			if err := callback(batch); err != nil {
				return fmt.Errorf("callback error at row %d: %w", rowNum, err)
			}
			batch = make([]prompb.TimeSeries, 0, r.batchSize)
		}
	}

	// Send remaining batch
	if len(batch) > 0 {
		if err := callback(batch); err != nil {
			return fmt.Errorf("callback error at final batch: %w", err)
		}
	}

	return nil
}

// convertCAdvisorToTimeSeries converts a cAdvisor metric to a Prometheus TimeSeries.
func convertCAdvisorToTimeSeries(timestamp time.Time, value float64, labels map[string]string) prompb.TimeSeries {
	promLabels := make([]prompb.Label, 0, len(labels))

	for k, v := range labels {
		if v == "" {
			continue
		}
		promLabels = append(promLabels, prompb.Label{Name: k, Value: v})
	}

	// Sort labels for consistency (required by Prometheus)
	sortLabels(promLabels)

	return prompb.TimeSeries{
		Labels: promLabels,
		Samples: []prompb.Sample{{
			Value:     value,
			Timestamp: timestamp.UnixMilli(),
		}},
	}
}

// sortLabels sorts labels by name for consistent ordering.
func sortLabels(labels []prompb.Label) {
	sort.Slice(labels, func(i, j int) bool {
		return labels[i].Name < labels[j].Name
	})
}
