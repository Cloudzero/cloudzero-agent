// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	config "github.com/cloudzero/cloudzero-agent/app/config/gator"
	"github.com/cloudzero/cloudzero-agent/app/logging/instr"
	"github.com/cloudzero/cloudzero-agent/app/storage/disk"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/cloudzero/cloudzero-agent/app/utils/lock"
	"github.com/cloudzero/cloudzero-agent/app/utils/parallel"
	"github.com/google/uuid"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// MetricShipper handles the periodic shipping of metrics to Cloudzero.
type MetricShipper struct {
	setting *config.Settings
	store   types.ReadableStore

	// Internal fields
	ctx          context.Context
	cancel       context.CancelFunc
	HTTPClient   *retryablehttp.Client
	shippedFiles uint64 // Counter for shipped files
	metrics      *instr.PrometheusMetrics
	shipperID    string // unique id for the shipper
}

// NewMetricShipper initializes a new MetricShipper.
func NewMetricShipper(ctx context.Context, s *config.Settings, store types.ReadableStore) (*MetricShipper, error) {
	ctx, cancel := context.WithCancel(ctx)

	// Initialize an HTTP client with the specified timeout
	httpClient := NewHTTPClient(ctx, s)

	// create the metrics listener
	metrics, err := InitMetrics()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to init metrics: %w", err)
	}

	// dump the config
	if log.Ctx(ctx).GetLevel() <= zerolog.DebugLevel {
		enc, err := json.MarshalIndent(s, "", "  ")
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to marshal the config as json: %w", err)
		}
		fmt.Println(string(enc))
	}

	return &MetricShipper{
		setting:    s,
		store:      store,
		ctx:        ctx,
		cancel:     cancel,
		HTTPClient: httpClient,
		metrics:    metrics,
	}, nil
}

func (m *MetricShipper) GetMetricHandler() http.Handler {
	return m.metrics.Handler()
}

// Run starts the MetricShipper service and blocks until a shutdown signal is received.
func (m *MetricShipper) Run() error {
	// create the required directories for this application
	if err := os.MkdirAll(m.GetUploadedDir(), filePermissions); err != nil {
		return errors.Join(ErrCreateDirectory, fmt.Errorf("failed to create the uploaded directory: %w", err))
	}

	// Set up channel to listen for OS signals
	sigChan := make(chan os.Signal, 1)
	// Listen for interrupt and termination signals
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	// Initialize ticker for periodic shipping
	ticker := time.NewTicker(m.setting.Cloudzero.SendInterval)
	defer ticker.Stop()

	log.Ctx(m.ctx).Info().Msg("Shipper service starting ...")

	// run at the start
	if err := m.runShipper(m.ctx); err != nil {
		log.Ctx(m.ctx).Err(err).Msg("Failed to run shipper")
		metricShipperRunFailTotal.WithLabelValues(GetErrStatusCode(err)).Inc()
	}

	for {
		select {
		case <-m.ctx.Done():
			log.Ctx(m.ctx).Info().Msg("Shipper service stopping")
			return nil

		case sig := <-sigChan:
			log.Ctx(m.ctx).Info().Str("signal", sig.String()).Msg("Received signal. Initiating shutdown.")

			// flush
			if err := m.ProcessFiles(m.ctx); err != nil {
				metricNewFilesErrorTotal.WithLabelValues(GetErrStatusCode(err)).Inc()
				log.Ctx(m.ctx).Err(err).Msg("Failed to flush the new metric files")
			}

			err := m.Shutdown()
			if err != nil {
				log.Ctx(m.ctx).Err(err).Msg("Failed to shutdown shipper service")
			}
			return nil

		case <-ticker.C:
			if err := m.runShipper(m.ctx); err != nil {
				log.Ctx(m.ctx).Err(err).Msg("Failed to run shipper")
				metricShipperRunFailTotal.WithLabelValues(GetErrStatusCode(err)).Inc()
			}
		}
	}
}

func (m *MetricShipper) runShipper(ctx context.Context) error {
	return m.metrics.SpanCtx(ctx, "shipper_runShipper", func(ctx context.Context, id string) (err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Ctx(ctx).Warn().Interface("panic", r).Msg("Recovered from a panic")
				err = fmt.Errorf("panic in shipper cycle: %v", r)
			}
		}()

		logger := instr.SpanLogger(ctx, id)
		logger.Debug().Msg("Running shipper cycle ...")

		// run the base request
		if err = m.ProcessFiles(ctx); err != nil {
			metricNewFilesErrorTotal.WithLabelValues(GetErrStatusCode(err)).Inc()
			return fmt.Errorf("failed to ship the metrics: %w", err) // WARNING -- THIS IS USED AS A MARKER STRING IN SMOKE TESTS
		}

		// check the disk usage
		if err = m.HandleDisk(ctx, time.Now().Add(-m.setting.Database.PurgeRules.MetricsOlderThan)); err != nil {
			metricDiskHandleErrorTotal.WithLabelValues(GetErrStatusCode(err)).Inc()
			return fmt.Errorf("failed to handle the disk usage: %w", err)
		}

		// used as a marker in tests to signify that the shipper was complete.
		// if you change this string, then change in the smoke tests as well.
		logger.Debug().Msg("Successfully ran the shipper cycle") // WARNING -- THIS IS USED AS A MARKER STRING IN SMOKE TESTS

		return err
	})
}

func (m *MetricShipper) ProcessFiles(ctx context.Context) error {
	return m.metrics.SpanCtx(ctx, "shipper_ProcessNewFiles", func(ctx context.Context, id string) error {
		logger := instr.SpanLogger(ctx, id)
		logger.Debug().Msg("Processing new files ...")

		// lock the base dir for the duration of the new file handling
		logger.Debug().Msg("Aquiring file lock")
		l := lock.NewFileLock(
			m.ctx, filepath.Join(m.GetBaseDir(), ".lock"),
			lock.WithStaleTimeout(time.Second*30), // detects stale timeout
			lock.WithRefreshInterval(time.Second*5),
			lock.WithMaxRetry(lockMaxRetry), // 5 min wait
		)
		if err := l.Acquire(); err != nil {
			return errors.Join(ErrCreateLock, fmt.Errorf("failed to acquire the lock file: %w", err))
		}
		defer func() {
			if err := l.Release(); err != nil {
				logger.Err(err).Msg("Failed to release the lock")
			}
		}()

		logger.Debug().Msg("Successfully acquired lock file")
		logger.Debug().Msg("Fetching the files from the disk store")

		// Process new files in parallel
		paths, err := m.store.GetFiles()
		if err != nil {
			return errors.Join(ErrFilesList, fmt.Errorf("failed to list the new files: %w", err))
		}

		if len(paths) == 0 {
			logger.Debug().Msg("No files found, skipping")
			return nil
		}

		logger.Debug().Int("files", len(paths)).Msg("Found files to ship")
		logger.Debug().Msg("Creating a list of metric files")

		// create a list of metric files
		files := make([]types.File, 0)
		for _, item := range paths {
			file, err := disk.NewMetricFile(item)
			if err != nil {
				return fmt.Errorf("failed to create the metric file: %w", err)
			}
			files = append(files, file)
		}

		// handle the file request
		if err := m.HandleRequest(ctx, files); err != nil {
			return err
		}

		// NOTE: used as a hook in integration tests to validate that the application worked
		logger.Debug().Int("numNewFiles", len(paths)).Msg("Successfully uploaded new files")
		metricNewFilesProcessingCurrent.WithLabelValues().Set(float64(len(paths)))
		return nil
	})
}

// HandleRequest takes in a list of files and runs them through the following:
//
// - Generate presigned URL
// - handles replay requests
// - Upload to the remote API
// - Rename the file to indicate upload
func (m *MetricShipper) HandleRequest(ctx context.Context, files []types.File) error {
	return m.metrics.SpanCtx(ctx, "shipper_handle_request", func(ctx context.Context, id string) error {
		logger := instr.SpanLogger(ctx, id)
		logger.Debug().Int("numFiles", len(files)).Msg("Handling request")
		metricHandleRequestFileCount.Observe(float64(len(files)))
		if len(files) == 0 {
			logger.Debug().Msg("there were no files in the request")
			return nil
		}

		// chunk into more reasonable sizes to mangage
		chunks := Chunk(files, filesChunkSize)
		logger.Debug().Int("chunks", len(chunks)).Msg("Processing files")

		for i, chunk := range chunks {
			logger.Debug().Int("chunk", i).Msg("Handling chunk")
			pm := parallel.New(shipperWorkerCount)
			defer pm.Close()

			// Assign pre-signed urls to each of the file references
			urlResponse, err := m.AllocatePresignedURLs(chunk)
			if err != nil {
				metricPresignedURLErrorTotal.WithLabelValues(GetErrStatusCode(err)).Inc()
				return fmt.Errorf("failed to allocate presigned URLs: %w", err)
			}

			// check if there were any objects returned
			if len(urlResponse.Allocation) == 0 && len(urlResponse.Replay) == 0 {
				logger.Debug().Msg("No files returned from the url allocation request, skipping")
				return nil
			}

			// hold references to files -> purls
			requests := make([]*UploadFileRequest, 0)

			// anytime there are issues parsing, we should send abandon requests
			abandonRequests := make([]*AbandonAPIPayloadFile, 0) // fileID: reason

			// parse the new files with their presigned urls
			for _, file := range chunk {
				if purl, exists := urlResponse.Allocation[GetRemoteFileID(file)]; exists {
					requests = append(requests, &UploadFileRequest{
						File:         file,
						PresignedURL: purl,
					})
				}
			}

			// search the file tree for the replay request files
			for k, v := range urlResponse.Replay {
				if paths, err := m.store.Find(ctx, GetRootFileID(k), ".json.br"); err != nil {
					for _, path := range paths {
						if file, err := disk.NewMetricFile(path); err == nil {
							requests = append(requests, &UploadFileRequest{
								File:         file,
								PresignedURL: v,
							})
						}
					}
				}
			}

			// process which files we did not find
			requestSet := types.NewSet[string]()
			for _, item := range requests {
				requestSet.Add(GetRemoteFileID(item.File))
			}
			for _, item := range urlResponse.Replay {
				if !requestSet.Contains(item) {
					abandonRequests = append(abandonRequests, &AbandonAPIPayloadFile{
						ReferenceID: item,
						Reason:      "failed to find this file locally",
					})
				}
			}

			// run all upload requests in parallel
			waiter := parallel.NewWaiter()
			for _, req := range requests {
				fn := func() error {
					// send the upload request
					if err := m.UploadFile(ctx, req); err != nil {
						metricFileUploadErrorTotal.WithLabelValues(GetErrStatusCode(err)).Inc()
						logger.Err(err).Str("file", req.File.UniqueID()).Msg("failed to upload the file")
						return err
					}

					// mark the file as uploaded
					if err := m.MarkFileUploaded(ctx, req.File); err != nil {
						metricMarkFileUploadedErrorTotal.WithLabelValues(GetErrStatusCode(err)).Inc()
						logger.Err(err).Str("file", req.File.UniqueID()).Msg("failed to mark file as uploaded")
						return err
					}

					atomic.AddUint64(&m.shippedFiles, 1)
					return nil
				}

				// add to queue
				pm.Run(fn, waiter)
			}
			waiter.Wait()

			// send abandon requests
			if err := m.AbandonFiles(ctx, abandonRequests); err != nil {
				logger.Err(err).Msg("failed to abandon files")
			}

			// TODO -- ignore errors in the waiter
		}

		logger.Debug().Msg("Successfully processed all of the files")
		metricHandleRequestSuccessTotal.WithLabelValues().Inc()

		return nil
	})
}

func (m *MetricShipper) GetBaseDir() string {
	return m.setting.Database.StoragePath
}

func (m *MetricShipper) GetUploadedDir() string {
	return filepath.Join(m.GetBaseDir(), UploadedSubDirectory)
}

// Shutdown gracefully stops the MetricShipper service.
func (m *MetricShipper) Shutdown() error {
	m.cancel()
	metricShutdownTotal.WithLabelValues().Inc()
	return nil
}

// GetShipperID will return a unique id for this shipper. This id is stored on the filesystem,
// and is meant to represent a relation between an uploaded file and which shipper this file came from.
// The id is not an id this instance of the shipper, but more an id of the filesystem in which the
// file came from
func (m *MetricShipper) GetShipperID() (string, error) {
	if m.shipperID == "" {
		// where the shipper id lives
		loc := filepath.Join(m.GetBaseDir(), ".shipperid")

		data, err := os.ReadFile(loc)
		if err == nil { //nolint:gocritic // impossible to do if/else statement here
			// file was read successfully
			m.shipperID = strings.TrimSpace(string(data))
		} else if os.IsNotExist(err) {
			// create the file
			file, err := os.Create(loc) //nolint:govet // err was in-fact read for what was needed
			if err != nil {
				return "", fmt.Errorf("failed to create the shipper id file: %w", err)
			}
			defer file.Close()

			podID, exists := os.LookupEnv("HOSTNAME")
			var id string
			if exists {
				// use hostname
				id = podID
			} else {
				// use uuid
				id = uuid.NewString()
			}

			// write an id to the file
			if _, err := file.WriteString(id); err != nil {
				return "", fmt.Errorf("failed to write an id to the id file: %w", err)
			}

			m.shipperID = id
		} else {
			return "", fmt.Errorf("unknown error getting the shipper id: %w", err)
		}
	}

	return m.shipperID, nil
}
