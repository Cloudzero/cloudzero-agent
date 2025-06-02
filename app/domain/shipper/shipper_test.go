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

	config "github.com/cloudzero/cloudzero-agent/app/config/gator"
	"github.com/cloudzero/cloudzero-agent/app/domain/shipper"
	"github.com/cloudzero/cloudzero-agent/app/storage/disk"
	"github.com/cloudzero/cloudzero-agent/app/types"
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
	require.Equal(t, mockResponseBody, urlResponse.Allocation)
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
	require.Equal(t, mockResponseBody, urlResponse.Allocation)

	// ensure the replay request was also captured in the url response
	require.Equal(t, 3, len(urlResponse.Replay))
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
	_, err = metricShipper.AllocatePresignedURLs([]types.File{})
	require.NoError(t, err)
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
	assert.ErrorIs(t, err, shipper.ErrHTTPUnknown)
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
	_, err = metricShipper.AllocatePresignedURLs(files)

	// Verify. Recieving no urls should not give an error
	assert.NoError(t, err)
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
	assert.ErrorIs(t, err, shipper.ErrGetRemoteBase)
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
	assert.ErrorIs(t, err, shipper.ErrHTTPRequestFailed)
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
	err = metricShipper.UploadFile(context.Background(), &shipper.UploadFileRequest{
		File:         files[0],
		PresignedURL: mockURL,
	})

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
	err = metricShipper.UploadFile(context.Background(), &shipper.UploadFileRequest{
		File:         files[0],
		PresignedURL: mockURL,
	})

	// Verify
	assert.Error(t, err)
	assert.ErrorIs(t, err, shipper.ErrHTTPUnknown)
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
	err = metricShipper.UploadFile(context.Background(), &shipper.UploadFileRequest{
		File:         files[0],
		PresignedURL: mockURL,
	})

	// Verify
	assert.Error(t, err)
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
	err = metricShipper.UploadFile(context.Background(), &shipper.UploadFileRequest{
		File:         files[0],
		PresignedURL: mockURL,
	})

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
	_, err = disk.NewMetricFile("/path/to/nonexistent/file.json.br")
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
	req := make([]*shipper.AbandonAPIPayloadFile, 0)
	req = append(req, &shipper.AbandonAPIPayloadFile{
		ReferenceID: "file1",
		Reason:      "file not found",
	})
	req = append(req, &shipper.AbandonAPIPayloadFile{
		ReferenceID: "file2",
		Reason:      "file not found",
	})
	err = metricShipper.AbandonFiles(context.Background(), req)
	require.NoError(t, err)
}

func TestUnit_Shipper_ShipperID_Normal(t *testing.T) {
	tmpDir := getTmpDir(t)
	mockLister := &MockAppendableFiles{baseDir: tmpDir}
	settings := getMockSettings("", tmpDir)
	metricShipper, err := shipper.NewMetricShipper(context.Background(), settings, mockLister)
	require.NoError(t, err, "failed to create the metric shipper")

	id, err := metricShipper.GetShipperID()
	require.NoError(t, err, "failed to get the shipperId")
	require.NotEmpty(t, id, "invalid id")

	id2, err := metricShipper.GetShipperID()
	require.NoError(t, err, "failed to get the shipperId the second time")
	require.Equal(t, id, id2, "the second call to GetShipperId returned a different value")
}

func TestUnit_Shipper_ShipperID_FromFile(t *testing.T) {
	tmpDir := getTmpDir(t)
	mockLister := &MockAppendableFiles{baseDir: tmpDir}
	settings := getMockSettings("", tmpDir)
	metricShipper, err := shipper.NewMetricShipper(context.Background(), settings, mockLister)
	require.NoError(t, err, "failed to create the metric shipper")

	expected := "shipper-id"

	// write a file
	err = os.WriteFile(filepath.Join(metricShipper.GetBaseDir(), ".shipperid"), []byte(expected), 0o755)
	require.NoError(t, err, "failed to create the shipperid file")

	// get the shipperid
	id, err := metricShipper.GetShipperID()
	require.NoError(t, err, "failed to get the shipper id")

	require.Equal(t, expected, id)
}

func TestUnit_Shipper_ShipperID_FromEnv(t *testing.T) {
	tmpDir := getTmpDir(t)
	mockLister := &MockAppendableFiles{baseDir: tmpDir}
	settings := getMockSettings("", tmpDir)
	metricShipper, err := shipper.NewMetricShipper(context.Background(), settings, mockLister)
	require.NoError(t, err, "failed to create the metric shipper")

	expected := "shipper-id"

	// set the env variable
	os.Setenv("HOSTNAME", expected)

	// get the shipper id
	id, err := metricShipper.GetShipperID()
	require.NoError(t, err, "failed to get the shipper id")

	require.Equal(t, expected, id)
}
