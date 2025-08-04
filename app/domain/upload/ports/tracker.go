// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package ports

import (
	"context"
	"time"
)

// FileTracker defines the interface for tracking file upload events
type FileTracker interface {
	// TrackFileCreation records a file creation event
	TrackFileCreation(ctx context.Context, event FileCreationEvent) error
	
	// UpdateLastPing updates the organization's last ping timestamp
	UpdateLastPing(ctx context.Context, organizationID string) error
}

// FileCreationEvent represents a file creation event to be tracked
type FileCreationEvent struct {
	OrganizationID string
	ReferenceID    string
	Key            string
	URL            string
	CreatedAt      time.Time
	CloudAccountID string
	ClusterName    string
	ShipperID      string
	Region         string
}