// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

import (
	"github.com/cloudzero/cloudzero-insights-controller/app/instr"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	presignedURLRequestTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "presigned_url_request_total",
			Help: "Total number of pre-signed url requests.",
		},
		[]string{},
	)

	presignedURLRequestFailureTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "presigned_url_request_failure_total",
			Help: "Total number of pre-signed url request failures.",
		},
		[]string{},
	)

	remoteWriteFileTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "remote_write_file_total",
			Help: "Total number of files sent to the remote file reciever",
		},
		[]string{},
	)

	remoteWriteFailureTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "remote_write_failure_total",
			Help: "Total number of failures pushing to the remote file receiver.",
		},
		[]string{"file_count"},
	)

	// Replay Requests
	// ----------------------------------------------------------
	metricReplayRequestTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "shipper_replay_request_total",
			Help: "Total number of replay requests receieved from the remote file receiver.",
		},
		[]string{},
	)

	metricReplayRequestCurrent = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "shipper_replay_request_current",
			Help: "The current number of replay requests queued",
		},
		[]string{},
	)

	metricReplayRequestFileCount = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "shipper_replay_request_file_count",
			Help:    "Number of files requested for a replay request",
			Buckets: prometheus.ExponentialBuckets(10, 2, 15),
		},
	)

	metricReplayRequestErrorTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "shipper_replay_request_error_total",
			Help: "Number of errors observed while processing replay requests",
		},
		[]string{"error"},
	)

	metricReplayRequestAbandonFilesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "shipper_replay_request_abandon_files_total",
			Help: "total number of abandoned files",
		},
		[]string{},
	)

	metricReplayRequestAbandonFilesErrorTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "shipper_replay_request_abandon_files_error_total",
			Help: "total number of abandoned files",
		},
		[]string{"error"},
	)

	// Disk Usage
	// ----------------------------------------------------------

	metricDiskTotalSizeBytes = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "shipper_disk_total_size_bytes",
			Help: "Total Size (bytes) of the pv",
		},
		[]string{},
	)

	metricCurrentDiskUsageBytes = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "shipper_current_disk_usage_bytes",
			Help: "Size (bytes) currently used in the pv",
		},
		[]string{},
	)

	metricCurrentDiskUsagePercentage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "shipper_current_disk_usage_percentage",
			Help: "Percentage currently used in the pv",
		},
		[]string{},
	)

	metricCurrentDiskUnsentFile = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "shipper_current_disk_unsent_file",
			Help: "Current number of unsent files found in the pv",
		},
		[]string{},
	)

	metricCurrentDiskSentFile = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "shipper_current_disk_sent_file",
			Help: "Current number of sent files found in the pv",
		},
		[]string{},
	)

	metricCurrentDiskReplayRequest = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "shipper_current_disk_replay_request",
			Help: "Current number of replay requests found in the pv",
		},
		[]string{},
	)

	metricDiskCleanupFailureTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "shipper_disk_cleanup_failure_total",
			Help: "Number of failures when purging files for disk space",
		},
		[]string{"storage_warning", "error"},
	)

	metricDiskCleanupSuccessTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "shipper_disk_cleanup_success_total",
			Help: "Number of successes when purging files for disk space",
		},
		[]string{"storage_warning"},
	)

	metricDiskCleanupPercentage = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "shipper_disk_cleanup_percentage",
			Help:    "Percent removed from the storage volume during a purge operation",
			Buckets: prometheus.LinearBuckets(0, 10, 11), // 0% to 100% in steps of 10%
		},
	)
)

func InitMetrics() (*instr.PrometheusMetrics, error) {
	return instr.NewPrometheusMetrics(
		// instr.WithDefaultRegistry(),
		instr.WithPromMetrics(
			presignedURLRequestTotal,
			presignedURLRequestFailureTotal,
			remoteWriteFileTotal,
			remoteWriteFailureTotal,

			// replay requests
			metricReplayRequestTotal,
			metricReplayRequestCurrent,
			metricReplayRequestFileCount,
			metricReplayRequestErrorTotal,
			metricReplayRequestAbandonFilesTotal,
			metricReplayRequestAbandonFilesErrorTotal,

			// disk usage
			metricDiskTotalSizeBytes,
			metricCurrentDiskUsageBytes,
			metricCurrentDiskUsagePercentage,
			metricCurrentDiskUnsentFile,
			metricCurrentDiskSentFile,
			metricCurrentDiskReplayRequest,
			metricDiskCleanupFailureTotal,
			metricDiskCleanupSuccessTotal,
			metricDiskCleanupPercentage,
		),
	)
}
