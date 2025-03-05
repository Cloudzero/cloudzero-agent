// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/cloudzero/cloudzero-insights-controller/app/config"
	"github.com/cloudzero/cloudzero-insights-controller/app/instr"
	"github.com/cloudzero/cloudzero-insights-controller/app/lock"
	"github.com/cloudzero/cloudzero-insights-controller/app/parallel"
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
	fmt.Println(m.setting.Cloudzero.SendInterval)
	ticker := time.NewTicker(m.setting.Cloudzero.SendInterval)
	defer ticker.Stop()

	log.Ctx(m.ctx).Info().Msg("Shipper service starting")

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
			if err := m.run(); err != nil {
				log.Ctx(m.ctx).Error().Err(err).Send()
			}
		}
	}
}

func (m *MetricShipper) run() error {
	log.Ctx(m.ctx).Info().Msg("Running shipper application")

	// run the base request
	if err := m.ProcessNewFiles(); err != nil {
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
}

func (m *MetricShipper) ProcessNewFiles() error {
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

	// create the files object
	files, err := NewMetricFilesFromPaths(paths)
	if err != nil {
		return fmt.Errorf("failed to create the files; %w", err)
	}
	if len(files) == 0 {
		log.Ctx(m.ctx).Info().Msg("No files found, skipping")
		return nil
	}

	// handle the file request
	if err := m.HandleRequest(files); err != nil {
		return err
	}

	// NOTE: used as a hook in integration tests to validate that the application worked
	log.Ctx(m.ctx).Info().Int("numNewFiles", len(paths)).Msg("Successfully uploaded new files")
	return nil
}

func (m *MetricShipper) ProcessReplayRequests() error {
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
		metricReplayRequestFileCount.Observe(float64(len(rr.ReferenceIDs)))

		if err := m.HandleReplayRequest(rr); err != nil {
			metricReplayRequestErrorTotal.WithLabelValues(err.Error()).Inc()
			return fmt.Errorf("failed to process replay request '%s': %w", rr.Filepath, err)
		}

		// decrease the current queue for this replay request
		metricReplayRequestCurrent.WithLabelValues().Dec()
	}

	log.Ctx(m.ctx).Info().Msg("Successfully handled all replay requests")

	return nil
}

func (m *MetricShipper) HandleReplayRequest(rr *ReplayRequest) error {
	log.Ctx(m.ctx).Debug().Str("rr", rr.Filepath).Int("numfiles", len(rr.ReferenceIDs)).Msg("Handling replay request")

	// compose loopup map of reference ids
	refIDLookup := make(map[string]any)
	for _, item := range rr.ReferenceIDs {
		refIDLookup[item] = struct{}{}
	}

	// fetch the new files that match these ids
	newFiles := make([]string, 0)
	if err := m.lister.Walk("", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// skip dir
		if info.IsDir() {
			return nil
		}

		// check for a match
		if _, exists := refIDLookup[info.Name()]; exists {
			newFiles = append(newFiles, info.Name())
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to get matching new files: %w", err)
	}
	log.Ctx(m.ctx).Debug().Int("newFiles", len(newFiles)).Send()

	// fetch the already uploadedFiles files that match these ids
	uploadedFiles := make([]string, 0)
	if err := m.lister.Walk(UploadedSubDirectory, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// skip dir
		if info.IsDir() {
			return nil
		}

		// check for a match
		if _, exists := refIDLookup[info.Name()]; exists {
			uploadedFiles = append(uploadedFiles, info.Name())
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to get matching uploaded files: %w", err)
	}
	log.Ctx(m.ctx).Debug().Int("uploadedFiles", len(uploadedFiles)).Send()

	// combine found ids into a map
	found := make(map[string]*MetricFile) // {ReferenceID: File}
	for _, item := range newFiles {
		file, err := NewMetricFile(filepath.Join(m.GetBaseDir(), filepath.Base(item)))
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}
		found[filepath.Base(item)] = file
	}
	for _, item := range uploadedFiles {
		file, err := NewMetricFile(filepath.Join(m.GetUploadedDir(), filepath.Base(item)))
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}
		found[filepath.Base(item)] = file
	}
	log.Ctx(m.ctx).Debug().Int("found", len(found)).Send()

	// compare the results and discover which files were not found
	missing := make([]string, 0)
	valid := make([]*MetricFile, 0)
	for _, item := range rr.ReferenceIDs {
		file, exists := found[filepath.Base(item)]
		if exists {
			valid = append(valid, file)
		} else {
			missing = append(missing, filepath.Base(item))
		}
	}

	log.Info().Msgf("Replay request '%s': %d/%d files found", rr.Filepath, len(valid), len(rr.ReferenceIDs))

	// send abandon requests for the non-found files
	if len(missing) > 0 {
		log.Info().Msgf("Replay request '%s': %d files missing, sending abandon request for these files", rr.Filepath, len(missing))
		if err := m.AbandonFiles(missing, "not found"); err != nil {
			metricReplayRequestAbandonFilesErrorTotal.WithLabelValues(err.Error()).Inc()
			return fmt.Errorf("failed to send the abandon file request: %w", err)
		}
	}

	// run the `HandleRequest` function for these files
	if err := m.HandleRequest(valid); err != nil {
		return fmt.Errorf("failed to upload replay request files: %w", err)
	}

	// delete the replay request
	if err := os.Remove(rr.Filepath); err != nil {
		return fmt.Errorf("failed to delete the replay request file: %w", err)
	}

	log.Ctx(m.ctx).Info().Str("rr", rr.Filepath).Msg("Successfully handled replay request")

	return nil
}

// Takes in a list of files and runs them through the following:
// - Generate presigned URL
// - Upload to the remote API
// - Rename the file to indicate upload
func (m *MetricShipper) HandleRequest(files []*MetricFile) error {
	log.Ctx(m.ctx).Info().Int("numFiles", len(files)).Msg("Handing request")
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
		files, err := m.AllocatePresignedURLs(chunk)
		if err != nil {
			return fmt.Errorf("failed to allocate presigned URLs: %w", err)
		}

		waiter := parallel.NewWaiter()
		for _, file := range files {
			fn := func() error {
				// Upload the file
				if err := m.Upload(file); err != nil {
					return fmt.Errorf("failed to upload %s: %w", file.ReferenceID, err)
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

	return nil
}

// Upload uploads the specified file to S3 using the provided presigned URL.
func (m *MetricShipper) Upload(file *MetricFile) error {
	log.Ctx(m.ctx).Debug().Str("fileName", file.ReferenceID).Msg("Uploading file")

	data, err := file.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to get the file data: %w", err)
	}

	// Create a unique context with a timeout for the upload
	ctx, cancel := context.WithTimeout(m.ctx, m.setting.Cloudzero.SendTimeout)
	defer cancel()

	// Create a new HTTP PUT request with the file as the body
	req, err := http.NewRequestWithContext(ctx, "PUT", file.PresignedURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create upload HTTP request: %w", err)
	}

	// Send the request
	resp, err := m.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("file upload HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check for successful upload
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected upload status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

func (m *MetricShipper) MarkFileUploaded(file *MetricFile) error {
	log.Ctx(m.ctx).Debug().Str("fileName", file.ReferenceID).Msg("Marking file as uploaded")

	// create the uploaded dir if needed
	uploadDir := m.GetUploadedDir()
	if err := os.MkdirAll(uploadDir, filePermissions); err != nil {
		return fmt.Errorf("failed to create the upload directory: %w", err)
	}

	// if the filepath already contains the uploaded location,
	// then ignore this entry
	if strings.Contains(file.Filepath(), UploadedSubDirectory) {
		return nil
	}

	// compose the new path
	new := filepath.Join(uploadDir, file.Filename())

	// rename the file (IS ATOMIC)
	if err := os.Rename(file.location, new); err != nil {
		return fmt.Errorf("failed to move the file to the uploaded directory: %s", err)
	}

	return nil
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
	return nil
}
