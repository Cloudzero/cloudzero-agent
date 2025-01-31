// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package domain_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/cloudzero/cloudzero-insights-controller/app/config"
	"github.com/cloudzero/cloudzero-insights-controller/app/domain"
)

type MockAppendableFiles struct {
	mock.Mock
}

func (m *MockAppendableFiles) GetFiles() ([]string, error) {
	args := m.Called()
	return args.Get(0).([]string), args.Error(1)
}

// MockRoundTripper is a mock implementation of http.RoundTripper
type MockRoundTripper struct {
	mockResponse *http.Response
	mockError    error
}

func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.mockResponse, m.mockError
}

func setupSettings(mockURL string) *config.Settings {
	return &config.Settings{
		ClusterName:    "test-cluster",
		CloudAccountID: "test-account",
		Region:         "us-east-1",
		Cloudzero: config.Cloudzero{
			Host:        mockURL,
			SendTimeout: 30,
		},
		Database: config.Database{
			StoragePath: "/tmp/storage",
		},
	}
}

func TestPerformShipping(t *testing.T) {
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
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	shipper := domain.NewMetricShipper(ctx, settings, mockFiles)
	shipper.Run()
	shipper.Shutdown()
	mockFiles.AssertExpectations(t)
}

func TestGetStatus(t *testing.T) {
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
	shipper := domain.NewMetricShipper(ctx, settings, mockFiles)
	stat, err := shipper.GetStatus()
	assert.NoError(t, err)
	assert.Equal(t, 0, stat.ShippableFiles)
	assert.Equal(t, uint64(0), stat.ShippedFiles)
}

func TestAllocatePresignedURL_Success(t *testing.T) {
	// Setup
	mockURL := "https://example.com/upload"
	expectedURL := "https://s3.amazonaws.com/bucket/file.tgz?signature=abc123"

	mockResponseBody := `{"urls": ["` + expectedURL + `", "` + expectedURL + `"]}`

	mockResponse := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(mockResponseBody)),
	}

	mockRoundTripper := &MockRoundTripper{
		mockResponse: mockResponse,
		mockError:    nil,
	}

	settings := setupSettings(mockURL)

	shipper := domain.NewMetricShipper(context.Background(), settings, nil)
	shipper.HttpClient.Transport = mockRoundTripper

	// Execute
	presignedURLs, err := shipper.AllocatePresignedURLs(2)

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, []string{expectedURL, expectedURL}, presignedURLs)
}

func TestAllocatePresignedURL_NoFiles(t *testing.T) {
	// Setup
	settings := setupSettings("https://example.com/upload")

	shipper := domain.NewMetricShipper(context.Background(), settings, nil)

	// Execute
	presignedURLs, err := shipper.AllocatePresignedURLs(0)

	// Verify
	assert.NoError(t, err)
	assert.Nil(t, presignedURLs)
}

func TestAllocatePresignedURL_HTTPError(t *testing.T) {
	// Setup
	mockURL := "https://example.com/upload"

	mockResponseBody := `{"error": "invalid request"}`

	mockResponse := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       io.NopCloser(bytes.NewBufferString(mockResponseBody)),
	}

	mockRoundTripper := &MockRoundTripper{
		mockResponse: mockResponse,
		mockError:    nil,
	}

	settings := setupSettings(mockURL)

	shipper := domain.NewMetricShipper(context.Background(), settings, nil)
	shipper.HttpClient.Transport = mockRoundTripper

	// Execute
	presignedURL, err := shipper.AllocatePresignedURLs(1)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status code 400")
	assert.Empty(t, presignedURL)
}

func TestAllocatePresignedURL_Unauthorized(t *testing.T) {
	// Setup
	mockURL := "https://example.com/upload"

	mockResponseBody := `{"error": "invalid request"}`

	mockResponse := &http.Response{
		StatusCode: http.StatusUnauthorized,
		Body:       io.NopCloser(bytes.NewBufferString(mockResponseBody)),
	}

	mockRoundTripper := &MockRoundTripper{
		mockResponse: mockResponse,
		mockError:    nil,
	}

	settings := setupSettings(mockURL)

	shipper := domain.NewMetricShipper(context.Background(), settings, nil)
	shipper.HttpClient.Transport = mockRoundTripper

	// Execute
	presignedURL, err := shipper.AllocatePresignedURLs(1)

	// Verify
	assert.Error(t, err)
	assert.ErrorIs(t, err, domain.ErrUnauthorized)
	assert.Empty(t, presignedURL)
}

func TestAllocatePresignedURL_MalformedResponse(t *testing.T) {
	// Setup
	mockURL := "https://example.com/upload"

	// Malformed JSON
	mockResponseBody := `{"urls": ["https://s3.amazonaws.com/bucket/file.tgz"`

	mockResponse := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(mockResponseBody)),
	}

	mockRoundTripper := &MockRoundTripper{
		mockResponse: mockResponse,
		mockError:    nil,
	}

	settings := setupSettings(mockURL)

	shipper := domain.NewMetricShipper(context.Background(), settings, nil)
	shipper.HttpClient.Transport = mockRoundTripper

	// Execute
	presignedURL, err := shipper.AllocatePresignedURLs(1)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode response")
	assert.Empty(t, presignedURL)
}

func TestAllocatePresignedURL_EmptyPresignedURL(t *testing.T) {
	// Setup
	mockURL := "https://example.com/upload"

	mockResponseBody := `{"urls": []}`

	mockResponse := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(mockResponseBody)),
	}

	mockRoundTripper := &MockRoundTripper{
		mockResponse: mockResponse,
		mockError:    nil,
	}

	settings := setupSettings(mockURL)

	shipper := domain.NewMetricShipper(context.Background(), settings, nil)
	shipper.HttpClient.Transport = mockRoundTripper

	// Execute
	presignedURL, err := shipper.AllocatePresignedURLs(1)

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

	shipper := domain.NewMetricShipper(context.Background(), settings, nil)

	// Execute
	presignedURL, err := shipper.AllocatePresignedURLs(1)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create HTTP request")
	assert.Empty(t, presignedURL)
}

func TestAllocatePresignedURL_HTTPClientError(t *testing.T) {
	// Setup
	mockURL := "https://example.com/upload"

	mockRoundTripper := &MockRoundTripper{
		mockResponse: nil,
		mockError:    errors.New("network error"),
	}

	settings := setupSettings(mockURL)

	shipper := domain.NewMetricShipper(context.Background(), settings, nil)
	shipper.HttpClient.Transport = mockRoundTripper

	// Execute
	presignedURL, err := shipper.AllocatePresignedURLs(1)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP request failed")
	assert.Empty(t, presignedURL)
}

func TestUploadFile_Success(t *testing.T) {
	// Setup
	mockURL := "https://s3.amazonaws.com/bucket/file.tgz?signature=abc123"

	mockResponse := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString("")),
	}
	mockRoundTripper := &MockRoundTripper{
		mockResponse: mockResponse,
		mockError:    nil,
	}

	settings := setupSettings(mockURL)

	shipper := domain.NewMetricShipper(context.Background(), settings, nil)
	shipper.HttpClient.Transport = mockRoundTripper

	// Create a temporary file to upload
	tempFile, err := os.CreateTemp("", "testfile-*.tgz")
	assert.NoError(t, err)
	defer os.Remove(tempFile.Name())

	// Write some data to the file
	_, err = tempFile.WriteString("test data")
	assert.NoError(t, err)
	tempFile.Close()

	// Execute
	err = shipper.UploadFile(mockURL, tempFile.Name())

	// Verify
	assert.NoError(t, err)
}

func TestUploadFile_HTTPError(t *testing.T) {
	mockURL := "https://s3.amazonaws.com/bucket/file.tgz?signature=abc123"
	mockResponseBody := "Bad Request"
	mockResponse := &http.Response{
		StatusCode: http.StatusBadRequest,
		Body:       io.NopCloser(bytes.NewBufferString(mockResponseBody)),
	}
	mockRoundTripper := &MockRoundTripper{
		mockResponse: mockResponse,
		mockError:    nil,
	}

	settings := setupSettings(mockURL)

	shipper := domain.NewMetricShipper(context.Background(), settings, nil)
	shipper.HttpClient.Transport = mockRoundTripper

	// Create a temporary file to upload
	tempFile, err := os.CreateTemp("", "testfile-*.tgz")
	assert.NoError(t, err)
	defer os.Remove(tempFile.Name())

	// Write some data to the file
	_, err = tempFile.WriteString("test data")
	assert.NoError(t, err)
	tempFile.Close()

	// Execute
	err = shipper.UploadFile(mockURL, tempFile.Name())

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected upload status code 400: Bad Request")
}

func TestUploadFile_CreateRequestError(t *testing.T) {
	// Use an invalid URL to force request creation error
	mockURL := "http://%41:8080/" // Invalid URL

	settings := setupSettings(mockURL)

	shipper := domain.NewMetricShipper(context.Background(), settings, nil)

	// Create a temporary file to upload
	tempFile, err := os.CreateTemp("", "testfile-*.tgz")
	assert.NoError(t, err)
	defer os.Remove(tempFile.Name())

	// Write some data to the file
	_, err = tempFile.WriteString("test data")
	assert.NoError(t, err)
	tempFile.Close()

	// Execute
	err = shipper.UploadFile(mockURL, tempFile.Name())

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create upload HTTP request")
}

func TestUploadFile_HTTPClientError(t *testing.T) {
	// Setup
	mockURL := "https://s3.amazonaws.com/bucket/file.tgz?signature=abc123"
	mockRoundTripper := &MockRoundTripper{
		mockResponse: nil,
		mockError:    errors.New("network error"),
	}

	settings := setupSettings(mockURL)

	shipper := domain.NewMetricShipper(context.Background(), settings, nil)
	shipper.HttpClient.Transport = mockRoundTripper

	// Create a temporary file to upload
	tempFile, err := os.CreateTemp("", "testfile-*.tgz")
	assert.NoError(t, err)
	defer os.Remove(tempFile.Name())

	// Write some data to the file
	_, err = tempFile.WriteString("test data")
	assert.NoError(t, err)
	tempFile.Close()

	// Execute
	err = shipper.UploadFile(mockURL, tempFile.Name())

	// Verify
	assert.Error(t, err)
}

func TestUploadFile_FileOpenError(t *testing.T) {
	// Setup
	mockURL := "https://s3.amazonaws.com/bucket/file.tgz?signature=abc123"

	settings := setupSettings(mockURL)

	shipper := domain.NewMetricShipper(context.Background(), settings, nil)

	// Use a non-existent file path
	nonExistentFile := "/path/to/nonexistent/file.tgz"

	// Execute
	err := shipper.UploadFile(mockURL, nonExistentFile)

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open file for upload")
}
