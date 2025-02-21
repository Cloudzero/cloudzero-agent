// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper_test

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"testing"

	"github.com/andybalholm/brotli"
	"github.com/cloudzero/cloudzero-insights-controller/app/domain/shipper"
	"github.com/cloudzero/cloudzero-insights-controller/app/types"
	"github.com/google/go-cmp/cmp"
	"github.com/parquet-go/parquet-go"
	"github.com/stretchr/testify/assert"
)

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

	parquetStreamer := shipper.NewParquetStreamer(pr)
	defer parquetStreamer.Close()

	parquetData, err := io.ReadAll(parquetStreamer)
	assert.NoError(t, err)

	parquetReader := parquet.NewGenericReader[types.Metric](bytes.NewReader(parquetData))
	defer parquetReader.Close()
	assert.Equal(t, len(testMetrics), int(parquetReader.NumRows()))

	decodedMetrics := make([]types.Metric, len(testMetrics))
	rowsRead, err := parquetReader.Read(decodedMetrics)
	assert.NoError(t, err)
	assert.Equal(t, len(testMetrics), rowsRead)

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

	parquetStreamer := shipper.NewParquetStreamer(pr)
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

	parquetStreamer := shipper.NewParquetStreamer(pr)
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

	parquetStreamer := shipper.NewParquetStreamer(pr)
	defer parquetStreamer.Close()

	_, err := io.ReadAll(parquetStreamer)
	assert.Error(t, err)
}
