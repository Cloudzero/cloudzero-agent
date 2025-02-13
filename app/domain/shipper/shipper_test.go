// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudzero/cloudzero-insights-controller/app/config"
)

func TestPerformShipping(t *testing.T) {
	stdout, _ := captureOutput(func() {
		logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
		ctx := logger.WithContext(context.Background())
		settings := &config.Settings{
			Cloudzero: config.Cloudzero{
				SendTimeout:  10,
				SendInterval: time.Second,
				Host:         "http://example.com",
			},
			Database: config.Database{
				StoragePath: t.TempDir(),
			},
		}

		mockFiles := &MockAppendableFiles{}
		mockFiles.On("GetFiles").Return([]string{}, nil)
		ctx, cancel := context.WithTimeout(ctx, time.Millisecond*1500)
		defer cancel()
		shipper, err := NewMetricShipper(ctx, settings, mockFiles)
		require.NoError(t, err)
		err = shipper.Run()
		require.NoError(t, err)
		err = shipper.Shutdown()
		require.NoError(t, err)
		mockFiles.AssertExpectations(t)
	})

	// ensure no errors when running
	require.NotContains(t, stdout, `"level":"error"`)
}

func TestGetMetrics(t *testing.T) {
	ctx := context.Background()
	settings := &config.Settings{
		Cloudzero: config.Cloudzero{
			SendTimeout:  10,
			SendInterval: 1,
			Host:         "http://example.com",
		},
		Database: config.Database{
			StoragePath: t.TempDir(),
		},
	}

	mockFiles := &MockAppendableFiles{}
	mockFiles.On("GetFiles").Return([]string{}, nil)
	shipper, err := NewMetricShipper(ctx, settings, mockFiles)
	require.NoError(t, err)

	// create a mock handler
	srv := httptest.NewServer(shipper.GetMetricHandler())
	defer srv.Close()

	// fetch metrics from the mock handler
	resp, err := http.Get(srv.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.NotEmpty(t, body)
}

func TestAllocatePresignedURL_Success(t *testing.T) {
	// Setup
	mockURL := "https://example.com/upload"
	expectedURL := "https://s3.amazonaws.com/bucket/file.tgz?signature=abc123"

	mockResponseBody := map[string]string{
		"file1": expectedURL,
		"file2": expectedURL,
	}

	mockRoundTripper := &MockRoundTripper{
		status:           http.StatusOK,
		mockResponseBody: mockResponseBody,
		mockError:        nil,
	}

	settings := setupSettings(mockURL)

	shipper, err := NewMetricShipper(context.Background(), settings, nil)
	require.NoError(t, err)
	shipper.HTTPClient.Transport = mockRoundTripper

	// Execute
	files, err := NewFilesFromPaths([]string{"file1", "file2"})
	require.NoError(t, err)
	files, err = shipper.AllocatePresignedURLs(files)
	require.NoError(t, err)

	presignedURLs := make([]string, len(files))
	for index, file := range files {
		presignedURLs[index] = file.PresignedURL
	}

	// Verify
	assert.Equal(t, []string{expectedURL, expectedURL}, presignedURLs)
}

func TestAllocatePresignedURL_NoFiles(t *testing.T) {
	// Setup
	settings := setupSettings("https://example.com/upload")

	shipper, err := NewMetricShipper(context.Background(), settings, nil)
	require.NoError(t, err)

	// Execute
	presignedURLs, err := shipper.AllocatePresignedURLs([]*File{})

	// Verify
	assert.NoError(t, err)
	assert.Nil(t, presignedURLs)
}

func TestAllocatePresignedURL_HTTPError(t *testing.T) {
	// Setup
	mockURL := "https://example.com/upload"

	mockResponseBody := map[string]string{
		"error": "invalid request",
	}

	mockRoundTripper := &MockRoundTripper{
		status:           http.StatusBadRequest,
		mockResponseBody: mockResponseBody,
		mockError:        nil,
	}

	settings := setupSettings(mockURL)

	shipper, err := NewMetricShipper(context.Background(), settings, nil)
	require.NoError(t, err)
	shipper.HTTPClient.Transport = mockRoundTripper

	// Execute
	files, err := NewFilesFromPaths([]string{"file1"})
	require.NoError(t, err)
	presignedURL, err := shipper.AllocatePresignedURLs(files)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status code 400")
	assert.Empty(t, presignedURL)
}

func TestAllocatePresignedURL_Unauthorized(t *testing.T) {
	// Setup
	mockURL := "https://example.com/upload"

	mockResponseBody := map[string]string{
		"error": "invalid request",
	}

	mockRoundTripper := &MockRoundTripper{
		status:           http.StatusUnauthorized,
		mockResponseBody: mockResponseBody,
		mockError:        nil,
	}

	settings := setupSettings(mockURL)

	shipper, err := NewMetricShipper(context.Background(), settings, nil)
	require.NoError(t, err)
	shipper.HTTPClient.Transport = mockRoundTripper

	// Execute
	files, err := NewFilesFromPaths([]string{"file1"})
	require.NoError(t, err)
	presignedURL, err := shipper.AllocatePresignedURLs(files)

	// Verify
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrUnauthorized)
	assert.Empty(t, presignedURL)
}

func TestAllocatePresignedURL_EmptyPresignedURL(t *testing.T) {
	// Setup
	mockURL := "https://example.com/upload"

	mockResponseBody := map[string]string{}

	mockRoundTripper := &MockRoundTripper{
		status:           http.StatusOK,
		mockResponseBody: mockResponseBody,
		mockError:        nil,
	}

	settings := setupSettings(mockURL)

	shipper, err := NewMetricShipper(context.Background(), settings, nil)
	require.NoError(t, err)
	shipper.HTTPClient.Transport = mockRoundTripper

	// Execute
	files, err := NewFilesFromPaths([]string{"file1"})
	require.NoError(t, err)
	presignedURL, err := shipper.AllocatePresignedURLs(files)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no presigned URLs returned")
	assert.Empty(t, presignedURL)
}

func TestAllocatePresignedURL_RequestCreationError(t *testing.T) {
	// Setup
	// Use an invalid URL to force request creation error
	mockURL := "http://%41:8080/" // Invalid URL

	settings := setupSettings(mockURL)

	shipper, err := NewMetricShipper(context.Background(), settings, nil)
	require.NoError(t, err)

	// Execute
	files, err := NewFilesFromPaths([]string{"file1"})
	require.NoError(t, err)
	presignedURL, err := shipper.AllocatePresignedURLs(files)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create HTTP request")
	assert.Empty(t, presignedURL)
}

func TestAllocatePresignedURL_HTTPClientError(t *testing.T) {
	// Setup
	mockURL := "https://example.com/upload"

	mockRoundTripper := &MockRoundTripper{
		mockResponseBody: nil,
		mockError:        errors.New("network error"),
	}

	settings := setupSettings(mockURL)

	shipper, err := NewMetricShipper(context.Background(), settings, nil)
	require.NoError(t, err)
	shipper.HTTPClient.Transport = mockRoundTripper

	// Execute
	files, err := NewFilesFromPaths([]string{"file1"})
	require.NoError(t, err)
	presignedURL, err := shipper.AllocatePresignedURLs(files)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP request failed")
	assert.Empty(t, presignedURL)
}

func TestUploadFile_Success(t *testing.T) {
	// Setup
	mockURL := "https://s3.amazonaws.com/bucket/file.tgz?signature=abc123"

	mockRoundTripper := &MockRoundTripper{
		status:           http.StatusOK,
		mockResponseBody: "",
		mockError:        nil,
	}

	settings := setupSettings(mockURL)

	shipper, err := NewMetricShipper(context.Background(), settings, nil)
	require.NoError(t, err)
	shipper.HTTPClient.Transport = mockRoundTripper

	// Create a temporary file to upload
	tempFile, err := os.CreateTemp("", "testfile-*.tgz")
	assert.NoError(t, err)
	defer os.Remove(tempFile.Name())

	// Write some data to the file
	_, err = tempFile.WriteString("test data")
	assert.NoError(t, err)
	tempFile.Close()

	// create the file obj
	file, err := NewFile(tempFile.Name(), FileWithPresignedURL(mockURL))
	require.NoError(t, err)

	// Execute
	err = shipper.Upload(file)

	// Verify
	assert.NoError(t, err)
}

func TestUploadFile_HTTPError(t *testing.T) {
	mockURL := "https://s3.amazonaws.com/bucket/file.tgz?signature=abc123"
	mockResponseBody := "Bad Request"

	mockRoundTripper := &MockRoundTripper{
		status:                 http.StatusBadRequest,
		mockResponseBodyString: mockResponseBody,
		mockError:              nil,
	}

	settings := setupSettings(mockURL)

	shipper, err := NewMetricShipper(context.Background(), settings, nil)
	require.NoError(t, err)
	shipper.HTTPClient.Transport = mockRoundTripper

	// Create a temporary file to upload
	tempFile, err := os.CreateTemp("", "testfile-*.tgz")
	assert.NoError(t, err)
	defer os.Remove(tempFile.Name())

	// Write some data to the file
	_, err = tempFile.WriteString("test data")
	assert.NoError(t, err)
	tempFile.Close()

	file, err := NewFile(tempFile.Name(), FileWithPresignedURL(mockURL))
	require.NoError(t, err)

	// Execute
	err = shipper.Upload(file)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected upload status code 400: Bad Request")
}

func TestUploadFile_CreateRequestError(t *testing.T) {
	// Use an invalid URL to force request creation error
	mockURL := "http://%41:8080/" // Invalid URL

	settings := setupSettings(mockURL)

	shipper, err := NewMetricShipper(context.Background(), settings, nil)
	require.NoError(t, err)

	// Create a temporary file to upload
	tempFile, err := os.CreateTemp("", "testfile-*.tgz")
	assert.NoError(t, err)
	defer os.Remove(tempFile.Name())

	// Write some data to the file
	_, err = tempFile.WriteString("test data")
	assert.NoError(t, err)
	tempFile.Close()

	file, err := NewFile(tempFile.Name(), FileWithPresignedURL(mockURL))
	require.NoError(t, err)

	// Execute
	err = shipper.Upload(file)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create upload HTTP request")
}

func TestUploadFile_HTTPClientError(t *testing.T) {
	// Setup
	mockURL := "https://s3.amazonaws.com/bucket/file.tgz?signature=abc123"
	mockRoundTripper := &MockRoundTripper{
		mockResponseBody: nil,
		mockError:        errors.New("network error"),
	}

	settings := setupSettings(mockURL)

	shipper, err := NewMetricShipper(context.Background(), settings, nil)
	require.NoError(t, err)
	shipper.HTTPClient.Transport = mockRoundTripper

	// Create a temporary file to upload
	tempFile, err := os.CreateTemp("", "testfile-*.tgz")
	assert.NoError(t, err)
	defer os.Remove(tempFile.Name())

	// Write some data to the file
	_, err = tempFile.WriteString("test data")
	assert.NoError(t, err)
	tempFile.Close()

	file, err := NewFile(tempFile.Name(), FileWithPresignedURL(mockURL))
	require.NoError(t, err)

	// Execute
	err = shipper.Upload(file)

	// Verify
	assert.Error(t, err)
}

func TestUploadFile_FileOpenError(t *testing.T) {
	// Setup
	mockURL := "https://s3.amazonaws.com/bucket/file.tgz?signature=abc123"

	settings := setupSettings(mockURL)

	_, err := NewMetricShipper(context.Background(), settings, nil)
	require.NoError(t, err)

	// Use a non-existent file path
	file, err := NewFile("/path/to/nonexistent/file.tgz", FileWithPresignedURL(mockURL))
	require.NoError(t, err)

	// read the file
	_, err = file.ReadFile()
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed to open the file")
}

func TestAbandonFiles_Success(t *testing.T) {
	// Setup
	mockURL := "https://example.com"

	mockResponseBody := map[string]string{
		"message": "Abandon request processed successfully",
	}

	mockRoundTripper := &MockRoundTripper{
		status:           http.StatusOK,
		mockResponseBody: mockResponseBody,
		mockError:        nil,
	}

	settings := setupSettings(mockURL)

	shipper, err := NewMetricShipper(context.Background(), settings, nil)
	require.NoError(t, err)
	shipper.HTTPClient.Transport = mockRoundTripper

	// Execute
	err = shipper.AbandonFiles([]string{"file1", "file2"}, "file not found")
	require.NoError(t, err)
}
