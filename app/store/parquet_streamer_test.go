// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package store_test

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/cloudzero/cloudzero-agent/app/store"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/google/go-cmp/cmp"
	"github.com/parquet-go/parquet-go"
	"github.com/stretchr/testify/assert"
)

var testMetrics = []types.Metric{
	{
		ClusterName:    "test-cluster",
		CloudAccountID: "1234567890",
		MetricName:     "test-metric-1",
		NodeName:       "my-node",
		CreatedAt:      time.UnixMilli(1741116110190).UTC(),
		Value:          "I'm a value!",
		TimeStamp:      time.UnixMilli(1741116110190).UTC(),
		Labels: map[string]string{
			"foo": "bar",
		},
	},
	{
		ClusterName:    "test-cluster",
		CloudAccountID: "1234567890",
		MetricName:     "test-metric-2",
		NodeName:       "my-node",
		CreatedAt:      time.UnixMilli(1741116110190).UTC(),
		Value:          "I'm a value!",
		TimeStamp:      time.UnixMilli(1741116110190).UTC(),
		Labels: map[string]string{
			"foo": "bar",
		},
	},
	{
		ClusterName:    "test-cluster",
		CloudAccountID: "1234567890",
		MetricName:     "test-metric-3",
		NodeName:       "my-node",
		CreatedAt:      time.UnixMilli(1741116110190).UTC(),
		Value:          "I'm a value!",
		TimeStamp:      time.UnixMilli(1741116110190).UTC(),
		Labels: map[string]string{
			"foo": "bar",
		},
	},
}

func TestNewParquetStreamer_RoundTrip(t *testing.T) {
	pr, pw := io.Pipe()

	go func() {
		compressor := brotli.NewWriterLevel(pw, 1)
		defer func() {
			compressor.Close()
			pw.Close()
		}()

		encoder := json.NewEncoder(compressor)
		err := encoder.Encode(testMetrics)
		assert.NoError(t, err)
	}()

	parquetStreamer := store.NewParquetStreamer(pr)
	defer parquetStreamer.Close()

	parquetData, err := io.ReadAll(parquetStreamer)
	assert.NoError(t, err)

	parquetReader := parquet.NewGenericReader[types.ParquetMetric](bytes.NewReader(parquetData))
	defer parquetReader.Close()
	assert.Equal(t, len(testMetrics), int(parquetReader.NumRows()))

	decodedParquetMetrics := make([]types.ParquetMetric, len(testMetrics))
	rowsRead, err := parquetReader.Read(decodedParquetMetrics)
	if err != nil {
		assert.ErrorIs(t, err, io.EOF)
	}
	assert.Equal(t, len(testMetrics), rowsRead)

	decodedMetrics := make([]types.Metric, 0, len(decodedParquetMetrics))
	for _, parquetMetric := range decodedParquetMetrics {
		decodedMetrics = append(decodedMetrics, parquetMetric.Metric())
	}

	if diff := cmp.Diff(decodedMetrics, testMetrics); diff != "" {
		t.Errorf("decoded metrics mismatch (-want +got):\n%s", diff)
	}
}

func TestNewParquetStreamer_WrongCompression(t *testing.T) {
	pr, pw := io.Pipe()

	go func() {
		compressor := gzip.NewWriter(pw)
		defer func() {
			compressor.Close()
			pw.Close()
		}()

		encoder := json.NewEncoder(compressor)
		err := encoder.Encode(testMetrics)
		assert.NoError(t, err)
	}()

	parquetStreamer := store.NewParquetStreamer(pr)
	defer parquetStreamer.Close()

	_, err := io.ReadAll(parquetStreamer)
	assert.Error(t, err)
}

func TestNewParquetStreamer_TruncatedJSON(t *testing.T) {
	pr, pw := io.Pipe()

	go func() {
		compressor := brotli.NewWriterLevel(pw, 1)
		defer func() {
			compressor.Close()
			pw.Close()
		}()

		jsonData, err := json.Marshal(testMetrics)
		assert.NoError(t, err)

		compressor.Write(jsonData[:len(jsonData)/2])
	}()

	parquetStreamer := store.NewParquetStreamer(pr)
	defer parquetStreamer.Close()

	_, err := io.ReadAll(parquetStreamer)
	assert.Error(t, err)
}

func TestNewParquetStreamer_TruncatedData(t *testing.T) {
	pr, pw := io.Pipe()

	go func() {
		compressedData := bytes.NewBuffer(nil)

		compressor := brotli.NewWriterLevel(compressedData, 1)

		encoder := json.NewEncoder(compressor)
		err := encoder.Encode(testMetrics)
		assert.NoError(t, err)

		err = compressor.Close()
		assert.NoError(t, err)

		compressedBytes := compressedData.Bytes()
		pw.Write(compressedBytes[:len(compressedBytes)/2])

		err = pw.Close()
		assert.NoError(t, err)
	}()

	parquetStreamer := store.NewParquetStreamer(pr)
	defer parquetStreamer.Close()

	_, err := io.ReadAll(parquetStreamer)
	assert.Error(t, err)
}
