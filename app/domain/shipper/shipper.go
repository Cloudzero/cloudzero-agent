// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/cloudzero/cloudzero-insights-controller/app/config"
	"github.com/cloudzero/cloudzero-insights-controller/app/instr"
	"github.com/cloudzero/cloudzero-insights-controller/app/lock"
	"github.com/cloudzero/cloudzero-insights-controller/app/parallel"
	"github.com/cloudzero/cloudzero-insights-controller/app/store"
	"github.com/cloudzero/cloudzero-insights-controller/app/types"
	"github.com/rs/zerolog/log"
)

// MetricShipper handles the periodic shipping of metrics to Cloudzero.
type MetricShipper struct {
	setting *config.Settings
	lister  types.AppendableFilesMonitor

	// Internal fields
	ctx          context.Context
	cancel       context.CancelFunc
	HTTPClient   *http.Client
	shippedFiles uint64 // Counter for shipped files
	metrics      *instr.PrometheusMetrics
}

// NewMetricShipper initializes a new MetricShipper.
func NewMetricShipper(ctx context.Context, s *config.Settings, f types.AppendableFilesMonitor) (*MetricShipper, error) {
	ctx, cancel := context.WithCancel(ctx)

	// Initialize an HTTP client with the specified timeout
	httpClient := &http.Client{
		Timeout: s.Cloudzero.SendTimeout,
	}

	// create the metrics listener
	metrics, err := InitMetrics()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to init metrics: %s", err)
	}

	return &MetricShipper{
		setting:    s,
		lister:     f,
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
		return fmt.Errorf("failed to create the uploaded directory: %w", err)
	}
	if err := os.Mkdir(m.GetReplayRequestDir(), filePermissions); err != nil {
		return fmt.Errorf("failed to create the replay request directory: %w", err)
	}

	// Set up channel to listen for OS signals
	sigChan := make(chan os.Signal, 1)
	// Listen for interrupt and termination signals
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	// Initialize ticker for periodic shipping
	ticker := time.NewTicker(m.setting.Cloudzero.SendInterval)
	defer ticker.Stop()

	log.Ctx(m.ctx).Info().Msg("Shipper service starting")

	// run at the start
	if err := m.runShipper(); err != nil {
		log.Ctx(m.ctx).Error().Err(err).Send()
	}

	for {
		select {
		case <-m.ctx.Done():
			log.Ctx(m.ctx).Info().Msg("Shipper service stopping")
			return nil

		case sig := <-sigChan:
			log.Ctx(m.ctx).Info().Msgf("Received signal %s. Initiating shutdown.", sig)
			err := m.Shutdown()
			if err != nil {
				log.Ctx(m.ctx).Error().Err(err).Msg("Failed to shutdown shipper service")
			}
			return nil

		case <-ticker.C:
			if err := m.runShipper(); err != nil {
				log.Ctx(m.ctx).Error().Err(err).Send()
			}
		}
	}
}

func (m *MetricShipper) runShipper() error {
	return m.metrics.Span("shipper_runShipper", func() error {
		log.Ctx(m.ctx).Info().Msg("Running shipper application")

		// run the base request
		if err := m.ProcessNewFiles(); err != nil {
			metricNewFilesErrorTotal.WithLabelValues(err.Error()).Inc()
			return fmt.Errorf("failed to ship the metrics: %w", err)
		}

		// run the replay request
		if err := m.ProcessReplayRequests(); err != nil {
			return fmt.Errorf("failed to process the replay requests: %w", err)
		}

		// check the disk usage
		if err := m.HandleDisk(time.Now().AddDate(0, 0, -int(m.setting.Database.PurgeMetricsOlderThanDay))); err != nil {
			return fmt.Errorf("failed to handle the disk usage: %w", err)
		}

		// used as a marker in tests to signify that the shipper was complete.
		// if you change this string, then change in the smoke tests as well.
		log.Ctx(m.ctx).Info().Msg("Successfully ran the shipper application")

		return nil
	})
}

func (m *MetricShipper) ProcessNewFiles() error {
	return m.metrics.Span("shipper_ProcessNewFiles", func() error {
		log.Ctx(m.ctx).Info().Msg("Processing new files ...")

		// lock the base dir for the duration of the new file handling
		log.Ctx(m.ctx).Debug().Msg("Aquiring file lock")
		l := lock.NewFileLock(
			m.ctx, filepath.Join(m.GetBaseDir(), ".lock"),
			lock.WithStaleTimeout(time.Second*30), // detects stale timeout
			lock.WithRefreshInterval(time.Second*5),
			lock.WithMaxRetry(lockMaxRetry), // 5 min wait
		)
		if err := l.Acquire(); err != nil {
			return fmt.Errorf("failed to acquire the lock: %w", err)
		}
		defer func() {
			if err := l.Release(); err != nil {
				log.Ctx(m.ctx).Error().Err(err).Msg("Failed to release the lock")
			}
		}()

		// Process new files in parallel
		paths, err := m.lister.GetFiles()
		if err != nil {
			return fmt.Errorf("failed to get shippable files: %w", err)
		}
		log.Ctx(m.ctx).Debug().Int("numFiles", len(paths)).Send()

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
		if err := m.HandleRequest(files); err != nil {
			return err
		}

		// NOTE: used as a hook in integration tests to validate that the application worked
		log.Ctx(m.ctx).Info().Int("numNewFiles", len(paths)).Msg("Successfully uploaded new files")
		metricNewFilesProcessingCurrent.WithLabelValues().Set(float64(len(paths)))
		return nil
	})
}

// Takes in a list of files and runs them through the following:
// - Generate presigned URL
// - Upload to the remote API
// - Rename the file to indicate upload
func (m *MetricShipper) HandleRequest(files []types.File) error {
	return m.metrics.Span("shipper_handle_request", func() error {
		log.Ctx(m.ctx).Info().Int("numFiles", len(files)).Msg("Handing request")
		metricHandleRequestFileCount.Observe(float64(len(files)))
		if len(files) == 0 {
			return nil
		}

		// chunk into more reasonable sizes to mangage
		chunks := Chunk(files, filesChunkSize)
		log.Ctx(m.ctx).Info().Msgf("processing files as %d chunks", len(chunks))

		for i, chunk := range chunks {
			log.Ctx(m.ctx).Debug().Msgf("handling chunk: %d", i)
			pm := parallel.New(shipperWorkerCount)
			defer pm.Close()

			// Assign pre-signed urls to each of the file references
			urlMap, err := m.AllocatePresignedURLs(chunk)
			if err != nil {
				metricPresignedURLErrorTotal.WithLabelValues(err.Error()).Inc()
				return fmt.Errorf("failed to allocate presigned URLs: %w", err)
			}

			waiter := parallel.NewWaiter()
			for _, file := range chunk {
				fn := func() error {
					// Upload the file
					if err := m.UploadFile(file, urlMap[GetRemoteFileID(file)]); err != nil {
						return fmt.Errorf("failed to upload %s: %w", file.UniqueID(), err)
					}

					// mark the file as uploaded
					if err := m.MarkFileUploaded(file); err != nil {
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
					return fmt.Errorf("failed to upload files; %w", err)
				}
			}
		}

		log.Ctx(m.ctx).Info().Msg("Successfully processed all of the files")
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
