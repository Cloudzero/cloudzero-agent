// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudzero/cloudzero-insights-controller/app/lock"
	"github.com/cloudzero/cloudzero-insights-controller/app/store"
	"github.com/cloudzero/cloudzero-insights-controller/app/types"
	"github.com/go-obvious/timestamp"
	"github.com/rs/zerolog/log"
)

type ReplayRequest struct {
	Filepath     string             `json:"filepath"`
	ReferenceIDs *types.Set[string] `json:"referenceIds"` //nolint:tagliatelle // I dont want to use IDs
}

type replayRequestHeader struct {
	RefID string `json:"ref_id"` //nolint:tagliatelle // upstream uses cammel case
	URL   string `json:"url"`
}

func NewReplayRequestFromHeader(value string) (*ReplayRequest, error) {
	// parse into list type
	rrh := make([]replayRequestHeader, 0)
	if err := json.Unmarshal([]byte(value), &rrh); err != nil {
		return nil, fmt.Errorf("failed to parse the replay request data: %w", err)
	}

	// convert to the replay request
	rr := ReplayRequest{
		ReferenceIDs: types.NewSet[string](),
	}
	for _, item := range rrh {
		rr.ReferenceIDs.Add(item.RefID)
	}

	return &rr, nil
}

// Saves a reply-request from the remote to disk to be picked up on next iteration
func (m *MetricShipper) SaveReplayRequest(rr *ReplayRequest) error {
	return m.metrics.Span("shipper_SaveReplayRequest", func() error {
		// create the directory if needed
		replayDir := m.GetReplayRequestDir()
		if err := os.MkdirAll(replayDir, filePermissions); err != nil {
			return fmt.Errorf("failed to create the replay request directory: %w", err)
		}

		// compose the filename
		rr.Filepath = filepath.Join(m.GetReplayRequestDir(), fmt.Sprintf(replayFileFormat, timestamp.Milli()))

		// encode to json
		enc, err := json.Marshal(rr)
		if err != nil {
			return fmt.Errorf("failed to encode the replay request to json: %w", err)
		}

		// write the file
		if err := os.WriteFile(rr.Filepath, enc, filePermissions); err != nil {
			return fmt.Errorf("failed to write the replay request to file: %w", err)
		}

		return nil
	})
}

// gets all active replay request files
func (m *MetricShipper) GetActiveReplayRequests() ([]*ReplayRequest, error) {
	requests := make([]*ReplayRequest, 0)

	err := m.metrics.Span("shipper_GetActiveReplayRequests", func() error {
		// create the directory if needed
		replayDir := m.GetReplayRequestDir()
		if err := os.MkdirAll(replayDir, filePermissions); err != nil {
			return fmt.Errorf("failed to create the replay request directory: %w", err)
		}

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

			// skip over invalid files (like lock files)
			if !strings.Contains(item.Name(), strings.Split(replayFileFormat, "-")[0]) || !strings.Contains(item.Name(), ".json") {
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
	})
	if err != nil {
		return nil, err
	}

	return requests, nil
}

func (m *MetricShipper) ProcessReplayRequests() error {
	return m.metrics.Span("shipper_ProcessReplayRequests", func() error {
		log.Ctx(m.ctx).Info().Msg("Processing replay requests")

		// ensure the directory is created
		if err := os.MkdirAll(m.GetReplayRequestDir(), filePermissions); err != nil {
			return fmt.Errorf("failed to create the replay request file directory: %w", err)
		}

		// lock the replay request dir for the duration of the replay request processing
		log.Ctx(m.ctx).Debug().Msg("Aquiring file lock")
		l := lock.NewFileLock(
			m.ctx, filepath.Join(m.GetReplayRequestDir(), ".lock"),
			lock.WithStaleTimeout(time.Second*30), // detects stale timeout
			lock.WithRefreshInterval(time.Second*5),
			lock.WithMaxRetry(lockMaxRetry), // 5 min wait
		)
		if err := l.Acquire(); err != nil {
			return fmt.Errorf("failed to aquire the lock: %w", err)
		}
		defer func() {
			if err := l.Release(); err != nil {
				log.Ctx(m.ctx).Error().Err(err).Msg("Failed to release the lock")
			}
		}()

		// read all valid replay request files
		requests, err := m.GetActiveReplayRequests()
		if err != nil {
			return fmt.Errorf("failed to get replay requests: %w", err)
		}

		if len(requests) == 0 {
			log.Ctx(m.ctx).Info().Msg("No replay requests found, skipping")
			return nil
		}

		// handle all valid replay requests
		for _, rr := range requests {
			metricReplayRequestFileCount.Observe(float64(rr.ReferenceIDs.Size()))

			if err := m.HandleReplayRequest(rr); err != nil {
				metricReplayRequestErrorTotal.WithLabelValues(err.Error()).Inc()
				return fmt.Errorf("failed to process replay request '%s': %w", rr.Filepath, err)
			}

			// decrease the current queue for this replay request
			metricReplayRequestCurrent.WithLabelValues().Dec()
		}

		log.Ctx(m.ctx).Info().Msg("Successfully handled all replay requests")

		return nil
	})
}

func (m *MetricShipper) HandleReplayRequest(rr *ReplayRequest) error {
	return m.metrics.Span("shipper_HandleReplayRequest", func() error {
		log.Ctx(m.ctx).Debug().Str("rr", rr.Filepath).Int("numfiles", rr.ReferenceIDs.Size()).Msg("Handling replay request")

		// fetch the new files that match these ids
		newFiles := make([]types.File, 0)
		if err := m.metrics.Span("shipper_HandleReplayRequest_listNewFiles", func() error {
			return m.lister.Walk("", func(path string, info fs.FileInfo, err error) error {
				if err != nil {
					return err
				}

				// skip dir
				if info.IsDir() {
					return nil
				}

				// create a new types.File to compare the remote ids
				storeFile, err := store.NewMetricFile(path)
				if err != nil {
					return errors.New("failed to create a new metric file")
				}

				// check for a match
				if rr.ReferenceIDs.Contains(GetRemoteFileID(storeFile)) {
					newFiles = append(newFiles, storeFile)
				}

				return nil
			})
		}); err != nil {
			return fmt.Errorf("failed to get matching new files: %w", err)
		}
		log.Ctx(m.ctx).Debug().Int("newFiles", len(newFiles)).Send()

		// fetch the already uploadedFiles files that match these ids
		uploadedFiles := make([]types.File, 0)
		if err := m.metrics.Span("shipper_HandleReplayRequest_listUploadedFiles", func() error {
			return m.lister.Walk(UploadedSubDirectory, func(path string, info fs.FileInfo, err error) error {
				if err != nil {
					return err
				}

				// skip dir
				if info.IsDir() {
					return nil
				}

				// create a new types.File to compare the remote ids
				storeFile, err := store.NewMetricFile(path)
				if err != nil {
					return errors.New("failed to create a new metric file")
				}

				// check for a match
				if rr.ReferenceIDs.Contains(GetRemoteFileID(storeFile)) {
					uploadedFiles = append(uploadedFiles, storeFile)
				}

				return nil
			})
		}); err != nil {
			return fmt.Errorf("failed to get matching uploaded files: %w", err)
		}
		log.Ctx(m.ctx).Debug().Int("uploadedFiles", len(uploadedFiles)).Send()

		// create a file array of all the files found to send to the remote
		total := make([]types.File, 0)
		total = append(total, newFiles...)
		total = append(total, uploadedFiles...)

		// check for missing files from the replay request
		found := types.NewSet[string]()
		for _, item := range total {
			found.Add(GetRemoteFileID(item))
		}
		log.Ctx(m.ctx).Info().Msgf("Replay request '%s': %d/%d files found", rr.Filepath, found.Size(), rr.ReferenceIDs.Size())

		// compare the results and discover which files were not found
		missing := rr.ReferenceIDs.Diff(found)

		// send abandon requests for the non-found files
		if missing.Size() > 0 {
			log.Info().Msgf("Replay request '%s': %d files missing, sending abandon request for these files", rr.Filepath, missing.Size())
			if err := m.AbandonFiles(missing.List(), "not found"); err != nil {
				metricReplayRequestAbandonFilesErrorTotal.WithLabelValues(err.Error()).Inc()
				return fmt.Errorf("failed to send the abandon file request: %w", err)
			}
		}

		// run the `HandleRequest` function for the found files
		if err := m.HandleRequest(total); err != nil {
			return fmt.Errorf("failed to upload replay request files: %w", err)
		}

		// delete the replay request
		if err := os.Remove(rr.Filepath); err != nil {
			return fmt.Errorf("failed to delete the replay request file: %w", err)
		}

		log.Ctx(m.ctx).Info().Str("rr", rr.Filepath).Msg("Successfully handled replay request")

		return nil
	})
}
