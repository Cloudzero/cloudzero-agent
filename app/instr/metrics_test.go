// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package instr

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func TestPrometheusHandler(t *testing.T) {
	// create a mock http service
	srv := httptest.NewServer(Handler())
	defer srv.Close()

	// record a sample metric
	testCounter.Add(context.Background(), 10)

	// check the result
	resp, err := http.Get(srv.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Contains(t, string(body), "test_counter")
	require.Contains(t, string(body), "10")
}

func TestMetrics(t *testing.T) {
	// create mock http server
	srv := httptest.NewServer(Handler())
	defer srv.Close()

	getBody := func(t *testing.T) string {
		resp, err := http.Get(srv.URL)
		require.NoError(t, err)
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		return string(body)
	}

	t.Run("RemoteWriteRequestCount", func(t *testing.T) {
		RemoteWriteRequestCount.Add(context.Background(), 1,
			metric.WithAttributes(
				attribute.String("endpoint", "https://test-endpoint-1.com"),
			),
		)

		RemoteWriteRequestCount.Add(context.Background(), 2,
			metric.WithAttributes(
				attribute.String("endpoint", "https://test-endpoint-2.com"),
			),
		)

		body := getBody(t)

		// ensure that things are being recorded correctly
		for _, line := range strings.Split(string(body), "\n") {
			if strings.Contains(line, "https://test-endpoint-1.com") {
				require.Contains(t, line, "1")
			}
			if strings.Contains(line, "https://test-endpoint-2.com") {
				require.Contains(t, line, "2")
			}
		}
	})

	t.Run("RemoteWriteRequestDurationSeconds", func(t *testing.T) {
		// add some sample ranges
		var sum float64 = 0
		for _, item := range []float64{0.02, 2.3, 10} {
			sum += item
			RemoteWriteRequestDurationSeconds.Record(context.Background(), item)
		}

		body := getBody(t)

		// ensure they are grouped into buckets
		for _, line := range strings.Split(string(body), "\n") {
			if strings.Contains(line, "le=\"0.025\"") {
				require.Contains(t, line, "1")
			}
			if strings.Contains(line, "le=\"5\"") {
				require.Contains(t, line, "2")
			}
			if strings.Contains(line, "le=\"10\"") {
				require.Contains(t, line, "3")
			}
			if strings.Contains(line, "remote_write_request_duration_count") {
				require.Contains(t, line, "3")
			}
			if strings.Contains(line, "remote_write_request_duration_sum") {
				require.Contains(t, line, fmt.Sprintf("%v", sum))
			}
		}
	})

	t.Run("RemoteWriteStatusCodes", func(t *testing.T) {
		// add various statuses
		statuses := []int{
			http.StatusOK,
			http.StatusInternalServerError,
			http.StatusBadRequest,
			http.StatusUnauthorized,
		}
		for _, item := range statuses {
			RemoteWriteStatusCodes.Add(context.Background(), 1, metric.WithAttributes(attribute.Int("code", item)))
		}
		body := getBody(t)

		// validate the body
		seen := 0
		for _, line := range strings.Split(body, "\n") {
			if strings.Contains(line, "remote_write_status_codes_total{") {
				seen += 1
				require.Contains(t, line, "} 1")
			}
		}
		require.Equal(t, 4, seen)
	})

	t.Run("RemoteWritePayloadSizeBytes", func(t *testing.T) {
		sizes := []int64{2034, 34201, 20432, 93384}
		var total int64 = 0
		for _, item := range sizes {
			total += item
			RemoteWritePayloadSizeBytes.Record(context.Background(), item)
		}
		body := getBody(t)
		fmt.Println(body)
		for _, line := range strings.Split(body, "\n") {
			if strings.Contains(line, "remote_write_payload_size_bytes_total{") {
				require.Contains(t, line, fmt.Sprintf("%v", total))
			}
		}
	})

	t.Run("RemoteWriteFailures", func(t *testing.T) {
		RemoteWriteFailures.Add(context.Background(), 1)
		body := getBody(t)
		for _, line := range strings.Split(body, "\n") {
			if strings.Contains(line, "remote_write_failures_total{") {
				require.Contains(t, line, "1")
			}
		}
	})
}
