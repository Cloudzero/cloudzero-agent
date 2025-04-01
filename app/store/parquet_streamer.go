// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package store provides storage functionality.
package store

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/andybalholm/brotli"
	"github.com/cloudzero/cloudzero-agent-validator/app/types"
	"github.com/parquet-go/parquet-go"
)

const (
	parquetBufferSize = 16384
)

// NewParquetStreamer reads a Brotli-compressed JSON file containing an array of
// Metrics, and returns a reader with the data transcoded to Snappy-compressed
// Parquet.
func NewParquetStreamer(input io.Reader) io.ReadCloser {
	decompressor := brotli.NewReader(input)

	decoder := json.NewDecoder(decompressor)
	decoder.DisallowUnknownFields()

	pipeReader, pipeWriter := io.Pipe()

	parquetWriter := parquet.NewGenericWriter[types.ParquetMetric](pipeWriter, parquet.Compression(&parquet.Snappy))

	go func() {
		defer func() {
			parquetWriter.Close()
			pipeWriter.Close()
			// decompressor.Close() // Necessary for cbrotli, but not the Go-native version
		}()

		if firstToken, err := decoder.Token(); err != nil {
			pipeWriter.CloseWithError(fmt.Errorf("failed to read first token from JSON: %w", err))
			return
		} else if firstToken != json.Delim('[') {
			pipeWriter.CloseWithError(fmt.Errorf("expected '[' at the beginning of the file, got %s", firstToken))
			return
		}

		for decoder.More() {
			var metrics []types.ParquetMetric = make([]types.ParquetMetric, 0, parquetBufferSize)

			for i := 0; i < parquetBufferSize && decoder.More(); i++ {
				var metric types.Metric
				if err := decoder.Decode(&metric); err != nil {
					pipeWriter.CloseWithError(fmt.Errorf("failed to decode JSON: %w", err))
					return
				}
				metrics = append(metrics, metric.Parquet())
			}

			_, err := parquetWriter.Write(metrics)
			if err != nil {
				pipeWriter.CloseWithError(fmt.Errorf("failed to write metrics to Parquet: %w", err))
				return
			}
		}

		if lastToken, err := decoder.Token(); err != nil {
			pipeWriter.CloseWithError(fmt.Errorf("failed to read last token from JSON: %w", err))
			return
		} else if lastToken != json.Delim(']') {
			pipeWriter.CloseWithError(fmt.Errorf("expected ']' at the end of the file, got %s", lastToken))
			return
		}
	}()

	return pipeReader
}
