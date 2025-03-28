// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package pusher_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/domain/pusher"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/types"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/types/mocks"
)

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
func setupTest(t *testing.T, clock types.TimeProvider, store types.ResourceStore, handlerFunc http.HandlerFunc, apiKeyContent string) (*pusher.MetricsPusher, string) {
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
			Host:            server.URL,
			MaxBytesPerSend: 40, // small - should be no more than 2 objects at a time (see mkRecords)
			SendInterval:    time.Second,
			SendTimeout:     5 * time.Millisecond,
			MaxRetries:      3,
		},
	}
	settings.SetAPIKey()

	p := pusher.New(context.Background(), store, clock, settings)
	rw := p.(*pusher.MetricsPusher)
	rw.ResetStats()
	return rw, server.URL
}

func mkRecords(tm time.Time, count int) []*types.ResourceTags {
	records := make([]*types.ResourceTags, count)
	for i := 0; i < count; i++ {
		records[i] = &types.ResourceTags{
			Type:          config.Deployment,
			Name:          fmt.Sprintf("test-deployment-%d", i),
			Labels:        &config.MetricLabelTags{"label": fmt.Sprintf("label-%d", i)},
			Annotations:   &config.MetricLabelTags{"label": fmt.Sprintf("annotation-%d", i)},
			MetricLabels:  &config.MetricLabels{"metric1": fmt.Sprintf("metric-label-%d", i)},
			RecordCreated: tm,
			RecordUpdated: tm,
			SentAt:        nil,
			Size:          20,
		}
	}
	return records
}

func Test_SupportsRunnableInterface(t *testing.T) {
	currentTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	mockClock := mocks.NewMockClock(currentTime)

	// Initialize the mock store
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mocks.NewMockResourceStore(ctrl)

	p, _ := setupTest(t, mockClock, mockStore, func(w http.ResponseWriter, r *http.Request) {}, "")
	assert.False(t, p.IsRunning())

	mockStore.EXPECT().FindAllBy(gomock.Any(), gomock.Any()).Return([]*types.ResourceTags{}, nil).AnyTimes()

	err := p.Run()
	assert.NoError(t, err)
	assert.True(t, p.IsRunning())

	err = p.Shutdown()
	assert.NoError(t, err)
	assert.False(t, p.IsRunning())
}

func Test_FlushMany(t *testing.T) {
	currentTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	mockClock := mocks.NewMockClock(currentTime)

	// Initialize the mock store
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mocks.NewMockResourceStore(ctrl)

	records := mkRecords(currentTime, 5)
	mockStore.EXPECT().Tx(gomock.Any(), gomock.Any()).Return(nil).Times(1)
	mockStore.EXPECT().FindAllBy(gomock.Any(), gomock.Any()).Return(records, nil).AnyTimes()

	// Fake the time it was sent
	sentAt := currentTime.Add(2 * time.Minute)
	mockClock.SetCurrentTime(sentAt)

	// Make sure the updates have the expected sentAt time
	mockStore.EXPECT().Update(gomock.Any(), gomock.Cond(func(m *types.ResourceTags) bool {
		return *m.SentAt == sentAt
	})).Return(nil).AnyTimes()

	// Capture the records that were sent
	apiKeyContent := "apiKeyContent"
	expectedSentCount := 3
	actualSentCount := 0
	p, host := setupTest(t, mockClock, mockStore,
		func(w http.ResponseWriter, r *http.Request) {
			actualSentCount++
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
		},
		apiKeyContent,
	)

	err := p.Flush()
	require.NoError(t, err)

	assert.Equal(t, expectedSentCount, actualSentCount)

	// now validate the metrics
	got := testutil.ToFloat64(pusher.RemoteWriteRecordsProcessed.WithLabelValues(host))
	require.Equal(t, 5.0, got, "remoteWriteRecordsProcessed metric should be 5")

	// XXX: DAN: the metrics count grow due to seperation / parsing logic
	got = testutil.ToFloat64(pusher.RemoteWriteTimeseriesSent.WithLabelValues(host))
	require.Equal(t, 10.0, got, "remoteWriteTimeseriesSent metric should be 10")

	got = testutil.ToFloat64(pusher.RemoteWriteDBFailures.WithLabelValues(host))
	require.Equal(t, 0.0, got, "remoteWriteDBFailures metric should be 0")
}

func Test_Flush_FindAll_ReturnsNothing(t *testing.T) {
	currentTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	mockClock := mocks.NewMockClock(currentTime)

	// Initialize the mock store
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mocks.NewMockResourceStore(ctrl)

	emptyList := mkRecords(currentTime, 0)
	mockStore.EXPECT().FindAllBy(gomock.Any(), gomock.Any()).Return(emptyList, nil).Times(1)

	p, host := setupTest(t, mockClock, mockStore, func(w http.ResponseWriter, r *http.Request) {}, "apiKeyContent")

	err := p.Flush()
	assert.NoError(t, err)

	// now validate the metrics
	got := testutil.ToFloat64(pusher.RemoteWriteRecordsProcessed.WithLabelValues(host))
	require.Equal(t, 0.0, got, "remoteWriteRecordsProcessed metric should be 0")

	// XXX: DAN: the metrics count grow due to seperation / parsing logic
	got = testutil.ToFloat64(pusher.RemoteWriteTimeseriesSent.WithLabelValues(host))
	require.Equal(t, 0.0, got, "remoteWriteTimeseriesSent metric should be 0")

	got = testutil.ToFloat64(pusher.RemoteWriteDBFailures.WithLabelValues(host))
	require.Equal(t, 0.0, got, "remoteWriteDBFailures metric should be 0")
}

func Test_Flush_Handles_FindAll_Error_Gracefully(t *testing.T) {
	currentTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	mockClock := mocks.NewMockClock(currentTime)

	// Initialize the mock store
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mocks.NewMockResourceStore(ctrl)

	expectedError := errors.New("find error")
	mockStore.EXPECT().FindAllBy(gomock.Any(), gomock.Any()).Return(nil, expectedError).Times(1)

	p, host := setupTest(t, mockClock, mockStore, func(w http.ResponseWriter, r *http.Request) {}, "apiKeyContent")

	err := p.Flush()
	assert.Error(t, err)
	assert.Equal(t, "failed to find records to send: find error", err.Error())

	// now validate the metrics
	got := testutil.ToFloat64(pusher.RemoteWriteRecordsProcessed.WithLabelValues(host))
	require.Equal(t, 0.0, got, "remoteWriteRecordsProcessed metric should be 0")

	got = testutil.ToFloat64(pusher.RemoteWriteTimeseriesSent.WithLabelValues(host))
	require.Equal(t, 0.0, got, "remoteWriteTimeseriesSent metric should be 0")

	got = testutil.ToFloat64(pusher.RemoteWriteDBFailures.WithLabelValues(host))
	require.Equal(t, 1.0, got, "remoteWriteDBFailures metric should be 1")
}

func Test_Flush_Handles_Tx_Error_Gracefully(t *testing.T) {
	currentTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	mockClock := mocks.NewMockClock(currentTime)

	// Initialize the mock store
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mocks.NewMockResourceStore(ctrl)

	list := mkRecords(currentTime, 1)
	mockStore.EXPECT().FindAllBy(gomock.Any(), gomock.Any()).Return(list, nil).Times(1)

	// Setup the transaction error
	expectedError := errors.New("find error")
	mockStore.EXPECT().Tx(gomock.Any(), gomock.Any()).Return(expectedError).Times(1)

	p, host := setupTest(t, mockClock, mockStore, func(w http.ResponseWriter, r *http.Request) {}, "apiKeyContent")

	err := p.Flush()
	assert.Error(t, err)
	assert.Equal(t, "failed to update sent_at for records: find error", err.Error())

	// now validate the metrics
	got := testutil.ToFloat64(pusher.RemoteWriteRecordsProcessed.WithLabelValues(host))
	require.Equal(t, 1.0, got, "remoteWriteRecordsProcessed metric should be 1")

	got = testutil.ToFloat64(pusher.RemoteWriteTimeseriesSent.WithLabelValues(host))
	require.Equal(t, 2.0, got, "remoteWriteTimeseriesSent metric should be 2")

	got = testutil.ToFloat64(pusher.RemoteWriteDBFailures.WithLabelValues(host))
	require.Equal(t, 1.0, got, "remoteWriteDBFailures metric should be 1")
}

func Test_Flush_Handles_Update_Error_Gracefully(t *testing.T) {
	currentTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	mockClock := mocks.NewMockClock(currentTime)

	// Initialize the mock store
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mocks.NewMockResourceStore(ctrl)

	list := mkRecords(currentTime, 1)
	mockStore.EXPECT().FindAllBy(gomock.Any(), gomock.Any()).Return(list, nil).Times(1)
	mockStore.EXPECT().Tx(gomock.Any(), gomock.Any()).Return(nil).Times(1)

	// Setup the transaction error
	expectedError := errors.New("find error")
	mockStore.EXPECT().Update(gomock.Any(), gomock.Any()).Return(expectedError).Times(1)

	p, host := setupTest(t, mockClock, mockStore, func(w http.ResponseWriter, r *http.Request) {}, "apiKeyContent")

	err := p.Flush()
	assert.Error(t, err)
	assert.Equal(t, "failed to update sent_at for records: failed to update sent_at for record: find error", err.Error())

	// now validate the metrics
	got := testutil.ToFloat64(pusher.RemoteWriteRecordsProcessed.WithLabelValues(host))
	require.Equal(t, 1.0, got, "remoteWriteRecordsProcessed metric should be 1")

	got = testutil.ToFloat64(pusher.RemoteWriteTimeseriesSent.WithLabelValues(host))
	require.Equal(t, 2.0, got, "remoteWriteTimeseriesSent metric should be 2")

	got = testutil.ToFloat64(pusher.RemoteWriteDBFailures.WithLabelValues(host))
	require.Equal(t, 2.0, got, "remoteWriteDBFailures metric should be 1")
}

func Test_Flush_Handles_SendFailure(t *testing.T) {
	currentTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	mockClock := mocks.NewMockClock(currentTime)

	// Initialize the mock store
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mocks.NewMockResourceStore(ctrl)

	records := mkRecords(currentTime, 1)
	mockStore.EXPECT().FindAllBy(gomock.Any(), gomock.Any()).Return(records, nil).AnyTimes()

	// Capture the records that were sent
	apiKeyContent := "apiKeyContent"
	p, host := setupTest(t, mockClock, mockStore,
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		},
		apiKeyContent,
	)

	err := p.Flush()
	require.Error(t, err)

	// now validate the metrics
	got := testutil.ToFloat64(pusher.RemoteWriteRecordsProcessed.WithLabelValues(host))
	require.Equal(t, 0.0, got, "remoteWriteRecordsProcessed metric should be 0")

	// XXX: DAN: the metrics count grow due to seperation / parsing logic
	got = testutil.ToFloat64(pusher.RemoteWriteTimeseriesSent.WithLabelValues(host))
	require.Equal(t, 0.0, got, "remoteWriteTimeseriesSent metric should be 0")

	got = testutil.ToFloat64(pusher.RemoteWriteDBFailures.WithLabelValues(host))
	require.Equal(t, 0.0, got, "remoteWriteDBFailures metric should be 0")

	got = testutil.ToFloat64(pusher.RemoteWriteFailures.WithLabelValues(host))
	require.Equal(t, 1.0, got, "RemoteWriteFailures metric should be 1")
}
