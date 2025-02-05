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

	currentDiskUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "remote_write_backlog_records",
			Help: "Total Size (bytes) used in the pv",
		},
		[]string{"bytes_available"},
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
			currentDiskUsage,
		),
	)
}
