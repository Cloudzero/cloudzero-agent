// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package store_test

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/andybalholm/brotli"
	"github.com/cloudzero/cloudzero-insights-controller/app/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricFile_ReadAll(t *testing.T) {
	tmpDir := t.TempDir()

	path := filepath.Join(tmpDir, "test-file.json.br")
	osFile, err := os.Create(path)
	require.NoError(t, err)

	// write to the os file
	func() {
		compressor := brotli.NewWriterLevel(osFile, 1)
		defer func() {
			compressor.Close()
			osFile.Close()
		}()

		encoder := json.NewEncoder(compressor)
		err := encoder.Encode(testMetrics)
		assert.NoError(t, err)
	}()

	// create a new metric file with this
	file, err := store.NewMetricFile(path)
	require.NoError(t, err)

	// read the data
	_, err = io.ReadAll(file)
	require.NoError(t, err)
}
