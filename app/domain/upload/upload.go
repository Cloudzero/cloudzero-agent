// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package upload

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudzero/cloudzero-agent/app/domain/upload/ports"
)

// Service handles upload URL generation and file tracking
type Service struct {
	storage StorageProvider
	tracker FileTracker
}

// StorageProvider is the interface for storage operations
type StorageProvider interface {
	GeneratePresignedURL(ctx context.Context, key string, expiration time.Duration) (string, error)
	BuildKey(params ports.KeyParams) string
}

// FileTracker is the interface for file tracking operations
type FileTracker interface {
	TrackFileCreation(ctx context.Context, event ports.FileCreationEvent) error
	UpdateLastPing(ctx context.Context, organizationID string) error
}

// NewService creates a new upload service with the provided dependencies
func NewService(storage StorageProvider, tracker FileTracker) *Service {
	return &Service{
		storage: storage,
		tracker: tracker,
	}
}

// UploadRequest represents the incoming upload request
type UploadRequest struct {
	Files          []FileRequest `json:"files"`
	OrganizationID string
	CloudAccountID string
	ClusterName    string
	ShipperID      string
	Region         string
}

// FileRequest represents a single file in the upload request
type FileRequest struct {
	ReferenceID string `json:"reference_id"`
}

// UploadResponse represents the response with presigned URLs
type UploadResponse struct {
	URLs   map[string]string `json:"urls"`
	Replay []ReplayURL       `json:"replay,omitempty"`
}

// ReplayURL represents a replay URL for the X-CloudZero-Replay header
type ReplayURL struct {
	RefID string `json:"ref_id"`
	URL   string `json:"url"`
}

// GenerateUploadURLs creates presigned URLs for the requested files
func (s *Service) GenerateUploadURLs(ctx context.Context, req UploadRequest) (*UploadResponse, error) {
	now := time.Now().UTC()
	expiration := 12 * time.Hour // 43200 seconds as per Python implementation
	
	urls := make(map[string]string)
	
	for _, file := range req.Files {
		// Build the key following the Python implementation pattern
		keyParams := ports.KeyParams{
			OrganizationID: req.OrganizationID,
			Year:           now.Year(),
			Month:          int(now.Month()),
			Day:            now.Day(),
			Hour:           now.Hour(),
			CloudAccountID: req.CloudAccountID,
			ClusterName:    req.ClusterName,
			ShipperID:      req.ShipperID,
			Region:         req.Region,
			ReferenceID:    file.ReferenceID,
		}
		
		key := s.storage.BuildKey(keyParams)
		
		// Generate presigned URL
		url, err := s.storage.GeneratePresignedURL(ctx, key, expiration)
		if err != nil {
			return nil, fmt.Errorf("failed to generate presigned URL for %s: %w", file.ReferenceID, err)
		}
		
		urls[file.ReferenceID] = url
		
		// Track file creation event
		event := ports.FileCreationEvent{
			OrganizationID: req.OrganizationID,
			ReferenceID:    file.ReferenceID,
			Key:            key,
			URL:            url,
			CreatedAt:      now,
			CloudAccountID: req.CloudAccountID,
			ClusterName:    req.ClusterName,
			ShipperID:      req.ShipperID,
			Region:         req.Region,
		}
		
		if err := s.tracker.TrackFileCreation(ctx, event); err != nil {
			// Log error but don't fail the request (following Python implementation pattern)
			// In a real implementation, you might want to fall back to SQS like the Python version
			continue
		}
	}
	
	// Update organization last ping
	if err := s.tracker.UpdateLastPing(ctx, req.OrganizationID); err != nil {
		// Log error but don't fail the request
	}
	
	return &UploadResponse{
		URLs: urls,
		// Replay URLs would be implemented based on business requirements
		Replay: []ReplayURL{},
	}, nil
}