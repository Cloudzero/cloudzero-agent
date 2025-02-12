// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudzero/cloudzero-insights-controller/app/lock"
	"github.com/go-obvious/timestamp"
)

const (
	replayFilePermissions = 0o755
)

type ReplayRequest struct {
	Filepath     string   `json:"filepath"`
	ReferenceIDs []string `json:"referenceIds"` //nolint:tagliatelle // I dont want to use IDs
}

// Saves a reply-request from the remote to disk to be picked up on next iteration
func (m *MetricShipper) SaveReplayRequest(ids []string) (*ReplayRequest, error) {
	// create the directory if needed
	replayDir := m.GetReplayRequestDir()
	if err := os.MkdirAll(replayDir, replayFilePermissions); err != nil {
		return nil, fmt.Errorf("failed to create the replay request directory: %w", err)
	}

	// lock the replay directory
	var rr *ReplayRequest
	if err := lock.LockDir(m.ctx, replayDir, func() error {
		// compose the filename
		filename := filepath.Join(m.GetReplayRequestDir(), fmt.Sprintf("replay-%d.json", timestamp.Milli()))

		// create the payload
		rr = &ReplayRequest{
			Filepath:     filename,
			ReferenceIDs: ids,
		}

		// encode to json
		enc, err := json.Marshal(rr)
		if err != nil {
			return fmt.Errorf("failed to encode the replay request to json: %w", err)
		}

		// write the file
		return os.WriteFile(filename, enc, replayFilePermissions)
	}); err != nil {
		return nil, err
	}

	return rr, nil
}

// gets all active replay request files
func (m *MetricShipper) GetActiveReplayRequests() ([]*ReplayRequest, error) {
	// create the directory if needed
	replayDir := m.GetReplayRequestDir()
	if err := os.MkdirAll(replayDir, replayFilePermissions); err != nil {
		return nil, fmt.Errorf("failed to create the replay request directory: %w", err)
	}

	requests := make([]*ReplayRequest, 0)

	// lock the replay request directory while we read from it
	if err := lock.LockDir(m.ctx, replayDir, func() error {
		// list all files
		entries, err := os.ReadDir(replayDir)
		if err != nil {
			return fmt.Errorf("failed to list the directory: %w", err)
		}

		// iterate and parse files
		for _, item := range entries {
			if item.IsDir() {
				continue
			}

			// read the file
			fullpath := filepath.Join(m.GetReplayRequestDir(), item.Name())
			data, err := os.ReadFile(fullpath)
			if err != nil {
				return fmt.Errorf("failed to read the file '%s': %w", fullpath, err)
			}

			// unserialize
			rr := ReplayRequest{}
			if err := json.Unmarshal(data, &rr); err != nil {
				return fmt.Errorf("failed to decode the replay request: %w", err)
			}
			requests = append(requests, &rr)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return requests, nil
}
