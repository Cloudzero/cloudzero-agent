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

func TestShipper_Unit_Metrics(t *testing.T) {
	pm, err := InitMetrics()
	require.NoError(t, err)

	srv := httptest.NewServer(pm.Handler())
	defer srv.Close()

	// other
	metricShutdownTotal.WithLabelValues().Inc()

	// new files
	metricNewFilesErrorTotal.WithLabelValues().Inc()
	metricNewFilesProcessingCurrent.WithLabelValues().Inc()

	// generic request handling
	metricHandleRequestFileCount.Observe(20)
	metricHandleRequestSuccessTotal.WithLabelValues().Inc()

	// presigned urls
	metricPresignedURLErrorTotal.WithLabelValues().Inc()

	// replay requests
	metricReplayRequestTotal.WithLabelValues().Inc()
	metricReplayRequestCurrent.WithLabelValues().Inc()
	metricReplayRequestFileCount.Observe(100)
	metricReplayRequestErrorTotal.WithLabelValues().Inc()
	metricReplayRequestAbandonFilesTotal.WithLabelValues().Inc()
	metricReplayRequestAbandonFilesErrorTotal.WithLabelValues().Inc()

	// disk usage
	metricDiskTotalSizeBytes.WithLabelValues().Inc()
	metricCurrentDiskUsageBytes.WithLabelValues().Inc()
	metricCurrentDiskUsagePercentage.WithLabelValues().Inc()
	metricCurrentDiskUnsentFile.WithLabelValues().Inc()
	metricCurrentDiskSentFile.WithLabelValues().Inc()
	metricCurrentDiskReplayRequest.WithLabelValues().Inc()
	metricDiskCleanupFailureTotal.WithLabelValues("80", "error").Inc()
	metricDiskCleanupSuccessTotal.WithLabelValues("40").Inc()
	metricDiskCleanupPercentage.Observe(20)

	// fetch metrics from the mock handler
	resp, err := http.Get(srv.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// validate

	require.Contains(t, string(body), "shipper_shutdown_total")

	require.Contains(t, string(body), "shipper_new_files_error_total")
	require.Contains(t, string(body), "shipper_new_files_processing_current")

	require.Contains(t, string(body), "shipper_handle_request_file_count")
	require.Contains(t, string(body), "shipper_handle_request_success_total")

	require.Contains(t, string(body), "shipper_presigned_url_error_total")

	// replay requests
	require.Contains(t, string(body), "shipper_replay_request_total")
	require.Contains(t, string(body), "shipper_replay_request_current")
	require.Contains(t, string(body), "shipper_replay_request_file_count")
	require.Contains(t, string(body), "shipper_replay_request_error_total")
	require.Contains(t, string(body), "shipper_replay_request_abandon_files_total")
	require.Contains(t, string(body), "shipper_replay_request_abandon_files_error_total")

	// disk usage
	require.Contains(t, string(body), "shipper_disk_total_size_bytes")
	require.Contains(t, string(body), "shipper_disk_current_usage_bytes")
	require.Contains(t, string(body), "shipper_disk_current_usage_percentage")
	require.Contains(t, string(body), "shipper_disk_current_unsent_file")
	require.Contains(t, string(body), "shipper_disk_current_sent_file")
	require.Contains(t, string(body), "shipper_disk_replay_request_current")
	require.Contains(t, string(body), "shipper_disk_cleanup_failure_total")
	require.Contains(t, string(body), "shipper_disk_cleanup_success_total")
	require.Contains(t, string(body), "shipper_disk_cleanup_percentage")
}
