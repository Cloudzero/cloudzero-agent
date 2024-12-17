// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package http

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/types"
)

// MockWriter is a mock implementation of the Writer interface.
type MockWriter struct {
	mock.Mock
}

func (m *MockWriter) WriteData(data types.ResourceTags, isCreate bool) error {
	args := m.Called(data, isCreate)
	return args.Error(0)
}

func (m *MockWriter) UpdateSentAtForRecords(records []types.ResourceTags, ct time.Time) (int64, error) {
	args := m.Called(records, ct)
	intVal, _ := args.Get(0).(int64)
	errVal := args.Error(1)
	return intVal, errVal
}

func (m *MockWriter) PurgeStaleData(rt time.Duration) error {
	return nil
}

// MockReader is a mock implementation of the Reader interface.
type MockReader struct {
	mock.Mock
}

func (m *MockReader) ReadData(ct time.Time) ([]types.ResourceTags, error) {
	args := m.Called(ct)
	mockRecords := args.Get(0)
	if mockRecords == nil {
		return nil, args.Error(1)
	}
	return mockRecords.([]types.ResourceTags), args.Error(1)
}

// MockClock is a mock implementation of the Clock interface.
type MockClock struct {
	mock.Mock
}

func (m *MockClock) GetCurrentTime() time.Time {
	args := m.Called()
	return args.Get(0).(time.Time)
}

// TestSetup encapsulates common setup for tests.
type TestSetup struct {
	apiKeyPath string
	server     *httptest.Server
	settings   *config.Settings
	mockClock  *MockClock
	mockWriter *MockWriter
	mockReader *MockReader
	rw         *RemoteWriter
}

// createAPIKeyFile creates a temporary API key file with the given content.
// It returns the file path and a cleanup function to remove the file.
func createAPIKeyFile(t *testing.T, apiKeyContent string) string {
	apiKeyFile, err := os.CreateTemp("", "api_key-*.txt")
	require.NoError(t, err, "Failed to create temp API key file")

	_, err = apiKeyFile.Write([]byte(apiKeyContent))
	require.NoError(t, err, "Failed to write to temp API key file")

	err = apiKeyFile.Close()
	require.NoError(t, err, "Failed to close temp API key file")

	t.Cleanup(func() {
		_ = os.Remove(apiKeyFile.Name())
	})

	return apiKeyFile.Name()
}

// setupTest initializes the test environment and returns a TestSetup instance.
func setupTest(t *testing.T, handlerFunc http.HandlerFunc, apiKeyContent string) *TestSetup {
	// Create temporary API key file
	apiKeyPath := createAPIKeyFile(t, apiKeyContent)

	// Start test HTTP server
	server := httptest.NewServer(http.HandlerFunc(handlerFunc))
	t.Cleanup(func() {
		server.Close()
	})

	// Initialize settings
	settings := &config.Settings{
		APIKeyPath: apiKeyPath,
		RemoteWrite: config.RemoteWrite{
			SendInterval: time.Minute,
			Host:         server.URL,
			SendTimeout:  5 * time.Second,
			MaxRetries:   3,
		},
	}
	settings.SetAPIKey()

	// Initialize mocks
	mockClock := new(MockClock)
	mockWriter := new(MockWriter)
	mockReader := new(MockReader)

	// Initialize RemoteWriter
	rw := &RemoteWriter{
		reader:   mockReader,
		writer:   mockWriter,
		settings: settings,
		clock:    mockClock,
	}

	return &TestSetup{
		apiKeyPath: apiKeyPath,
		server:     server,
		settings:   settings,
		mockClock:  mockClock,
		mockWriter: mockWriter,
		mockReader: mockReader,
		rw:         rw,
	}
}

// resetAllMetrics resets all Prometheus metrics used in the tests.
func resetAllMetrics() {
	remoteWriteTimeseriesSent.Reset()
	remoteWriteBacklog.Reset()
	remoteWriteFailures.Reset()
	remoteWriteRequestDuration.Reset()
	remoteWriteResponseCodes.Reset()
	remoteWritePayloadSizeBytes.Reset()
	remoteWriteRecordsProcessed.Reset()
	remoteWriteDBFailures.Reset()
}

func TestRemoteWriter_Flush(t *testing.T) {
	t.Run("successful flush with metrics check", func(t *testing.T) {
		resetAllMetrics()

		apiKeyContent := "test-api-key"
		handler := func(w http.ResponseWriter, r *http.Request) {
			// Basic request validation
			if r.Method != http.MethodPost {
				t.Errorf("Expected POST method, got: %s", r.Method)
			}
			if r.Header.Get("Authorization") != fmt.Sprintf("Bearer %s", apiKeyContent) {
				t.Errorf("Expected Authorization header to be 'Bearer %s', got: %s", apiKeyContent, r.Header.Get("Authorization"))
			}
			if r.Header.Get("Content-Encoding") != "snappy" {
				t.Errorf("Expected Content-Encoding 'snappy', got: %s", r.Header.Get("Content-Encoding"))
			}
			if r.Header.Get("Content-Type") != "application/x-protobuf" {
				t.Errorf("Expected Content-Type 'application/x-protobuf', got: %s", r.Header.Get("Content-Type"))
			}
			w.WriteHeader(http.StatusOK)
		}

		setup := setupTest(t, handler, apiKeyContent)

		// Prepare test data
		currentTime := time.Now().UTC()
		singleRecord := types.ResourceTags{
			Name:          "test-deployment",
			Type:          config.Deployment,
			Labels:        &config.MetricLabelTags{"label1": "value1"},
			Annotations:   nil,
			MetricLabels:  &config.MetricLabels{"metric1": "value1"},
			RecordCreated: currentTime,
			RecordUpdated: currentTime,
			SentAt:        nil,
			Size:          20,
		}
		records := []types.ResourceTags{singleRecord}

		expectedTime := currentTime.Add(time.Minute) // Arbitrary adjustment for example
		setup.mockClock.On("GetCurrentTime").Return(expectedTime).Once()
		setup.mockReader.On("ReadData", expectedTime).Return(records, nil).Once()
		setup.mockWriter.On("UpdateSentAtForRecords", records, expectedTime).Return(int64(1), nil).Once()
		setup.mockReader.On("ReadData", expectedTime).Return([]types.ResourceTags{}, nil).Once()

		// Execute Flush
		err := setup.rw.Flush()
		require.NoError(t, err, "Flush should not return an error")

		// Validate metrics
		endpoint := setup.server.URL

		wantTimeseries := float64(1)
		got := testutil.ToFloat64(remoteWriteTimeseriesSent.WithLabelValues(endpoint))
		require.Equal(t, wantTimeseries, got, "remoteWriteTimeseriesSent metric mismatch")

		got = testutil.ToFloat64(remoteWriteBacklog.WithLabelValues(endpoint))
		require.Equal(t, 0.0, got, "remoteWriteBacklog metric should be 0")

		got = testutil.ToFloat64(remoteWriteResponseCodes.WithLabelValues(endpoint, "200"))
		require.Equal(t, 1.0, got, "remoteWriteResponseCodes for 200 should be 1")

		got = testutil.ToFloat64(remoteWriteFailures.WithLabelValues(endpoint))
		require.Equal(t, 0.0, got, "remoteWriteFailures metric should be 0")

		require.True(t, testutil.CollectAndCount(remoteWriteRequestDuration) > 0, "remoteWriteRequestDuration should have observations")
		require.True(t, testutil.CollectAndCount(remoteWritePayloadSizeBytes) > 0, "remoteWritePayloadSizeBytes should have observations")

		// Assert mock expectations
		setup.mockReader.AssertExpectations(t)
		setup.mockWriter.AssertExpectations(t)
	})

	t.Run("no records to process", func(t *testing.T) {
		resetAllMetrics()

		apiKeyContent := "test-api-key-no-records"
		handler := func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("Expected no request, but received one with method: %s", r.Method)
		}

		setup := setupTest(t, handler, apiKeyContent)

		// Setup mocks for no records
		currentTime := time.Now().UTC()
		setup.mockClock.On("GetCurrentTime").Return(currentTime).Once()
		setup.mockReader.On("ReadData", currentTime).Return([]types.ResourceTags{}, nil).Once()

		// Execute Flush
		err := setup.rw.Flush()
		require.NoError(t, err, "Flush should not return an error when there are no records")

		// Validate metrics
		endpoint := setup.server.URL

		got := testutil.ToFloat64(remoteWriteBacklog.WithLabelValues(endpoint))
		require.Equal(t, 0.0, got, "remoteWriteBacklog metric should be 0")

		got = testutil.ToFloat64(remoteWriteTimeseriesSent.WithLabelValues(endpoint))
		require.Equal(t, 0.0, got, "remoteWriteTimeseriesSent metric should be 0")

		// Assert mock expectations
		setup.mockReader.AssertExpectations(t)
		setup.mockWriter.AssertNotCalled(t, "UpdateSentAtForRecords", mock.Anything, mock.Anything)
	})
}

func TestRemoteWriter_Flush_DBUpdateSuccess(t *testing.T) {
	resetAllMetrics()

	apiKeyContent := "test-api-key-db-success"
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	setup := setupTest(t, handler, apiKeyContent)

	// Prepare test data
	currentTime := time.Now().UTC()
	singleRecord := types.ResourceTags{
		Name:          "test-deployment-db-success",
		Type:          config.Deployment,
		Labels:        &config.MetricLabelTags{"label1": "value1"},
		Annotations:   nil,
		MetricLabels:  &config.MetricLabels{"metric1": "value1"},
		RecordCreated: currentTime,
		RecordUpdated: currentTime,
		SentAt:        nil,
		Size:          30,
	}
	records := []types.ResourceTags{singleRecord}

	expectedTime := currentTime.Add(2 * time.Minute) // Arbitrary adjustment for example
	setup.mockClock.On("GetCurrentTime").Return(expectedTime).Once()
	setup.mockReader.On("ReadData", expectedTime).Return(records, nil).Once()
	setup.mockWriter.On("UpdateSentAtForRecords", records, expectedTime).Return(int64(len(records)), nil).Once()
	setup.mockReader.On("ReadData", expectedTime).Return([]types.ResourceTags{}, nil).Once()

	// Execute Flush
	err := setup.rw.Flush()
	require.NoError(t, err, "Flush should not return an error on successful DB update")

	// Validate metrics
	endpoint := setup.server.URL

	got := testutil.ToFloat64(remoteWriteTimeseriesSent.WithLabelValues(endpoint))
	require.Equal(t, 1.0, got, "remoteWriteTimeseriesSent metric should be 1")

	got = testutil.ToFloat64(remoteWriteRecordsProcessed.WithLabelValues(endpoint))
	require.Equal(t, 1.0, got, "remoteWriteRecordsProcessed metric should be 1")

	got = testutil.ToFloat64(remoteWriteDBFailures.WithLabelValues(endpoint))
	require.Equal(t, 0.0, got, "remoteWriteDBFailures metric should be 0")

	// Assert mock expectations
	setup.mockReader.AssertExpectations(t)
	setup.mockWriter.AssertExpectations(t)
}

func TestRemoteWriter_Flush_DBUpdateFailure(t *testing.T) {
	resetAllMetrics()

	apiKeyContent := "test-api-key-db-failure"
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}

	setup := setupTest(t, handler, apiKeyContent)

	// Prepare test data
	currentTime := time.Now().UTC()
	singleRecord := types.ResourceTags{
		Name:          "test-deployment-db-failure",
		Type:          config.Deployment,
		Labels:        &config.MetricLabelTags{"label1": "value1"},
		Annotations:   nil,
		MetricLabels:  &config.MetricLabels{"metric1": "value1"},
		RecordCreated: currentTime,
		RecordUpdated: currentTime,
		SentAt:        nil,
		Size:          25,
	}
	records := []types.ResourceTags{singleRecord}

	expectedTime := currentTime.Add(3 * time.Minute) // Arbitrary adjustment for example
	setup.mockClock.On("GetCurrentTime").Return(expectedTime).Once()
	setup.mockReader.On("ReadData", expectedTime).Return(records, nil).Once()
	setup.mockWriter.On("UpdateSentAtForRecords", records, expectedTime).Return(int64(0), fmt.Errorf("database error")).Once()

	// Execute Flush
	err := setup.rw.Flush()
	require.Error(t, err, "Flush should return an error when DB update fails")

	// Validate metrics
	endpoint := setup.server.URL

	got := testutil.ToFloat64(remoteWriteTimeseriesSent.WithLabelValues(endpoint))
	require.Equal(t, 1.0, got, "remoteWriteTimeseriesSent metric should be 1")

	got = testutil.ToFloat64(remoteWriteRecordsProcessed.WithLabelValues(endpoint))
	require.Equal(t, 0.0, got, "remoteWriteRecordsProcessed metric should be 0 due to DB failure")

	got = testutil.ToFloat64(remoteWriteDBFailures.WithLabelValues(endpoint))
	require.Equal(t, 1.0, got, "remoteWriteDBFailures metric should be 1")

	// Assert mock expectations
	setup.mockReader.AssertExpectations(t)
	setup.mockWriter.AssertExpectations(t)
}
