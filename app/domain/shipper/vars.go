// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

import "errors"

// public
const (
	ReplaySubDirectory   = "replay"
	UploadedSubDirectory = "uploaded"
	CriticalPurgePercent = 20
)

// private
const (
	shipperWorkerCount = 10
	expirationTime     = 3600
	filePermissions    = 0o755
	lockMaxRetry       = 60
)

var (
	ErrUnauthorized = errors.New("unauthorized request - possible invalid API key")
	ErrNoURLs       = errors.New("no presigned URLs returned")
)
