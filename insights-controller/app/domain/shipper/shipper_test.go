// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/cloudzero/cloudzero-insights-controller/app/config/gator"
	"github.com/cloudzero/cloudzero-insights-controller/app/domain/shipper"
	"github.com/cloudzero/cloudzero-insights-controller/app/store"
	"github.com/cloudzero/cloudzero-insights-controller/app/types"
)

func TestShipper_Unit_PerformShipping(t *testing.T) {
	stdout, _ := captureOutput(func() {
		tmpDir := t.TempDir()
		logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
		ctx := logger.WithContext(context.Background())
		settings := &config.Settings{
			Cloudzero: config.Cloudzero{
				SendTimeout:  10,
				SendInterval: time.Second,
				Host:         "http://example.com",
			},
			Database: config.Database{
				StoragePath: tmpDir,
			},
		}

		mockLister := &MockAppendableFiles{baseDir: tmpDir}
		mockLister.On("GetUsage").Return(&types.StoreUsage{PercentUsed: 49}, nil)
		mockLister.On("GetFiles", mock.Anything).Return([]string{}, nil)
		mockLister.On("Walk", mock.Anything, mock.Anything).Return(nil)
		ctx, cancel := context.WithTimeout(ctx, time.Millisecond*1500)
		defer cancel()
		metricShipper, err := shipper.NewMetricShipper(ctx, settings, mockLister)
		require.NoError(t, err)
		err = metricShipper.Run()
		require.NoError(t, err)
		err = metricShipper.Shutdown()
		require.NoError(t, err)
		mockLister.AssertExpectations(t)
	})

	// ensure no errors when running
	require.NotContains(t, stdout, `"level":"error"`)
}

func TestShipper_Unit_Shutdown(t *testing.T) {
	stdout, _ := captureOutput(func() {
		tmpDir := t.TempDir()
		logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
		ctx := logger.WithContext(context.Background())
		settings := &config.Settings{
			Cloudzero: config.Cloudzero{
				SendTimeout:  10,
				SendInterval: time.Second,
				Host:         "http://example.com",
			},
			Database: config.Database{
				StoragePath: tmpDir,
			},
		}

		mockLister := &MockAppendableFiles{baseDir: tmpDir}
		mockLister.On("GetUsage").Return(&types.StoreUsage{PercentUsed: 49}, nil)
		mockLister.On("GetFiles", mock.Anything).Return([]string{}, nil)
		mockLister.On("Walk", mock.Anything, mock.Anything).Return(nil)

		var wg sync.WaitGroup

		// run in a thread
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(ctx, time.Second*5)
			defer cancel()
			metricShipper, err := shipper.NewMetricShipper(ctx, settings, mockLister)
			require.NoError(t, err)
			err = metricShipper.Run()
			require.NoError(t, err)
		}()

		// wait
		time.Sleep(time.Second)

		// send the sigint
		process, err := os.FindProcess(os.Getpid())
		require.NoError(t, err)
		err = process.Signal(syscall.SIGINT)
		require.NoError(t, err)

		// wait for the container to clean up
		wg.Wait()
	})

	// ensure no errors when running
	require.Contains(t, stdout, `Received signal. Initiating shutdown.`)
}

func TestShipper_Unit_GetMetrics(t *testing.T) {
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
	metricShipper, err := shipper.NewMetricShipper(ctx, settings, mockFiles)
	require.NoError(t, err)

	// create a mock handler
	srv := httptest.NewServer(metricShipper.GetMetricHandler())
	defer srv.Close()

	// fetch metrics from the mock handler
	resp, err := http.Get(srv.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.NotEmpty(t, body)
}

func TestShipper_Unit_AllocatePresignedURL_Success(t *testing.T) {
	// Setup
	mockURL := "https://example.com/upload"

	// create some test files
	tmpDir := getTmpDir(t)
	testFiles := createTestFiles(t, tmpDir, 2)

	// create the expected response
	mockResponseBody := map[string]string{}
	for _, item := range testFiles {
		mockResponseBody[shipper.GetRemoteFileID(item)] = "https://s3.amazonaws.com/bucket/file.parquet?signature=abc123"
	}

	mockRoundTripper := &MockRoundTripper{
		status:           http.StatusOK,
		mockResponseBody: mockResponseBody,
		mockError:        nil,
	}

	settings := getMockSettings(mockURL, tmpDir)

	metricShipper, err := shipper.NewMetricShipper(context.Background(), settings, nil)
	require.NoError(t, err)
	metricShipper.HTTPClient.Transport = mockRoundTripper

	// Execute
	require.NoError(t, err)
	urlResponse, err := metricShipper.AllocatePresignedURLs(testFiles)
	require.NoError(t, err)

	// Verify
	require.Equal(t, mockResponseBody, urlResponse)
}

func TestShipper_Unit_AllocatePresignedURL_ReplayRequestHeader(t *testing.T) {
	// Setup
	mockURL := "https://example.com/upload"

	// create some test files
	tmpDir := getTmpDir(t)
	testFiles := createTestFiles(t, tmpDir, 2)

	// create the expected response
	mockResponseBody := map[string]string{}
	for _, item := range testFiles {
		mockResponseBody[shipper.GetRemoteFileID(item)] = "https://s3.amazonaws.com/bucket/file.parquet?signature=abc123"
	}

	// create a replay request header
	headers := http.Header{}
	headers.Add(shipper.ReplayRequestHeader, `[{"ref_id":"id-1"},{"ref_id":"id-2"},{"ref_id":"id-3"}]`)

	mockRoundTripper := &MockRoundTripper{
		status:           http.StatusOK,
		mockResponseBody: mockResponseBody,
		mockError:        nil,
		headers:          headers,
	}

	settings := getMockSettings(mockURL, tmpDir)

	metricShipper, err := shipper.NewMetricShipper(context.Background(), settings, nil)
	require.NoError(t, err)
	metricShipper.HTTPClient.Transport = mockRoundTripper

	// Execute
	require.NoError(t, err)
	urlResponse, err := metricShipper.AllocatePresignedURLs(testFiles)
	require.NoError(t, err)

	// Verify
	require.Equal(t, mockResponseBody, urlResponse)

	// ensure the replay request was parsed and saved to disk
	elems, err := os.ReadDir(filepath.Join(metricShipper.GetBaseDir(), shipper.ReplaySubDirectory))
	require.NoError(t, err)

	require.Equal(t, 1, len(elems))
}

func TestShipper_Unit_AllocatePresignedURL_NoFiles(t *testing.T) {
	// Setup
	tmpDir := getTmpDir(t)
	mockURL := "https://example.com/upload"

	mockRoundTripper := &MockRoundTripper{
		status:    http.StatusOK,
		mockError: nil,
	}

	settings := getMockSettings(mockURL, tmpDir)

	metricShipper, err := shipper.NewMetricShipper(context.Background(), settings, nil)
	require.NoError(t, err)
	metricShipper.HTTPClient.Transport = mockRoundTripper

	// Execute
	require.NoError(t, err)
	urlResponse, err := metricShipper.AllocatePresignedURLs([]types.File{})
	require.Equal(t, err, shipper.ErrNoURLs)
	require.Empty(t, urlResponse)
}

func TestShipper_Unit_AllocatePresignedURL_HTTPError(t *testing.T) {
	// Setup
	tmpDir := getTmpDir(t)
	mockURL := "https://example.com/upload"

	mockResponseBody := map[string]string{
		"error": "invalid request",
	}

	mockRoundTripper := &MockRoundTripper{
		status:           http.StatusBadRequest,
		mockResponseBody: mockResponseBody,
		mockError:        nil,
	}

	settings := getMockSettings(mockURL, tmpDir)

	metricShipper, err := shipper.NewMetricShipper(context.Background(), settings, nil)
	require.NoError(t, err)
	metricShipper.HTTPClient.Transport = mockRoundTripper

	// Execute
	files := createTestFiles(t, tmpDir, 2)
	require.NoError(t, err)
	presignedURL, err := metricShipper.AllocatePresignedURLs(files)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status code 400")
	assert.Empty(t, presignedURL)
}

func TestShiper_Unit_AllocatePresignedURL_Unauthorized(t *testing.T) {
	// Setup
	tmpDir := getTmpDir(t)
	mockURL := "https://example.com/upload"

	mockResponseBody := map[string]string{
		"error": "invalid request",
	}

	mockRoundTripper := &MockRoundTripper{
		status:           http.StatusUnauthorized,
		mockResponseBody: mockResponseBody,
		mockError:        nil,
	}

	settings := getMockSettings(mockURL, tmpDir)

	metricShipper, err := shipper.NewMetricShipper(context.Background(), settings, nil)
	require.NoError(t, err)
	metricShipper.HTTPClient.Transport = mockRoundTripper

	// Execute
	files := createTestFiles(t, tmpDir, 2)
	require.NoError(t, err)
	presignedURL, err := metricShipper.AllocatePresignedURLs(files)

	// Verify
	assert.Error(t, err)
	assert.ErrorIs(t, err, shipper.ErrUnauthorized)
	assert.Empty(t, presignedURL)
}

func TestShipper_Unit_AllocatePresignedURL_EmptyPresignedURL(t *testing.T) {
	// Setup
	tmpDir := getTmpDir(t)
	mockURL := "https://example.com/upload"

	mockResponseBody := map[string]string{}

	mockRoundTripper := &MockRoundTripper{
		status:           http.StatusOK,
		mockResponseBody: mockResponseBody,
		mockError:        nil,
	}

	settings := getMockSettings(mockURL, tmpDir)

	metricShipper, err := shipper.NewMetricShipper(context.Background(), settings, nil)
	require.NoError(t, err)
	metricShipper.HTTPClient.Transport = mockRoundTripper

	// Execute
	files := createTestFiles(t, tmpDir, 2)
	require.NoError(t, err)
	presignedURL, err := metricShipper.AllocatePresignedURLs(files)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no presigned URLs returned")
	assert.Empty(t, presignedURL)
}

func TestShipper_Unit_AllocatePresignedURL_RequestCreationError(t *testing.T) {
	// Setup
	tmpDir := getTmpDir(t)
	// Use an invalid URL to force request creation error
	mockURL := "http://%41:8080/" // Invalid URL

	settings := getMockSettings(mockURL, tmpDir)

	metricShipper, err := shipper.NewMetricShipper(context.Background(), settings, nil)
	require.NoError(t, err)

	// Execute
	files := createTestFiles(t, tmpDir, 2)
	require.NoError(t, err)
	presignedURL, err := metricShipper.AllocatePresignedURLs(files)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get remote base")
	assert.Empty(t, presignedURL)
}

func TestShipper_Unit_AllocatePresignedURL_HTTPClientError(t *testing.T) {
	// Setup
	tmpDir := getTmpDir(t)
	mockURL := "https://example.com/upload"

	mockRoundTripper := &MockRoundTripper{
		mockResponseBody: nil,
		mockError:        errors.New("network error"),
	}

	settings := getMockSettings(mockURL, tmpDir)

	metricShipper, err := shipper.NewMetricShipper(context.Background(), settings, nil)
	require.NoError(t, err)
	metricShipper.HTTPClient.Transport = mockRoundTripper

	// Execute
	files := createTestFiles(t, tmpDir, 2)
	require.NoError(t, err)
	presignedURL, err := metricShipper.AllocatePresignedURLs(files)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP request failed")
	assert.Empty(t, presignedURL)
}

func TestShipper_Unit_UploadFile_Success(t *testing.T) {
	// Setup
	tmpDir := getTmpDir(t)
	mockURL := "https://s3.amazonaws.com/bucket/file.parquet?signature=abc123"

	mockRoundTripper := &MockRoundTripper{
		status:           http.StatusOK,
		mockResponseBody: "",
		mockError:        nil,
	}

	settings := getMockSettings(mockURL, tmpDir)

	metricShipper, err := shipper.NewMetricShipper(context.Background(), settings, nil)
	require.NoError(t, err)
	metricShipper.HTTPClient.Transport = mockRoundTripper

	files := createTestFiles(t, tmpDir, 1)

	// Execute
	err = metricShipper.UploadFile(context.Background(), files[0], mockURL)

	// Verify
	assert.NoError(t, err)
}

func TestShipper_Unit_UploadFile_HTTPError(t *testing.T) {
	tmpDir := getTmpDir(t)
	mockURL := "https://s3.amazonaws.com/bucket/file.parquet?signature=abc123"
	mockResponseBody := "Bad Request"

	mockRoundTripper := &MockRoundTripper{
		status:                 http.StatusBadRequest,
		mockResponseBodyString: mockResponseBody,
		mockError:              nil,
	}

	settings := getMockSettings(mockURL, tmpDir)

	metricShipper, err := shipper.NewMetricShipper(context.Background(), settings, nil)
	require.NoError(t, err)
	metricShipper.HTTPClient.Transport = mockRoundTripper

	files := createTestFiles(t, tmpDir, 1)

	// Execute
	err = metricShipper.UploadFile(context.Background(), files[0], mockURL)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected upload status code 400: Bad Request")
}

func TestShipper_Unit_UploadFile_CreateRequestError(t *testing.T) {
	// Use an invalid URL to force request creation error
	tmpDir := getTmpDir(t)
	mockURL := "http://%41:8080/" // Invalid URL

	settings := getMockSettings(mockURL, tmpDir)

	metricShipper, err := shipper.NewMetricShipper(context.Background(), settings, nil)
	require.NoError(t, err)

	files := createTestFiles(t, tmpDir, 1)

	// Execute
	err = metricShipper.UploadFile(context.Background(), files[0], mockURL)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create upload HTTP request")
}

func TestShipper_Unit_UploadFile_HTTPClientError(t *testing.T) {
	// Setup
	tmpDir := getTmpDir(t)
	mockURL := "https://s3.amazonaws.com/bucket/file.parquet?signature=abc123"
	mockRoundTripper := &MockRoundTripper{
		mockResponseBody: nil,
		mockError:        errors.New("network error"),
	}

	settings := getMockSettings(mockURL, tmpDir)

	metricShipper, err := shipper.NewMetricShipper(context.Background(), settings, nil)
	require.NoError(t, err)
	metricShipper.HTTPClient.Transport = mockRoundTripper

	files := createTestFiles(t, tmpDir, 1)

	// Execute
	err = metricShipper.UploadFile(context.Background(), files[0], mockURL)

	// Verify
	assert.Error(t, err)
}

func TestShipper_Unit_UploadFile_FileOpenError(t *testing.T) {
	// Setup
	tmpDir := getTmpDir(t)
	mockURL := "https://s3.amazonaws.com/bucket/file.parquet?signature=abc123"

	settings := getMockSettings(mockURL, tmpDir)

	_, err := shipper.NewMetricShipper(context.Background(), settings, nil)
	require.NoError(t, err)

	// Use a non-existent file path
	_, err = store.NewMetricFile("/path/to/nonexistent/file.json.br")
	require.Error(t, err)
}

func TestShipper_Unit_AbandonFiles_Success(t *testing.T) {
	// Setup
	tmpDir := getTmpDir(t)
	mockURL := "https://example.com"

	mockResponseBody := map[string]string{
		"message": "Abandon request processed successfully",
	}

	mockRoundTripper := &MockRoundTripper{
		status:           http.StatusOK,
		mockResponseBody: mockResponseBody,
		mockError:        nil,
	}

	settings := getMockSettings(mockURL, tmpDir)

	metricShipper, err := shipper.NewMetricShipper(context.Background(), settings, nil)
	require.NoError(t, err)
	metricShipper.HTTPClient.Transport = mockRoundTripper

	// Execute
	err = metricShipper.AbandonFiles(context.Background(), []string{"file1", "file2"}, "file not found")
	require.NoError(t, err)
}
