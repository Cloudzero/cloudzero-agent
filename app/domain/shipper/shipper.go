// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"time"

	config "github.com/cloudzero/cloudzero-agent-validator/app/config/gator"
	"github.com/cloudzero/cloudzero-agent-validator/app/instr"
	"github.com/cloudzero/cloudzero-agent-validator/app/lock"
	"github.com/cloudzero/cloudzero-agent-validator/app/parallel"
	"github.com/cloudzero/cloudzero-agent-validator/app/store"
	"github.com/cloudzero/cloudzero-agent-validator/app/types"
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
	HTTPClient   *http.Client
	shippedFiles uint64 // Counter for shipped files
	metrics      *instr.PrometheusMetrics
	shipperID    string // unique id for the shipper
}

// NewMetricShipper initializes a new MetricShipper.
func NewMetricShipper(ctx context.Context, s *config.Settings, store types.ReadableStore) (*MetricShipper, error) {
	ctx, cancel := context.WithCancel(ctx)

	// Initialize an HTTP client with the specified timeout
	httpClient := &http.Client{
		Timeout: s.Cloudzero.SendTimeout,
	}

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
	if err := os.Mkdir(m.GetUploadedDir(), filePermissions); err != nil {
		log.Ctx(m.ctx).Err(err).Msg("failed to create the uploaded directory")
		return ErrCreateDirectory
	}
	if err := os.Mkdir(m.GetReplayRequestDir(), filePermissions); err != nil {
		log.Ctx(m.ctx).Err(err).Msg("failed to create the replay request directory")
		return ErrCreateDirectory
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
		return fmt.Errorf("failed to run the shipper: %w", err)
	}

	for {
		select {
		case <-m.ctx.Done():
			log.Ctx(m.ctx).Info().Msg("Shipper service stopping")
			return nil

		case sig := <-sigChan:
			log.Ctx(m.ctx).Info().Str("signal", sig.String()).Msg("Received signal. Initiating shutdown.")

			// flush
			if err := m.ProcessNewFiles(m.ctx); err != nil {
				metricNewFilesErrorTotal.WithLabelValues(GetErrStatusCode(err)).Inc()
				return fmt.Errorf("failed to ship the metrics: %w", err)
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
	return m.metrics.SpanCtx(ctx, "shipper_runShipper", func(ctx context.Context, id string) error {
		logger := instr.SpanLogger(ctx, id)
		logger.Debug().Msg("Running shipper cycle ...")

		// run the base request
		if err := m.ProcessNewFiles(ctx); err != nil {
			metricNewFilesErrorTotal.WithLabelValues(GetErrStatusCode(err)).Inc()
			return fmt.Errorf("failed to ship the metrics: %w", err)
		}

		// run the replay request
		if err := m.ProcessReplayRequests(ctx); err != nil {
			metricReplayRequestErrorTotal.WithLabelValues(GetErrStatusCode(err)).Inc()
			return fmt.Errorf("failed to process the replay requests: %w", err)
		}

		// check the disk usage
		if err := m.HandleDisk(ctx, time.Now().Add(-m.setting.Database.PurgeRules.MetricsOlderThan)); err != nil {
			metricDiskHandleErrorTotal.WithLabelValues(GetErrStatusCode(err)).Inc()
			return fmt.Errorf("failed to handle the disk usage: %w", err)
		}

		// used as a marker in tests to signify that the shipper was complete.
		// if you change this string, then change in the smoke tests as well.
		logger.Debug().Msg("Successfully ran the shipper cycle")

		return nil
	})
}

func (m *MetricShipper) ProcessNewFiles(ctx context.Context) error {
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
			logger.Err(err).Msg("failed to acquire the lock file")
			return ErrCreateLock
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
			logger.Err(err).Msg("failed to list the new files")
			return ErrFilesList
		}

		if len(paths) == 0 {
			logger.Debug().Msg("No files found to ship")
			return nil
		}

		logger.Debug().Int("files", len(paths)).Msg("Found files to ship")
		logger.Debug().Msg("Creating a list of metric files")

		// create a list of metric files
		files := make([]types.File, 0)
		for _, item := range paths {
			file, err := store.NewMetricFile(item)
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
			urlMap, err := m.AllocatePresignedURLs(chunk)
			if err != nil {
				metricPresignedURLErrorTotal.WithLabelValues(GetErrStatusCode(err)).Inc()
				return fmt.Errorf("failed to allocate presigned URLs: %w", err)
			}

			waiter := parallel.NewWaiter()
			for _, file := range chunk {
				fn := func() error {
					// Upload the file
					if err := m.UploadFile(ctx, file, urlMap[GetRemoteFileID(file)]); err != nil {
						metricFileUploadErrorTotal.WithLabelValues(GetErrStatusCode(err)).Inc()
						return fmt.Errorf("failed to upload %s: %w", file.UniqueID(), err)
					}

					// mark the file as uploaded
					if err := m.MarkFileUploaded(ctx, file); err != nil {
						metricMarkFileUploadedErrorTotal.WithLabelValues(GetErrStatusCode(err)).Inc()
						return fmt.Errorf("failed to mark the file as uploaded: %w", err)
					}

					atomic.AddUint64(&m.shippedFiles, 1)
					return nil
				}
				pm.Run(fn, waiter)
			}
			waiter.Wait()

			// check for errors in the waiter
			for err := range waiter.Err() {
				if err != nil {
					return fmt.Errorf("failed to upload files: %w", err)
				}
			}
		}

		logger.Debug().Msg("Successfully processed all of the files")
		metricHandleRequestSuccessTotal.WithLabelValues().Inc()

		return nil
	})
}

func (m *MetricShipper) GetBaseDir() string {
	return m.setting.Database.StoragePath
}

func (m *MetricShipper) GetReplayRequestDir() string {
	return filepath.Join(m.GetBaseDir(), ReplaySubDirectory)
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
