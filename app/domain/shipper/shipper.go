// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/cloudzero/cloudzero-insights-controller/app/config"
	"github.com/cloudzero/cloudzero-insights-controller/app/instr"
	"github.com/cloudzero/cloudzero-insights-controller/app/parallel"
	"github.com/cloudzero/cloudzero-insights-controller/app/types"
	"github.com/rs/zerolog/log"
)

const (
	shipperWorkerCount = 10
	expirationTime     = 3600
)

var (
	ErrUnauthorized = errors.New("unauthorized request - possible invalid API key")
	ErrNoURLs       = errors.New("no presigned URLs returned")
)

// MetricShipper handles the periodic shipping of metrics to Cloudzero.
type MetricShipper struct {
	setting *config.Settings
	lister  types.AppendableFiles

	// Internal fields
	ctx          context.Context
	cancel       context.CancelFunc
	HTTPClient   *http.Client
	shippedFiles uint64 // Counter for shipped files
	metrics      *instr.PrometheusMetrics
}

// NewMetricShipper initializes a new MetricShipper.
func NewMetricShipper(ctx context.Context, s *config.Settings, f types.AppendableFiles) (*MetricShipper, error) {
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
			if err := m.ProcessReplayRequest(); err != nil {
				return fmt.Errorf("failed to run the replay request: %w", err)
			}
		}
	}
}

func (m *MetricShipper) ProcessNewFiles() error {
	pm := parallel.New(shipperWorkerCount)
	defer pm.Close()

	// Process new files in parallel
	paths, err := m.lister.GetFiles()
	if err != nil {
		return fmt.Errorf("failed to get shippable files: %w", err)
	}

	// create the files object
	files, err := NewFilesFromPaths(paths) // TODO -- replace with builder
	if err != nil {
		return fmt.Errorf("failed to create the files; %w", err)
	}
	if len(files) == 0 {
		return nil
	}

	// handle the file request
	return m.HandleRequest(files)
}

func (m *MetricShipper) ProcessReplayRequest() error {
	// TODO

	// read the reference ids from the file

	// write into a list of files

	// run the `HandleRequest` function

	// read the replay request
	return errors.New("UNIMPLEMENTED")
}

// Takes in a list of files and runs them through the following:
// - Generate presigned URL
// - Upload to the remote API
// - Rename the file to indicate upload
func (m *MetricShipper) HandleRequest(files []*File) error {
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

	return nil
}

// Shutdown gracefully stops the MetricShipper service.
func (m *MetricShipper) Shutdown() error {
	m.cancel()
	return nil
}
