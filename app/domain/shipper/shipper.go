// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package shipper implements the CloudZero agent's metric shipping functionality.
//
// This package is responsible for periodically uploading stored metrics to CloudZero's
// servers via presigned URLs. It handles the complete upload workflow including:
//
//   - File discovery and batching from local storage
//   - Presigned URL allocation from CloudZero API
//   - Parallel file uploads to cloud storage (S3/equivalent)
//   - File lifecycle management (marking as uploaded, cleanup)
//   - Error handling and retry logic for failed uploads
//   - Disk space management and old file purging
//
// Architecture:
//   - MetricShipper: Main service that orchestrates the shipping process
//   - Periodic execution: Runs on configurable intervals (typically every few minutes)
//   - File locking: Prevents concurrent shipping operations
//   - Parallel processing: Uses worker pools for efficient bulk uploads
//   - Monitoring: Exposes Prometheus metrics for observability
//
// Key workflows:
//   1. Periodic timer triggers shipping cycle
//   2. Lock acquisition prevents concurrent operations
//   3. File discovery finds new metrics to upload
//   4. Presigned URL allocation from CloudZero API
//   5. Parallel upload execution with retry logic
//   6. File marking and cleanup after successful upload
//   7. Disk usage monitoring and purging of old files
//
// Integration points:
//   - types.ReadableStore: Interface to local metric storage
//   - config.Settings: Shipping intervals, CloudZero endpoints, credentials
//   - HTTP client: Configurable timeout and retry settings for API calls
//   - File system: Direct file operations for upload and cleanup
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

// MetricShipper handles the periodic shipping of stored metrics to CloudZero servers.
// It runs as a long-lived service that continuously processes local metric files,
// uploads them to cloud storage via presigned URLs, and manages file lifecycle.
//
// The shipper implements several key patterns:
//   - Periodic execution: Runs shipping cycles at configurable intervals
//   - File locking: Prevents concurrent operations that could corrupt data
//   - Parallel processing: Uses worker pools to upload multiple files simultaneously
//   - Retry logic: Handles transient failures with exponential backoff
//   - Resource management: Monitors disk usage and purges old files
//   - Graceful shutdown: Responds to OS signals and context cancellation
//
// Lifecycle:
//   1. NewMetricShipper() creates and configures the shipper
//   2. Run() starts the main service loop with signal handling
//   3. Periodic shipping cycles process files automatically
//   4. Shutdown() or signal reception triggers graceful stop
//
// Error handling:
//   - Individual file upload failures don't stop the shipping cycle
//   - Persistent failures trigger abandon requests to CloudZero API
//   - Critical errors (lock failures, storage issues) abort the cycle
//   - All errors are logged and exposed via Prometheus metrics
type MetricShipper struct {
	// setting contains the agent configuration including CloudZero endpoints,
	// shipping intervals, storage paths, and authentication credentials
	setting *config.Settings
	
	// store provides read access to the local metric storage system,
	// typically a disk-based implementation that persists JSON metric files
	store types.ReadableStore

	// Internal service management fields
	
	// ctx is the main context for the shipper service lifecycle
	ctx context.Context
	
	// cancel function stops the shipper service and all background operations
	cancel context.CancelFunc
	
	// HTTPClient handles all outbound HTTP requests with configured timeouts,
	// retry policies, and authentication for CloudZero API calls
	HTTPClient *retryablehttp.Client
	
	// shippedFiles tracks the total number of files successfully uploaded,
	// used for monitoring and debugging shipping performance
	shippedFiles uint64
	
	// metrics provides Prometheus monitoring for shipping operations,
	// tracking success rates, error counts, and performance statistics
	metrics *instr.PrometheusMetrics
	
	// shipperID is a unique identifier for this shipper instance, persisted
	// to disk to maintain consistency across restarts and correlate uploaded files
	shipperID string
}

// NewMetricShipper creates and initializes a new MetricShipper instance with all
// required dependencies and configurations. The shipper is ready to use immediately
// after creation, but Run() must be called to start the periodic shipping service.
//
// Parameters:
//   - ctx: Parent context for the shipper lifecycle, cancellation stops all operations
//   - s: Configuration settings including CloudZero endpoints, intervals, and credentials
//   - store: Local storage interface for reading metric files to upload
//
// Returns:
//   - *MetricShipper: Configured shipper instance ready to run
//   - error: Configuration or initialization error
//
// The constructor performs several setup operations:
//   - Creates isolated context for service lifecycle management
//   - Initializes HTTP client with retry policies and timeouts
//   - Sets up Prometheus metrics for monitoring shipping operations
//   - Validates configuration and logs settings in debug mode
//
// Example:
//   shipper, err := NewMetricShipper(ctx, settings, diskStore)
//   if err != nil {
//       return fmt.Errorf("failed to create shipper: %w", err)
//   }
//   defer shipper.Shutdown()
//   
//   // Start the shipping service (blocks until context cancelled)
//   if err := shipper.Run(); err != nil {
//       log.Printf("shipper stopped: %v", err)
//   }
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
