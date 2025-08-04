// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package ports

import (
	"context"
	"time"
)

// StorageProvider defines the interface for object storage operations
type StorageProvider interface {
	// GeneratePresignedURL creates a presigned URL for uploading a file
	GeneratePresignedURL(ctx context.Context, key string, expiration time.Duration) (string, error)
	
	// BuildKey constructs the S3-compatible key path for a file
	BuildKey(params KeyParams) string
}

// KeyParams contains the parameters needed to build a storage key
type KeyParams struct {
	OrganizationID string
	Year           int
	Month          int
	Day            int
	Hour           int
	CloudAccountID string
	ClusterName    string
	ShipperID      string
	Region         string
	ReferenceID    string
}