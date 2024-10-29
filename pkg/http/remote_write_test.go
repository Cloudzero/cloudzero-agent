package http

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"net/http/httptest"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/storage"
	"github.com/stretchr/testify/mock"
)

// create mock writer
type MockWriter struct {
	mock.Mock
}

func (m *MockWriter) WriteData(data storage.ResourceTags) error {
	args := m.Called(data)
	return args.Error(1)
}

func (m *MockWriter) UpdateSentAtForRecords(records []storage.ResourceTags, ct time.Time) (int64, error) {
	args := m.Called(records, ct)
	return 0, args.Error(0)
}

func (m *MockWriter) PurgeStaleData(rt time.Duration) error {
	return nil
}

// create mock reader
type MockReader struct {
	mock.Mock
}

// create mock clock
type MockClock struct {
	mock.Mock
}

func (m *MockClock) GetCurrentTime() time.Time {
	args := m.Called()
	return args.Get(0).(time.Time)
}

func (m *MockReader) ReadData(ct time.Time) ([]storage.ResourceTags, error) {
	args := m.Called(ct)
	mockRecords := args.Get(0)
	return mockRecords.([]storage.ResourceTags), args.Error(1)
}
func TestRemoteWriter_Flush(t *testing.T) {

	testApiKey := "test-api-key"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			t.Errorf("Expected Accept: application/json header, got: %s", r.Header.Get("Accept"))
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
		},
	}

	t.Run("successful flush", func(t *testing.T) {
		mockClock := new(MockClock)
		mockWriter := new(MockWriter)
		mockReader := new(MockReader)
		rw := &RemoteWriter{reader: mockReader, writer: mockWriter, settings: settings, clock: mockClock}
		currentTime := time.Now().UTC()
		testNamespace := "test-namespace"
		singleRecord := storage.ResourceTags{
			Name:         "test-deployment",
			Type:         config.Deployment,
			Namespace:    &testNamespace,
			Labels:       &config.MetricLabelTags{"label1": "value1"},
			Annotations:  nil,
			MetricLabels: &config.MetricLabels{"metric1": "value1"},
			CreatedAt:    currentTime,
			UpdatedAt:    currentTime,
			SentAt:       nil,
			Size:         20,
		}
		records := []storage.ResourceTags{singleRecord}

		expectedTime := time.Now().UTC()
		mockClock.On("GetCurrentTime").Return(expectedTime)
		mockReader.On("ReadData", expectedTime).Return(records, nil).Once()
		mockReader.On("ReadData", expectedTime).Return([]storage.ResourceTags{}, nil)
		mockWriter.On("UpdateSentAtForRecords", records, expectedTime).Return(nil)

		rw.Flush()
		mockReader.AssertExpectations(t)
	})

	t.Run("no records to process", func(t *testing.T) {
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

		rw.Flush()
		mockReader.AssertExpectations(t)
		mockWriter.AssertNotCalled(t, "UpdateSentAtForRecords", mock.Anything)
	})
}
