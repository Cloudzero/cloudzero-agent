// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package metricio

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/prometheus/prometheus/prompb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRemoteWriter(t *testing.T) {
	writer := NewRemoteWriter("http://localhost:9009/api/v1/push")
	assert.NotNil(t, writer)
	assert.Equal(t, "http://localhost:9009/api/v1/push", writer.url)
}

func TestRemoteWriter_Write_Success(t *testing.T) {
	// Create a test server
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)

		// Verify request headers
		assert.Equal(t, "application/x-protobuf", r.Header.Get("Content-Type"))
		assert.Equal(t, "snappy", r.Header.Get("Content-Encoding"))
		assert.Equal(t, "0.1.0", r.Header.Get("X-Prometheus-Remote-Write-Version"))

		// Read body
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.NotEmpty(t, body)

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	writer := NewRemoteWriter(server.URL)

	timeSeries := []prompb.TimeSeries{
		{
			Labels: []prompb.Label{
				{Name: "__name__", Value: "test_metric"},
			},
			Samples: []prompb.Sample{
				{Value: 42.0, Timestamp: 1704067200000},
			},
		},
	}

	err := writer.Write(context.Background(), timeSeries)
	require.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&requestCount))
}

func TestRemoteWriter_Write_Empty(t *testing.T) {
	writer := NewRemoteWriter("http://localhost:9009/api/v1/push")

	// Writing empty slice should return immediately without error
	err := writer.Write(context.Background(), []prompb.TimeSeries{})
	require.NoError(t, err)
}

func TestRemoteWriter_Write_ServerError_Retry(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&requestCount, 1)
		if count < 3 {
			// First two requests fail
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("server error"))
			return
		}
		// Third request succeeds
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	writer := NewRemoteWriter(server.URL)

	timeSeries := []prompb.TimeSeries{
		{
			Labels:  []prompb.Label{{Name: "__name__", Value: "test"}},
			Samples: []prompb.Sample{{Value: 1.0, Timestamp: 1000}},
		},
	}

	err := writer.Write(context.Background(), timeSeries)
	require.NoError(t, err)
	assert.Equal(t, int32(3), atomic.LoadInt32(&requestCount))
}

func TestRemoteWriter_Write_PermanentFailure(t *testing.T) {
	var requestCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("permanent error"))
	}))
	defer server.Close()

	writer := NewRemoteWriter(server.URL)

	timeSeries := []prompb.TimeSeries{
		{
			Labels:  []prompb.Label{{Name: "__name__", Value: "test"}},
			Samples: []prompb.Sample{{Value: 1.0, Timestamp: 1000}},
		},
	}

	err := writer.Write(context.Background(), timeSeries)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed after")
	assert.Equal(t, int32(3), atomic.LoadInt32(&requestCount)) // DefaultMaxRetries
}

func TestRemoteWriter_Write_MultipleSeries(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	writer := NewRemoteWriter(server.URL)

	timeSeries := []prompb.TimeSeries{
		{
			Labels:  []prompb.Label{{Name: "__name__", Value: "metric1"}},
			Samples: []prompb.Sample{{Value: 1.0, Timestamp: 1000}},
		},
		{
			Labels:  []prompb.Label{{Name: "__name__", Value: "metric2"}},
			Samples: []prompb.Sample{{Value: 2.0, Timestamp: 2000}},
		},
		{
			Labels:  []prompb.Label{{Name: "__name__", Value: "metric3"}},
			Samples: []prompb.Sample{{Value: 3.0, Timestamp: 3000}},
		},
	}

	err := writer.Write(context.Background(), timeSeries)
	require.NoError(t, err)
}

func TestRemoteWriter_Write_Various_StatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{"200 OK", http.StatusOK, false},
		{"204 No Content", http.StatusNoContent, false},
		{"400 Bad Request", http.StatusBadRequest, true},
		{"401 Unauthorized", http.StatusUnauthorized, true},
		{"500 Internal Server Error", http.StatusInternalServerError, true},
		{"503 Service Unavailable", http.StatusServiceUnavailable, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			writer := NewRemoteWriter(server.URL)
			// Override retries for faster tests
			writer.maxRetries = 1

			timeSeries := []prompb.TimeSeries{
				{
					Labels:  []prompb.Label{{Name: "__name__", Value: "test"}},
					Samples: []prompb.Sample{{Value: 1.0, Timestamp: 1000}},
				},
			}

			err := writer.Write(context.Background(), timeSeries)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRemoteWriter_Write_NetworkError(t *testing.T) {
	// Use an address that will fail to connect
	writer := NewRemoteWriter("http://localhost:1") // Invalid port
	writer.maxRetries = 1

	timeSeries := []prompb.TimeSeries{
		{
			Labels:  []prompb.Label{{Name: "__name__", Value: "test"}},
			Samples: []prompb.Sample{{Value: 1.0, Timestamp: 1000}},
		},
	}

	err := writer.Write(context.Background(), timeSeries)
	require.Error(t, err)
}

func TestRemoteWriter_Write_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block until context is cancelled
		<-r.Context().Done()
	}))
	defer server.Close()

	writer := NewRemoteWriter(server.URL)
	writer.maxRetries = 1

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	timeSeries := []prompb.TimeSeries{
		{
			Labels:  []prompb.Label{{Name: "__name__", Value: "test"}},
			Samples: []prompb.Sample{{Value: 1.0, Timestamp: 1000}},
		},
	}

	err := writer.Write(ctx, timeSeries)
	require.Error(t, err)
}
