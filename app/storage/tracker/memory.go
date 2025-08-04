// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package tracker

import (
	"context"
	"sync"
	"time"

	"github.com/cloudzero/cloudzero-agent/app/domain/upload/ports"
	"github.com/rs/zerolog/log"
)

// MemoryTracker provides in-memory file tracking for testing purposes
type MemoryTracker struct {
	mu                sync.RWMutex
	fileEvents        []ports.FileCreationEvent
	lastPingTimestamp map[string]time.Time
}

// NewMemoryTracker creates a new in-memory file tracker
func NewMemoryTracker() *MemoryTracker {
	return &MemoryTracker{
		fileEvents:        make([]ports.FileCreationEvent, 0),
		lastPingTimestamp: make(map[string]time.Time),
	}
}

// TrackFileCreation records a file creation event in memory
func (t *MemoryTracker) TrackFileCreation(ctx context.Context, event ports.FileCreationEvent) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.fileEvents = append(t.fileEvents, event)
	
	log.Info().
		Str("organization_id", event.OrganizationID).
		Str("reference_id", event.ReferenceID).
		Str("key", event.Key).
		Str("cloud_account_id", event.CloudAccountID).
		Str("cluster_name", event.ClusterName).
		Str("shipper_id", event.ShipperID).
		Str("region", event.Region).
		Time("created_at", event.CreatedAt).
		Msg("File creation event tracked")

	return nil
}

// UpdateLastPing updates the organization's last ping timestamp
func (t *MemoryTracker) UpdateLastPing(ctx context.Context, organizationID string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now().UTC()
	t.lastPingTimestamp[organizationID] = now
	
	log.Info().
		Str("organization_id", organizationID).
		Time("last_ping", now).
		Msg("Organization last ping updated")

	return nil
}

// GetFileEvents returns all tracked file events (for testing/debugging)
func (t *MemoryTracker) GetFileEvents() []ports.FileCreationEvent {
	t.mu.RLock()
	defer t.mu.RUnlock()

	events := make([]ports.FileCreationEvent, len(t.fileEvents))
	copy(events, t.fileEvents)
	return events
}

// GetLastPing returns the last ping timestamp for an organization (for testing/debugging)
func (t *MemoryTracker) GetLastPing(organizationID string) (time.Time, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	timestamp, exists := t.lastPingTimestamp[organizationID]
	return timestamp, exists
}

// Clear resets all tracked data (for testing)
func (t *MemoryTracker) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.fileEvents = make([]ports.FileCreationEvent, 0)
	t.lastPingTimestamp = make(map[string]time.Time)
}