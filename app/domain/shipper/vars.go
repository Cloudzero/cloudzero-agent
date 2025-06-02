// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

// public
const (
	ReplaySubDirectory      = "replay"
	UploadedSubDirectory    = "uploaded"
	CriticalPurgePercent    = 20
	ReplayRequestHeader     = "X-CloudZero-Replay"
	ShipperIDRequestHeader  = "X-CloudZero-Shipper-ID"
	AppVersionRequestHeader = "X-CloudZero-Version"
)

// private
const (
	shipperWorkerCount  = 3
	expirationTime      = 3600
	filePermissions     = 0o755
	lockMaxRetry        = 60
	replayFileFormat    = "replay-%d.json"
	filesChunkSize      = 200
	remoteFileExtension = ".parquet"

	abandonAPIPath = "/abandon"
	uploadAPIPath  = "/upload"
)
