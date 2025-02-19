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
	currentDiskUsage.WithLabelValues("10000").Inc()

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
	require.Contains(t, string(body), "remote_write_backlog_records")
}
