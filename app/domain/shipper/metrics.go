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

	replayRequestTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "replay_request_total",
			Help: "Total number of replay requests receieved from the remote file receiver.",
		},
		[]string{},
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

	metricCurrentDiskUnsentFileCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "shipper_current_disk_unsent_file_count",
			Help: "Current number of unsent files found in the pv",
		},
		[]string{},
	)

	metricCurrentDiskSentFileCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "shipper_current_disk_sent_file_count",
			Help: "Current number of sent files found in the pv",
		},
		[]string{},
	)

	metricCurrentDiskReplayRequestCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "shipper_current_disk_replay_request_count",
			Help: "Current number of replay requests found in the pv",
		},
		[]string{},
	)

	metricDiskCleanupSuccessTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "shipper_disk_cleanup_success_total",
			Help: "Number of successes when purging files for disk space",
		},
		[]string{"storage_warning"},
	)

	metricDiskCleanupFailureTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "shipper_disk_cleanup_failure_total",
			Help: "Number of failures when purging files for disk space",
		},
		[]string{"storage_warning"},
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
			replayRequestTotal,

			// disk usage
			metricDiskTotalSizeBytes,
			metricCurrentDiskUsageBytes,
			metricCurrentDiskUsagePercentage,
			metricCurrentDiskUnsentFileCount,
			metricCurrentDiskSentFileCount,
			metricCurrentDiskReplayRequestCount,
			metricDiskCleanupSuccessTotal,
			metricDiskCleanupFailureTotal,
		),
	)
}
