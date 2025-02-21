// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShipper_Metrics(t *testing.T) {
	pm, err := InitMetrics()
	require.NoError(t, err)

	srv := httptest.NewServer(pm.Handler())
	defer srv.Close()

	// record all metrics
	presignedURLRequestTotal.WithLabelValues().Inc()
	presignedURLRequestFailureTotal.WithLabelValues().Inc()
	remoteWriteFileTotal.WithLabelValues().Inc()
	remoteWriteFailureTotal.WithLabelValues("10").Inc()
	replayRequestTotal.WithLabelValues().Inc()

	// disk usage
	metricDiskTotalSizeBytes.WithLabelValues().Inc()
	metricCurrentDiskUsageBytes.WithLabelValues().Inc()
	metricCurrentDiskUsagePercentage.WithLabelValues().Inc()
	metricCurrentDiskUnsentFile.WithLabelValues().Inc()
	metricCurrentDiskSentFile.WithLabelValues().Inc()
	metricCurrentDiskReplayRequest.WithLabelValues().Inc()
	metricDiskCleanupSuccessTotal.WithLabelValues("low").Inc()
	metricDiskCleanupFailureTotal.WithLabelValues("none").Inc()

	// fetch metrics from the mock handler
	resp, err := http.Get(srv.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// validate
	require.Contains(t, string(body), "presigned_url_request_total")
	require.Contains(t, string(body), "presigned_url_request_failure_total")
	require.Contains(t, string(body), "remote_write_file_total")
	require.Contains(t, string(body), "remote_write_failure_total")
	require.Contains(t, string(body), "replay_request_total")

	// disk usage
	require.Contains(t, string(body), "shipper_disk_total_size_bytes")
	require.Contains(t, string(body), "shipper_current_disk_usage_bytes")
	require.Contains(t, string(body), "shipper_current_disk_usage_percentage")
	require.Contains(t, string(body), "shipper_current_disk_unsent_file")
	require.Contains(t, string(body), "shipper_current_disk_sent_file")
	require.Contains(t, string(body), "shipper_current_disk_replay_request")
	require.Contains(t, string(body), "shipper_disk_cleanup_success_total")
	require.Contains(t, string(body), "shipper_disk_cleanup_failure_total")
}
