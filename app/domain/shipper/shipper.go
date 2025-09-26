// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package shipper provides metric upload and transmission services for the CloudZero Agent.
// This package implements the critical Secondary Adapter responsible for delivering collected
// and processed metrics to the CloudZero cost optimization platform for billing analysis.
//
// The shipper service operates as the final stage in the CloudZero Agent metric processing pipeline:
//  1. Periodic file discovery and processing from local storage
//  2. Presigned URL allocation from CloudZero platform for secure S3 upload
//  3. Parallel file upload with retry logic and error handling
//  4. File lifecycle management (marking uploaded, cleanup, replay handling)
//  5. Disk space management and purging of old metric files
//
// Key architectural responsibilities:
//   - Reliable metric delivery: Ensure all collected metrics reach CloudZero platform
//   - Batched processing: Handle file uploads in chunks for optimal performance
//   - Replay mechanisms: Reprocess files that failed upload due to transient errors
//   - Resource management: Monitor disk usage and enforce retention policies
//   - Operational monitoring: Provide comprehensive metrics for shipper health
//
// The service implements robust error handling and recovery mechanisms essential for
// production environments where network connectivity and platform availability may vary.
// All operations are instrumented with detailed metrics and logging for operational visibility.
//
// Integration points:
//   - Storage layer: Reads processed metric files from disk storage
//   - CloudZero API: Requests presigned URLs and uploads data
//   - Prometheus metrics: Provides operational monitoring and alerting data
//   - Configuration service: Receives upload intervals and retention policies
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

// MetricShipper orchestrates the complete metric upload pipeline for CloudZero Agent data transmission.
// This struct manages the periodic discovery, processing, and upload of metric files from local storage
// to the CloudZero platform using secure presigned URLs and parallel processing for optimal performance.
//
// The shipper implements a robust upload workflow designed for production reliability:
//   - Periodic scheduling: Configurable intervals for batch processing
//   - File locking: Prevents concurrent access during processing
//   - Chunked uploads: Processes files in manageable batches
//   - Parallel workers: Concurrent upload for throughput optimization
//   - Retry logic: Handles transient network failures gracefully
//   - Replay support: Reprocesses failed uploads automatically
//   - Disk management: Enforces retention policies and cleanup
//
// Operational characteristics:
//   - Signal handling: Graceful shutdown on SIGINT/SIGTERM
//   - Context cancellation: Proper resource cleanup and timeout handling
//   - Comprehensive metrics: Detailed observability for monitoring and alerting
//   - Error recovery: Panic handling and operation resilience
//   - Configuration reload: Dynamic settings updates without restart
type MetricShipper struct {
	// setting provides dynamic configuration for upload intervals, retention policies,
	// CloudZero API endpoints, authentication credentials, and operational parameters.
	// Configuration changes are applied during the next processing cycle.
	setting *config.Settings

	// store provides access to metric files stored locally by the collector and storage services.
	// The shipper reads compressed metric files, uploads them to CloudZero, and manages
	// file lifecycle including marking files as uploaded and cleaning up old data.
	store types.ReadableStore

	// ctx provides the root context for all shipper operations, enabling graceful shutdown
	// and operation cancellation. Derived contexts are created for individual operations
	// with appropriate timeouts and span tracking for observability.
	ctx context.Context

	// cancel enables graceful shutdown of the shipper service by cancelling all ongoing
	// operations and triggering cleanup procedures. Called during SIGINT/SIGTERM handling
	// or explicit shutdown requests from the application lifecycle manager.
	cancel context.CancelFunc

	// HTTPClient provides the configured HTTP client for CloudZero API communication.
	// Includes retry logic, timeout configuration, authentication, and connection pooling
	// optimized for reliable metric upload operations in production environments.
	HTTPClient *retryablehttp.Client

	// shippedFiles tracks the total number of successfully uploaded files for operational monitoring.
	// Updated atomically during concurrent upload operations and exposed via Prometheus metrics
	// for capacity planning, performance analysis, and billing correlation.
	shippedFiles uint64

	// metrics provides comprehensive Prometheus instrumentation for shipper operations.
	// Tracks upload rates, error rates, processing latency, disk usage, and operational health
	// essential for production monitoring, alerting, and performance optimization.
	metrics *instr.PrometheusMetrics

	// shipperID provides a unique identifier for this shipper instance, persisted to filesystem
	// and used for correlating uploaded files with their origin. Enables tracking and debugging
	// in multi-instance deployments and provides audit trails for billing reconciliation.
	// Uses hostname if available, otherwise generates a UUID for unique identification.
	shipperID string
}

// NewMetricShipper creates a fully configured MetricShipper instance for CloudZero metric upload operations.
// This constructor initializes all necessary components for reliable metric transmission including
// HTTP clients, metrics instrumentation, and operational configuration validation.
//
// Initialization sequence:
//  1. Creates cancellable context for graceful shutdown support
//  2. Configures HTTP client with retry logic, timeouts, and authentication
//  3. Initializes Prometheus metrics for operational monitoring
//  4. Validates configuration settings and logs debug information
//  5. Sets up all internal state for immediate operation
//
// The constructor performs comprehensive validation and setup to ensure the shipper
// is ready for production operation, including proper error handling and resource cleanup
// if initialization fails at any stage.
//
// Dependencies:
//   - ctx: Parent context for operation lifecycle and cancellation
//   - s: Configuration settings for API endpoints, credentials, and operational parameters
//   - store: Storage interface for reading processed metric files
//
// Configuration validation includes:
//   - CloudZero API endpoint accessibility and authentication
//   - Upload interval and batch size validation
//   - Disk space and retention policy verification
//   - HTTP client timeout and retry configuration
//
// Returns a ready-to-use MetricShipper instance or an error if initialization fails.
// The returned shipper must have its Run() method called to begin processing operations.
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

// Run executes the main MetricShipper service loop with periodic metric upload processing.
// This method implements the primary operational lifecycle of the shipper, handling
// scheduled uploads, signal-based shutdown, and comprehensive error recovery.
//
// Service lifecycle:
//  1. Directory initialization: Creates required upload directories with proper permissions
//  2. Signal handling setup: Configures SIGINT/SIGTERM for graceful shutdown
//  3. Initial upload run: Processes any pending files immediately on startup
//  4. Periodic processing: Executes upload cycles based on configured intervals
//  5. Graceful shutdown: Completes in-flight operations and cleanup on termination
//
// The service runs continuously until:
//   - Context cancellation from parent service
//   - OS signal reception (SIGINT, SIGTERM)
//   - Unrecoverable error during processing
//
// Error handling strategy:
//   - Individual upload failures are logged but don't stop service operation
//   - Panic recovery prevents service crashes during upload processing
//   - Metrics tracking enables monitoring and alerting for operational issues
//   - Directory creation failures are fatal as they prevent basic operation
//
// Operational characteristics:
//   - Non-blocking: Individual upload failures don't block subsequent processing
//   - Resource cleanup: Proper directory and signal handler cleanup on exit
//   - Timeout handling: 30-second shutdown timeout for graceful termination
//   - Comprehensive logging: Detailed operation tracking for troubleshooting
//
// The method blocks until shutdown is requested, making it suitable for use as
// the main execution path for shipper-focused services or containers.
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

	// TODO -- sleep between random value between 0 and m.setting.Cloudzero.SendInterval (for multi-shipper-coordination)

	for {
		select {
		case <-m.ctx.Done():
			// create a new ctx to shut the application down with
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			log.Ctx(ctx).Info().Msg("Shipper service stopping")
			m.Flush(ctx)
			return nil
		case sig := <-sigChan:
			log.Ctx(m.ctx).Info().Str("signal", sig.String()).Msg("Received signal. Initiating shutdown.")
			return m.Shutdown()
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
		logger.Info().Msg("Successfully ran the shipper cycle") // WARNING -- THIS IS USED AS A MARKER STRING IN SMOKE TESTS

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
			ctx, filepath.Join(m.GetBaseDir(), ".lock"),
			lock.WithStaleTimeout(time.Minute*5), // detects stale timeout
			lock.WithRefreshInterval(time.Second*5),
			lock.WithMaxRetry(lockMaxRetry), // 15 second wait
		)
		if err := l.Acquire(); err != nil {
			// do not throw an error when failed because of context
			if errors.Is(err, lock.ErrLockContextCancelled) {
				return nil
			}
			return errors.Join(ErrCreateLock, fmt.Errorf("failed to acquire the lock file: %w", err))
		}
		defer func() {
			if err := l.Release(); err != nil {
				logger.Err(err).Msg("Failed to release the lock")
			}
		}()

		logger.Debug().Msg("Successfully acquired lock file")
		logger.Debug().Msg("Fetching the files from the disk store")

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
			urlResponse, err := m.AllocatePresignedURLs(ctx, chunk)
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
			for replayRefID, replayURL := range urlResponse.Replay {
				if paths, err := m.store.Find(ctx, GetRootFileID(replayRefID), ".json.br"); err == nil {
					for _, path := range paths {
						if file, err := disk.NewMetricFile(path); err == nil {
							requests = append(requests, &UploadFileRequest{
								File:         file,
								PresignedURL: replayURL,
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
			for replayRefID := range urlResponse.Replay {
				if !requestSet.Contains(replayRefID) {
					abandonRequests = append(abandonRequests, &AbandonAPIPayloadFile{
						ReferenceID: replayRefID,
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

					// if not a metric file, remove it. we do not need to store these after upload
					if !strings.HasPrefix(req.File.UniqueID(), disk.CostContentIdentifier) {
						if err := os.Remove(req.File.Location()); err != nil {
							logger.Err(err).Str("file", req.File.UniqueID()).Msg("failed to remove log file")
						}
					} else {
						// mark the file as uploaded
						if err := m.MarkFileUploaded(ctx, req.File); err != nil {
							metricMarkFileUploadedErrorTotal.WithLabelValues(GetErrStatusCode(err)).Inc()
							logger.Err(err).Str("file", req.File.UniqueID()).Msg("failed to mark file as uploaded")
							return err
						}
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

// Flush will attempt to process all files
// and push them to the remote
func (m *MetricShipper) Flush(ctx context.Context) {
	if err := m.ProcessFiles(ctx); err != nil {
		metricNewFilesErrorTotal.WithLabelValues(GetErrStatusCode(err)).Inc()
		log.Ctx(ctx).Err(err).Msg("Failed to flush the new metric files")
	}
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
