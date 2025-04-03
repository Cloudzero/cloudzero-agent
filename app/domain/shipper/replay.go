// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudzero/cloudzero-agent-validator/app/instr"
	"github.com/cloudzero/cloudzero-agent-validator/app/lock"
	"github.com/cloudzero/cloudzero-agent-validator/app/store"
	"github.com/cloudzero/cloudzero-agent-validator/app/types"
	"github.com/go-obvious/timestamp"
	"github.com/rs/zerolog"
)

type ReplayRequest struct {
	Filepath     string             `json:"filepath"`
	ReferenceIDs *types.Set[string] `json:"referenceIds"` //nolint:tagliatelle // I dont want to use IDs
}

type replayRequestHeaderValue struct {
	RefID string `json:"ref_id"` //nolint:tagliatelle // upstream uses cammel case
	URL   string `json:"url"`
}

func NewReplayRequestFromHeader(value string) (*ReplayRequest, error) {
	// parse into list type
	rrh := make([]replayRequestHeaderValue, 0)
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

// SaveReplayRequest saves a reply-request from the remote to disk to be picked
// up on next iteration.
func (m *MetricShipper) SaveReplayRequest(ctx context.Context, rr *ReplayRequest) error {
	return m.metrics.SpanCtx(ctx, "shipper_SaveReplayRequest", func(ctx context.Context, id string) error {
		// create the directory if needed
		replayDir := m.GetReplayRequestDir()
		if err := os.MkdirAll(replayDir, filePermissions); err != nil {
			return errors.Join(ErrCreateDirectory, fmt.Errorf("failed to create the replay request directory: %w", err))
		}

		// compose the filename
		rr.Filepath = filepath.Join(m.GetReplayRequestDir(), fmt.Sprintf(replayFileFormat, timestamp.Milli()))

		// encode to json
		enc, err := json.Marshal(rr)
		if err != nil {
			return errors.Join(ErrEncodeBody, fmt.Errorf("failed to encode the replay request to json: %w", err))
		}

		// write the file
		if err := os.WriteFile(rr.Filepath, enc, filePermissions); err != nil {
			return errors.Join(ErrFileCreate, fmt.Errorf("failed to write the replay request to file: %w", err))
		}

		return nil
	})
}

// GetActiveReplayRequests gets all active replay request files
func (m *MetricShipper) GetActiveReplayRequests(ctx context.Context) ([]*ReplayRequest, error) {
	requests := make([]*ReplayRequest, 0)

	err := m.metrics.SpanCtx(ctx, "shipper_GetActiveReplayRequests", func(ctx context.Context, id string) error {
		// create the directory if needed
		replayDir := m.GetReplayRequestDir()
		if err := os.MkdirAll(replayDir, filePermissions); err != nil {
			return errors.Join(ErrCreateDirectory, fmt.Errorf("failed to create the replay request directory: %w", err))
		}

		// list all files
		entries, err := os.ReadDir(replayDir)
		if err != nil {
			return errors.Join(ErrFilesList, fmt.Errorf("failed to list the replay request directory: %w", err))
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
				return errors.Join(ErrFileRead, fmt.Errorf("failed to read the replay request file: path=%s, err=%w", fullpath, err))
			}

			// unserialize
			rr := ReplayRequest{}
			if err := json.Unmarshal(data, &rr); err != nil {
				return errors.Join(ErrInvalidBody, fmt.Errorf("failed to decode the replay request: %w", err))
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

func (m *MetricShipper) ProcessReplayRequests(ctx context.Context) error {
	return m.metrics.SpanCtx(ctx, "shipper_ProcessReplayRequests", func(ctx context.Context, id string) error {
		logger := instr.SpanLogger(ctx, id)
		logger.Debug().Msg("Processing replay requests")

		// ensure the directory is created
		if err := os.MkdirAll(m.GetReplayRequestDir(), filePermissions); err != nil {
			return errors.Join(ErrCreateDirectory, fmt.Errorf("failed to create the replay request file directory: %w", err))
		}

		// lock the replay request dir for the duration of the replay request processing
		logger.Debug().Msg("Aquiring replay request file lock")
		l := lock.NewFileLock(
			m.ctx, filepath.Join(m.GetReplayRequestDir(), ".lock"),
			lock.WithStaleTimeout(time.Second*30), // detects stale timeout
			lock.WithRefreshInterval(time.Second*5),
			lock.WithMaxRetry(lockMaxRetry), // 5 min wait
		)
		if err := l.Acquire(); err != nil {
			return errors.Join(ErrCreateLock, fmt.Errorf("failed to acquire replay request lock: %w", err))
		}
		defer func() {
			if err := l.Release(); err != nil {
				logger.Err(err).Msg("failed to release the replay request lock")
			}
		}()

		logger.Debug().Msg("Successfully acquired file lock")

		// read all valid replay request files
		requests, err := m.GetActiveReplayRequests(ctx)
		if err != nil {
			return fmt.Errorf("failed to get replay requests: %w", err)
		}

		if len(requests) == 0 {
			logger.Debug().Msg("No replay requests found, skipping")
			return nil
		}

		logger.Debug().Int("length", len(requests)).Msg("Processing replay requests")

		// handle all valid replay requests
		for _, rr := range requests {
			logger.Debug().Str("replayRequestFilepath", rr.Filepath).Int("referenceIds", rr.ReferenceIDs.Size()).Msg("Processing replay request")
			metricReplayRequestFileCount.Observe(float64(rr.ReferenceIDs.Size()))

			if err := m.HandleReplayRequest(ctx, rr); err != nil {
				return fmt.Errorf("failed to process replay request '%s': %w", rr.Filepath, err)
			}

			// decrease the current queue for this replay request
			logger.Debug().Str("replayRequestFilepath", rr.Filepath).Msg("Successfully processed replay request")
			metricReplayRequestCurrent.WithLabelValues().Dec()
		}

		logger.Debug().Msg("Successfully handled all replay requests")

		return nil
	})
}

func (m *MetricShipper) HandleReplayRequest(ctx context.Context, rr *ReplayRequest) error {
	return m.metrics.SpanCtx(ctx, "shipper_HandleReplayRequest", func(ctx context.Context, id string) error {
		logger := instr.SpanLogger(ctx, id, func(ctx zerolog.Context) zerolog.Context {
			return ctx.Str("rr", rr.Filepath).Int("numfiles", rr.ReferenceIDs.Size())
		})
		logger.Debug().Msg("Handling replay request ...")

		// fetch the new files that match these ids
		logger.Debug().Msg("Searching for new files in the disk store")
		newFiles := make([]types.File, 0)
		if err := m.metrics.SpanCtx(ctx, "shipper_HandleReplayRequest_listNewFiles", func(ctx context.Context, id string) error {
			return m.store.Walk("", func(path string, info fs.FileInfo, err error) error {
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
			return errors.Join(ErrFilesList, fmt.Errorf("failed to get the matching new files: %w", err))
		}
		logger.Debug().Int("files", len(newFiles)).Msg("found new files")

		// fetch the already uploadedFiles files that match these ids
		logger.Debug().Msg("Searching for previously uploaded files ...")
		uploadedFiles := make([]types.File, 0)
		if err := m.metrics.SpanCtx(ctx, "shipper_HandleReplayRequest_listUploadedFiles", func(ctx context.Context, id string) error {
			return m.store.Walk(UploadedSubDirectory, func(path string, info fs.FileInfo, err error) error {
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
			return errors.Join(ErrFilesList, fmt.Errorf("failed to get matching uploaded files: %w", err))
		}
		logger.Debug().Int("files", len(uploadedFiles)).Msg("found uploaded files")

		// create a file array of all the files found to send to the remote
		logger.Debug().Msg("Combining all files into an array")
		total := make([]types.File, 0)
		total = append(total, newFiles...)
		total = append(total, uploadedFiles...)

		// check for missing files from the replay request
		found := types.NewSet[string]()
		for _, item := range total {
			found.Add(GetRemoteFileID(item))
		}
		logger.Debug().Int("found", found.Size()).Int("totalRequested", rr.ReferenceIDs.Size()).Msg("Replay request files found")

		// compare the results and discover which files were not found
		missing := rr.ReferenceIDs.Diff(found)

		// send abandon requests for the non-found files
		if missing.Size() > 0 {
			logger.Debug().Int("numNotFound", missing.Size()).Msg("Sending abandon requests for not found files")
			if err := m.AbandonFiles(ctx, missing.List(), "not found"); err != nil {
				metricReplayRequestAbandonFilesErrorTotal.WithLabelValues(GetErrStatusCode(err)).Inc()
				return fmt.Errorf("failed to send the abandon file request: %w", err)
			}
			logger.Debug().Msg("Successfully sent the abandon requests")

			// log the number of success abandoned files
			metricReplayRequestAbandonFilesTotal.WithLabelValues().Add(float64(missing.Size()))
		}

		// run the `HandleRequest` function for the found files
		if err := m.HandleRequest(ctx, total); err != nil {
			return fmt.Errorf("failed to upload replay request files: %w", err)
		}

		// delete the replay request
		logger.Debug().Msg("Deleting the replay request")
		if err := os.Remove(rr.Filepath); err != nil {
			return errors.Join(ErrFileRemove, fmt.Errorf("failed to delete the replay request file: %w", err))
		}

		logger.Debug().Str("rr", rr.Filepath).Msg("Successfully handled replay request")

		return nil
	})
}
