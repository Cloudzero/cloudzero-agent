// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package disk_test

import (
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/andybalholm/brotli"
	"github.com/cloudzero/cloudzero-agent/app/storage/disk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnit_Storage_Disk_MetricFile_ReadAll(t *testing.T) {
	tmpDir := t.TempDir()

	osFile, err := os.CreateTemp(tmpDir, "test-file-*.json.br")
	require.NoError(t, err)

	// write to the os file
	compressor := brotli.NewWriterLevel(osFile, 1)
	defer func() {
		compressor.Close()
		osFile.Close()
	}()

	encoder := json.NewEncoder(compressor)
	err = encoder.Encode(testMetrics)
	assert.NoError(t, err)

	// create a new metric file with this
	file, err := disk.NewMetricFile(osFile.Name())
	require.NoError(t, err)

	// get a unique id
	require.NotEmpty(t, file.UniqueID(), "failed to get the unique id")

	// get the location
	_ = file.Location()

	// get the filesize
	s, err := file.Size()
	require.NoError(t, err, "failed to get the size")
	require.NotEqual(t, 0, s)

	// read the data
	_, err = io.ReadAll(file)
	require.NoError(t, err)

	// close the file
	require.NoError(t, file.Close(), "failed to close the file")
}
