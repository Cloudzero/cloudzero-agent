// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
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

const (
	shipperWorkerCount = 10
	expirationTime     = 3600
	replaySubdirName   = "replay"
	filePermissions    = 0o755
	lockMaxRetry       = 60
)

var (
	ErrUnauthorized = errors.New("unauthorized request - possible invalid API key")
	ErrNoURLs       = errors.New("no presigned URLs returned")
)

// MetricShipper handles the periodic shipping of metrics to Cloudzero.
type MetricShipper struct {
	setting *config.Settings
	lister  types.AppendableDisk

	// Internal fields
	ctx          context.Context
	cancel       context.CancelFunc
	HTTPClient   *http.Client
	shippedFiles uint64 // Counter for shipped files
	metrics      *instr.PrometheusMetrics
}

// NewMetricShipper initializes a new MetricShipper.
func NewMetricShipper(ctx context.Context, s *config.Settings, f types.AppendableDisk) (*MetricShipper, error) {
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
	// Set up channel to listen for OS signals
	sigChan := make(chan os.Signal, 1)
	// Listen for interrupt and termination signals
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	// Initialize ticker for periodic shipping
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
			// run the base request
			if err := m.ProcessNewFiles(); err != nil {
				log.Ctx(m.ctx).Error().Err(err).Msg("Failed to ship metrics")
			}

			// run the replay request
			if err := m.ProcessReplayRequests(); err != nil {
				log.Ctx(m.ctx).Error().Err(err).Msg("Failed to process replay requests")
			}

			// check the disk usage
			if err := m.HandleDisk(); err != nil {
				log.Ctx(m.ctx).Error().Err(err).Msg("Failed to handle the disk usage")
			}
		}
	}
}

func (m *MetricShipper) ProcessNewFiles() error {
	// ensure the directory is created
	if err := os.MkdirAll(m.GetBaseDir(), filePermissions); err != nil {
		return fmt.Errorf("failed to create the base file directory: %w", err)
	}

	// lock the base dir for the duration of the new file handling
	l := lock.NewFileLock(
		m.ctx, filepath.Join(m.GetBaseDir(), ".lock"),
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

	pm := parallel.New(shipperWorkerCount)
	defer pm.Close()

	// Process new files in parallel
	paths, err := m.lister.GetFiles()
	if err != nil {
		return fmt.Errorf("failed to get shippable files: %w", err)
	}

	// create the files object
	files, err := NewMetricFilesFromPaths(paths) // TODO -- replace with builder
	if err != nil {
		return fmt.Errorf("failed to create the files; %w", err)
	}
	if len(files) == 0 {
		return nil
	}

	// handle the file request
	return m.HandleRequest(files)
}

func (m *MetricShipper) ProcessReplayRequests() error {
	// ensure the directory is created
	if err := os.MkdirAll(m.GetReplayRequestDir(), filePermissions); err != nil {
		return fmt.Errorf("failed to create the replay request file directory: %w", err)
	}

	// lock the replay request dir for the duration of the replay request processing
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

	// handle all valid replay requests
	for _, rr := range requests {
		if err := m.HandleReplayRequest(rr); err != nil {
			return fmt.Errorf("failed to process replay request '%s': %w", rr.Filepath, err)
		}
	}

	return nil
}

func (m *MetricShipper) HandleReplayRequest(rr *ReplayRequest) error {
	// fetch the new files that match these ids
	new, err := m.lister.GetMatching("", rr.ReferenceIDs)
	if err != nil {
		return fmt.Errorf("failed to get matching new files: %w", err)
	}

	// fetch the already uploaded files that match these ids
	uploaded, err := m.lister.GetMatching(m.setting.Database.StorageUploadSubpath, rr.ReferenceIDs)
	if err != nil {
		return fmt.Errorf("failed to get matching uploaded files: %w", err)
	}

	// combine found ids into a map
	found := make(map[string]*MetricFile) // {ReferenceID: File}
	for _, item := range new {
		file, err := NewMetricFile(filepath.Join(m.setting.Database.StoragePath, filepath.Base(item)))
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}
		found[filepath.Base(item)] = file
	}
	for _, item := range uploaded {
		file, err := NewMetricFile(filepath.Join(m.GetUploadedDir(), filepath.Base(item)))
		if err != nil {
			return fmt.Errorf("failed to create file: %w", err)
		}
		found[filepath.Base(item)] = file
	}

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

	return nil
}

// Takes in a list of files and runs them through the following:
// - Generate presigned URL
// - Upload to the remote API
// - Rename the file to indicate upload
func (m *MetricShipper) HandleRequest(files []*MetricFile) error {
	pm := parallel.New(shipperWorkerCount)
	defer pm.Close()

	// Assign pre-signed urls to each of the file references
	files, err := m.AllocatePresignedURLs(files)
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

	return nil
}

// Upload uploads the specified file to S3 using the provided presigned URL.
func (m *MetricShipper) Upload(file *MetricFile) error {
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
	// create the uploaded dir if needed
	uploadDir := m.GetUploadedDir()
	if err := os.MkdirAll(uploadDir, filePermissions); err != nil {
		return fmt.Errorf("failed to create the upload directory: %w", err)
	}

	// if the filepath already contains the uploaded location,
	// then ignore this entry
	if strings.Contains(file.Filepath(), m.setting.Database.StorageUploadSubpath) {
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
	return filepath.Join(m.GetBaseDir(), replaySubdirName)
}

func (m *MetricShipper) GetUploadedDir() string {
	return filepath.Join(m.GetBaseDir(), m.setting.Database.StorageUploadSubpath)
}

// Shutdown gracefully stops the MetricShipper service.
func (m *MetricShipper) Shutdown() error {
	m.cancel()
	return nil
}
