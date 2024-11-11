package domain_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/cloudzero/cirrus-remote-write/app/config"
	"github.com/cloudzero/cirrus-remote-write/app/domain"
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
			Host:              mockURL,
			APIKey:            "test-api-key",
			LockStaleDuration: 10 * time.Minute,
			SendTimeout:       30,
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
			SendTimeout:       10,
			SendInterval:      1,
			Host:              "http://example.com",
			APIKey:            "test-api-key",
			LockStaleDuration: 10 * time.Minute,
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
			SendTimeout:       10,
			SendInterval:      1,
			Host:              "http://example.com",
			APIKey:            "test-api-key",
			LockStaleDuration: 10 * time.Minute,
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

	mockResponseBody := `{"presigned_url": "` + expectedURL + `"}`

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
	presignedURL, err := shipper.AllocatePresignedURL("file.tgz")

	// Verify
	assert.NoError(t, err)
	assert.Equal(t, expectedURL, presignedURL)
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
	presignedURL, err := shipper.AllocatePresignedURL("file.tgz")

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status code 400")
	assert.Empty(t, presignedURL)
}

func TestAllocatePresignedURL_MalformedResponse(t *testing.T) {
	// Setup
	mockURL := "https://example.com/upload"

	// Malformed JSON
	mockResponseBody := `{"presigned_url": "https://s3.amazonaws.com/bucket/file.tgz"`

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
	presignedURL, err := shipper.AllocatePresignedURL("file.tgz")

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode response")
	assert.Empty(t, presignedURL)
}

func TestAllocatePresignedURL_EmptyPresignedURL(t *testing.T) {
	// Setup
	mockURL := "https://example.com/upload"

	mockResponseBody := `{"presigned_url": ""}`

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
	presignedURL, err := shipper.AllocatePresignedURL("file.tgz")

	// Verify
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "presigned_url is empty")
	assert.Empty(t, presignedURL)
}

func TestAllocatePresignedURL_RequestCreationError(t *testing.T) {
	// Setup
	// Use an invalid URL to force request creation error
	mockURL := "http://%41:8080/" // Invalid URL

	settings := setupSettings(mockURL)

	shipper := domain.NewMetricShipper(context.Background(), settings, nil)

	// Execute
	presignedURL, err := shipper.AllocatePresignedURL("file.tgz")

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
	presignedURL, err := shipper.AllocatePresignedURL("file.tgz")

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

// TestLockFile_Success tests successful locking of a file when no lock exists.
func TestLockFile_Success(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "testfile.txt")
	// Create the file to lock
	err := os.WriteFile(filePath, []byte("test data"), 0644)
	assert.NoError(t, err)

	settings := setupSettings("")

	shipper := domain.NewMetricShipper(context.Background(), settings, nil)

	// Execute
	locked, err := shipper.LockFile(filePath)

	// Verify
	assert.NoError(t, err)
	assert.True(t, locked)

	// Check that lock file exists
	lockFilePath := filePath + ".lock"
	_, err = os.Stat(lockFilePath)
	assert.NoError(t, err)
}

// TestLockFile_AlreadyLocked_NotStale tests that locking fails when a valid lock exists.
func TestLockFile_AlreadyLocked_NotStale(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "testfile.txt")
	// Create the file to lock
	err := os.WriteFile(filePath, []byte("test data"), 0644)
	assert.NoError(t, err)

	// Create a lock file with current modification time
	lockFilePath := filePath + ".lock"
	err = os.WriteFile(lockFilePath, []byte("lock"), 0600)
	assert.NoError(t, err)

	settings := setupSettings("")

	shipper := domain.NewMetricShipper(context.Background(), settings, nil)

	// Execute
	locked, err := shipper.LockFile(filePath)

	// Verify
	assert.NoError(t, err)
	assert.False(t, locked)
}

// TestLockFile_AlreadyLocked_Stale tests that a stale lock is removed and locking succeeds.
func TestLockFile_AlreadyLocked_Stale(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "testfile.txt")
	// Create the file to lock
	err := os.WriteFile(filePath, []byte("test data"), 0644)
	assert.NoError(t, err)

	// Create a stale lock file by setting its modification time to past
	lockFilePath := filePath + ".lock"
	err = os.WriteFile(lockFilePath, []byte("lock"), 0600)
	assert.NoError(t, err)

	// Set modification time to 20 minutes ago
	staleTime := time.Now().Add(-20 * time.Minute)
	err = os.Chtimes(lockFilePath, staleTime, staleTime)
	assert.NoError(t, err)

	settings := setupSettings("")

	shipper := domain.NewMetricShipper(context.Background(), settings, nil)

	// Execute
	locked, err := shipper.LockFile(filePath)

	// Verify
	assert.NoError(t, err)
	assert.True(t, locked)

	// Check that lock file exists
	_, err = os.Stat(lockFilePath)
	assert.NoError(t, err)
}

// TestLockFile_CreateError tests handling of errors when creating a lock file.
func TestLockFile_CreateError(t *testing.T) {
	// Setup
	// Attempt to lock a file in a non-existent directory
	invalidPath := "/nonexistent_dir/testfile.txt"

	settings := setupSettings("")

	shipper := domain.NewMetricShipper(context.Background(), settings, nil)

	// Execute
	locked, err := shipper.LockFile(invalidPath)

	// Verify
	assert.Error(t, err)
	assert.False(t, locked)
}

// TestUnlockFile_Success tests successful unlocking of a file.
func TestUnlockFile_Success(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "testfile.txt")
	// Create the file and lock file
	err := os.WriteFile(filePath, []byte("test data"), 0644)
	assert.NoError(t, err)

	lockFilePath := filePath + ".lock"
	err = os.WriteFile(lockFilePath, []byte("lock"), 0600)
	assert.NoError(t, err)

	settings := setupSettings("")

	shipper := domain.NewMetricShipper(context.Background(), settings, nil)

	// Ensure lock file exists
	_, err = os.Stat(lockFilePath)
	assert.NoError(t, err)

	// Execute
	shipper.UnlockFile(filePath)

	// Verify lock file is removed
	_, err = os.Stat(lockFilePath)
	assert.True(t, os.IsNotExist(err))
}

// TestUnlockFile_NotExist tests unlocking a file when no lock exists.
func TestUnlockFile_NotExist(t *testing.T) {
	// Setup
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "testfile.txt")

	settings := setupSettings("")

	shipper := domain.NewMetricShipper(context.Background(), settings, nil)

	// Ensure lock file does not exist
	lockFilePath := filePath + ".lock"
	_, err := os.Stat(lockFilePath)
	assert.True(t, os.IsNotExist(err))

	// Execute
	shipper.UnlockFile(filePath)

	// Verify no error and lock file still does not exist
	_, err = os.Stat(lockFilePath)
	assert.True(t, os.IsNotExist(err))
}
