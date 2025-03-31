// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package compress_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cloudzero/cloudzero-insights-controller/app/compress"
)

func TestFile(t *testing.T) {
	// Create a temporary file to be compressed
	tmpFile, err := os.CreateTemp("", "testfile")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Write some data to the temporary file
	_, err = tmpFile.WriteString("This is a test file.")
	assert.NoError(t, err)
	tmpFile.Close()

	// Call the File function to compress the temporary file
	compressedFilePath, err := compress.File(tmpFile.Name())
	assert.NoError(t, err)
	assert.NotNil(t, compressedFilePath)

	// Check if the compressed file exists
	_, err = os.Stat(*compressedFilePath)
	assert.NoError(t, err)

	// Clean up the compressed file
	err = os.Remove(*compressedFilePath)
	assert.NoError(t, err)
}

func TestFile_FileDoesNotExist(t *testing.T) {
	// Call the File function with a non-existent file path
	compressedFilePath, err := compress.File("non_existent_file.txt")
	assert.NoError(t, err)
	assert.Nil(t, compressedFilePath)
}

func TestFile_EmptyFile(t *testing.T) {
	// Create a temporary empty file to be compressed
	tmpFile, err := os.CreateTemp("", "emptyfile")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Call the File function to compress the empty file
	compressedFilePath, err := compress.File(tmpFile.Name())
	assert.NoError(t, err)
	assert.NotNil(t, compressedFilePath)

	// Check if the compressed file exists
	_, err = os.Stat(*compressedFilePath)
	assert.NoError(t, err)

	// Clean up the compressed file
	err = os.Remove(*compressedFilePath)
	assert.NoError(t, err)
}

func TestFile_LargeFile(t *testing.T) {
	// Create a temporary large file to be compressed
	tmpFile, err := os.CreateTemp("", "largefile")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Write a large amount of data to the temporary file
	largeData := make([]byte, 10*1024*1024) // 10 MB
	_, err = tmpFile.Write(largeData)
	assert.NoError(t, err)
	tmpFile.Close()

	// Call the File function to compress the large file
	compressedFilePath, err := compress.File(tmpFile.Name())
	assert.NoError(t, err)
	assert.NotNil(t, compressedFilePath)

	// Check if the compressed file exists
	_, err = os.Stat(*compressedFilePath)
	assert.NoError(t, err)

	// Clean up the compressed file
	err = os.Remove(*compressedFilePath)
	assert.NoError(t, err)
}

func TestFile_MissingFile(t *testing.T) {
	// Call the File function with a missing file path
	compressedFilePath, err := compress.File("")
	assert.NoError(t, err)
	assert.Nil(t, compressedFilePath)
}

func TestFile_Directory(t *testing.T) {
	// Create a temporary directory to be compressed
	tmpDir, err := os.MkdirTemp("", "testdir")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Call the File function to compress the directory
	compressedFilePath, err := compress.File(tmpDir)
	assert.Error(t, err)
	assert.Nil(t, compressedFilePath)
}
