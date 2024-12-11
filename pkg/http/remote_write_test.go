package http

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/storage"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/mock"
)

type MockWriter struct {
	mock.Mock
}

func (m *MockWriter) WriteData(data storage.ResourceTags, isCreate bool) error {
	args := m.Called(data)
	return args.Error(1)
}

func (m *MockWriter) UpdateSentAtForRecords(records []storage.ResourceTags, ct time.Time) (int64, error) {
	args := m.Called(records, ct)
	intVal, _ := args.Get(0).(int64)
	errVal := args.Error(1)
	return intVal, errVal
}

func (m *MockWriter) PurgeStaleData(rt time.Duration) error {
	return nil
}

type MockReader struct {
	mock.Mock
}

func (m *MockReader) ReadData(ct time.Time) ([]storage.ResourceTags, error) {
	args := m.Called(ct)
	mockRecords := args.Get(0)
	return mockRecords.([]storage.ResourceTags), args.Error(1)
}

type MockClock struct {
	mock.Mock
}

func (m *MockClock) GetCurrentTime() time.Time {
	args := m.Called()
	return args.Get(0).(time.Time)
}

func TestRemoteWriter_Flush(t *testing.T) {
	testApiKey := "test-api-key"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Basic request validation
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got: %s", r.Method)
		}
		if r.Header.Get("Authorization") != fmt.Sprintf("Bearer %s", testApiKey) {
			t.Errorf("Expected test-api-key for Authorization, got: %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Encoding") != "snappy" {
			t.Errorf("Expected snappy for Content-Encoding, got: %s", r.Header.Get("Content-Encoding"))
		}
		if r.Header.Get("Content-Type") != "application/x-protobuf" {
			t.Errorf("Expected Content-Type: application/x-protobuf, got: %s", r.Header.Get("Content-Type"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	settings := &config.Settings{
		RemoteWrite: config.RemoteWrite{
			SendInterval: time.Minute,
			Host:         server.URL,
			APIKey:       testApiKey,
			SendTimeout:  5 * time.Second,
			MaxRetries:   3,
		},
	}

	t.Run("successful flush with metrics check", func(t *testing.T) {
		// Reset metrics to a known state before test
		remoteWriteTimeseriesSent.Reset()
		remoteWriteBacklog.Reset()
		remoteWriteFailures.Reset()
		remoteWriteRequestDuration.Reset()
		remoteWriteResponseCodes.Reset()
		remoteWritePayloadSizeBytes.Reset()

		mockClock := new(MockClock)
		mockWriter := new(MockWriter)
		mockReader := new(MockReader)
		rw := &RemoteWriter{reader: mockReader, writer: mockWriter, settings: settings, clock: mockClock}

		currentTime := time.Now().UTC()
		singleRecord := storage.ResourceTags{
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
		records := []storage.ResourceTags{singleRecord}

		expectedTime := time.Now().UTC()
		mockClock.On("GetCurrentTime").Return(expectedTime).Once()
		mockReader.On("ReadData", expectedTime).Return(records, nil).Once()
		mockWriter.On("UpdateSentAtForRecords", records, expectedTime).Return(int64(1), nil).Once()

		// After sending the first batch, we expect the next ReadData call to return no records
		mockReader.On("ReadData", expectedTime).Return([]storage.ResourceTags{}, nil).Once()

		err := rw.Flush()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Check metrics
		endpoint := server.URL

		// We sent 1 timeseries (two sets if annotations were included, but here just one timeseries),
		// The formatMetrics method returns at least one timeseries per record, possibly more.
		// In this example:
		// Each record with labels creates one timeseries. (No annotations in this record.)
		wantTimeseries := float64(1)

		// remoteWriteTimeseriesSent should have incremented by wantTimeseries
		if got := testutil.ToFloat64(remoteWriteTimeseriesSent.WithLabelValues(endpoint)); got != wantTimeseries {
			t.Errorf("remoteWriteTimeseriesSent: got %f, want %f", got, wantTimeseries)
		}

		// Expect backlog of 0 by the end of flush
		if got := testutil.ToFloat64(remoteWriteBacklog.WithLabelValues(endpoint)); got != 0 {
			t.Errorf("remoteWriteBacklog: got %f, want 0", got)
		}

		// Since we got a 200 OK, remoteWriteResponseCodes with "200" label should have incremented
		if got := testutil.ToFloat64(remoteWriteResponseCodes.WithLabelValues(endpoint, "200")); got != 1 {
			t.Errorf("remoteWriteResponseCodes 200: got %f, want 1", got)
		}

		// No failures expected in this scenario
		if got := testutil.ToFloat64(remoteWriteFailures.WithLabelValues(endpoint)); got != 0 {
			t.Errorf("remoteWriteFailures: got %f, want 0", got)
		}

		// Request durations should have at least one observation
		// Since it's a histogram, we just check that there's something registered
		if count := testutil.CollectAndCount(remoteWriteRequestDuration); count == 0 {
			t.Error("expected remoteWriteRequestDuration to have observations, got none")
		}

		// Payload size should also have at least one observation
		if count := testutil.CollectAndCount(remoteWritePayloadSizeBytes); count == 0 {
			t.Error("expected remoteWritePayloadSizeBytes to have observations, got none")
		}

		mockReader.AssertExpectations(t)
		mockWriter.AssertExpectations(t)
	})

	t.Run("no records to process", func(t *testing.T) {
		// Reset metrics again for a clean slate
		remoteWriteTimeseriesSent.Reset()
		remoteWriteBacklog.Reset()
		remoteWriteFailures.Reset()
		remoteWriteRequestDuration.Reset()
		remoteWriteResponseCodes.Reset()
		remoteWritePayloadSizeBytes.Reset()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("Expected no request, got: %s", r.Method)
		}))
		defer server.Close()

		settings := &config.Settings{
			RemoteWrite: config.RemoteWrite{
				SendInterval: time.Minute,
				Host:         server.URL,
				APIKey:       testApiKey,
				SendTimeout:  5 * time.Second,
			},
		}

		mockClock := new(MockClock)
		mockWriter := new(MockWriter)
		mockReader := new(MockReader)
		rw := &RemoteWriter{reader: mockReader, writer: mockWriter, settings: settings, clock: mockClock}

		records := []storage.ResourceTags{}
		expectedTime := time.Now().UTC()
		mockClock.On("GetCurrentTime").Return(expectedTime)
		mockReader.On("ReadData", expectedTime).Return(records, nil)

		err := rw.Flush()
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// We had no records, so backlog should have been set to 0
		endpoint := server.URL
		if got := testutil.ToFloat64(remoteWriteBacklog.WithLabelValues(endpoint)); got != 0 {
			t.Errorf("remoteWriteBacklog: got %f, want 0", got)
		}

		// Since we never sent any timeseries, remoteWriteTimeseriesSent should remain 0
		if got := testutil.ToFloat64(remoteWriteTimeseriesSent.WithLabelValues(endpoint)); got != 0 {
			t.Errorf("remoteWriteTimeseriesSent: got %f, want 0", got)
		}

		mockReader.AssertExpectations(t)
		mockWriter.AssertNotCalled(t, "UpdateSentAtForRecords", mock.Anything)
	})
}

func TestRemoteWriter_Flush_DBUpdateSuccess(t *testing.T) {
	// Reset all relevant metrics to a known state
	remoteWriteTimeseriesSent.Reset()
	remoteWriteRecordsProcessed.Reset()
	remoteWriteDBFailures.Reset()

	testApiKey := "test-api-key"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	settings := &config.Settings{
		RemoteWrite: config.RemoteWrite{
			SendInterval: time.Minute,
			Host:         server.URL,
			APIKey:       testApiKey,
			SendTimeout:  5 * time.Second,
			MaxRetries:   3,
		},
	}

	mockClock := new(MockClock)
	mockWriter := new(MockWriter)
	mockReader := new(MockReader)
	rw := &RemoteWriter{reader: mockReader, writer: mockWriter, settings: settings, clock: mockClock}

	currentTime := time.Now().UTC()
	singleRecord := storage.ResourceTags{
		Name:          "test-deployment",
		Type:          config.Deployment,
		Labels:        &config.MetricLabelTags{"label1": "value1"},
		Annotations:   nil,
		MetricLabels:  &config.MetricLabels{"metric1": "value1"},
		RecordCreated: currentTime,
		RecordUpdated: currentTime,
		SentAt:        nil,
	}
	records := []storage.ResourceTags{singleRecord}

	expectedTime := time.Now().UTC()
	endpoint := server.URL

	// Set expectations
	mockClock.On("GetCurrentTime").Return(expectedTime).Once()
	mockReader.On("ReadData", expectedTime).Return(records, nil).Once()
	// On the second iteration, no more records
	mockReader.On("ReadData", expectedTime).Return([]storage.ResourceTags{}, nil).Once()
	// Simulate successful update
	mockWriter.On("UpdateSentAtForRecords", records, expectedTime).Return(int64(len(records)), nil).Once()

	// Run Flush
	err := rw.Flush()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Validate metrics
	// Timeseries sent should be incremented by 1 (we had 1 record)
	if got := testutil.ToFloat64(remoteWriteTimeseriesSent.WithLabelValues(endpoint)); got != 1.0 {
		t.Errorf("remoteWriteTimeseriesSent: got %f, want 1.0", got)
	}

	// Records processed should be incremented by the number of records (1)
	if got := testutil.ToFloat64(remoteWriteRecordsProcessed.WithLabelValues(endpoint)); got != 1.0 {
		t.Errorf("remoteWriteRecordsProcessed: got %f, want 1.0", got)
	}

	// No DB failures should be recorded
	if got := testutil.ToFloat64(remoteWriteDBFailures.WithLabelValues(endpoint)); got != 0.0 {
		t.Errorf("remoteWriteDBFailures: got %f, want 0.0", got)
	}
}

func TestRemoteWriter_Flush_DBUpdateFailure(t *testing.T) {
	// Reset metrics again
	remoteWriteTimeseriesSent.Reset()
	remoteWriteRecordsProcessed.Reset()
	remoteWriteDBFailures.Reset()

	testApiKey := "test-api-key"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	settings := &config.Settings{
		RemoteWrite: config.RemoteWrite{
			SendInterval: time.Minute,
			Host:         server.URL,
			APIKey:       testApiKey,
			SendTimeout:  5 * time.Second,
			MaxRetries:   3,
		},
	}

	mockClock := new(MockClock)
	mockWriter := new(MockWriter)
	mockReader := new(MockReader)
	rw := &RemoteWriter{reader: mockReader, writer: mockWriter, settings: settings, clock: mockClock}

	currentTime := time.Now().UTC()
	singleRecord := storage.ResourceTags{
		Name:          "test-deployment",
		Type:          config.Deployment,
		Labels:        &config.MetricLabelTags{"label1": "value1"},
		Annotations:   nil,
		MetricLabels:  &config.MetricLabels{"metric1": "value1"},
		RecordCreated: currentTime,
		RecordUpdated: currentTime,
		SentAt:        nil,
	}
	records := []storage.ResourceTags{singleRecord}

	expectedTime := time.Now().UTC()
	endpoint := server.URL

	mockClock.On("GetCurrentTime").Return(expectedTime).Once()
	mockReader.On("ReadData", expectedTime).Return(records, nil).Once()
	// Expect that we won't get to the second read due to error
	mockWriter.On("UpdateSentAtForRecords", records, expectedTime).Return(int64(0), fmt.Errorf("database error")).Once()

	// Call Flush - it will fail updating the database
	err := rw.Flush()
	if err == nil {
		t.Fatalf("expected an error, got nil")
	}

	// Timeseries sent should have been incremented by 1 because the push succeeded
	// before we tried to update the database.
	if got := testutil.ToFloat64(remoteWriteTimeseriesSent.WithLabelValues(endpoint)); got != 1.0 {
		t.Errorf("remoteWriteTimeseriesSent: got %f, want 1.0", got)
	}

	// Records processed should NOT increment because the database update failed
	if got := testutil.ToFloat64(remoteWriteRecordsProcessed.WithLabelValues(endpoint)); got != 0.0 {
		t.Errorf("remoteWriteRecordsProcessed: got %f, want 0.0", got)
	}

	// DB failures should increment by 1
	if got := testutil.ToFloat64(remoteWriteDBFailures.WithLabelValues(endpoint)); got != 1.0 {
		t.Errorf("remoteWriteDBFailures: got %f, want 1.0", got)
	}
}
