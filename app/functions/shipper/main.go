// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package main implements the CloudZero metric shipper service.
//
// The shipper is responsible for uploading stored metrics from local disk
// to CloudZero servers via presigned URLs. It coordinates with the collector
// component to ensure orderly shutdown and complete data transfer.
//
// Key responsibilities:
//   - Periodic upload of stored metric files to CloudZero cloud storage
//   - Coordination with collector for graceful shutdown sequencing
//   - Secret monitoring for dynamic API key updates
//   - File lifecycle management (upload, marking, cleanup)
//   - HTTP server for metrics exposition and health checking
//   - Optional performance profiling endpoints
//
// Service lifecycle:
//   1. Configuration loading and validation from YAML files
//   2. Logger initialization with field filtering for clean output
//   3. Disk storage interface setup for reading metric files
//   4. Secret monitor startup for dynamic credential management
//   5. MetricShipper domain service creation and background startup
//   6. HTTP server initialization for monitoring and profiling
//   7. Signal handling with collector coordination for graceful shutdown
//
// Command-line usage:
//   shipper -config /path/to/config.yaml
//
// Shutdown coordination:
//   The shipper waits for the collector's shutdown marker file before
//   proceeding with its own shutdown, ensuring no metrics are lost
//   during the shutdown process.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-obvious/server"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/cloudzero/cloudzero-agent/app/build"
	config "github.com/cloudzero/cloudzero-agent/app/config/gator"
	"github.com/cloudzero/cloudzero-agent/app/domain/monitor"
	"github.com/cloudzero/cloudzero-agent/app/domain/shipper"
	"github.com/cloudzero/cloudzero-agent/app/handlers"
	"github.com/cloudzero/cloudzero-agent/app/http/middleware"
	"github.com/cloudzero/cloudzero-agent/app/logging"
	"github.com/cloudzero/cloudzero-agent/app/storage/disk"
)

// Shutdown coordination constants for collector-shipper synchronization
const (
	// ShutdownGracePeriod defines the maximum time to wait for collector shutdown confirmation.
	// This prevents indefinite blocking if the collector fails to create its shutdown marker file.
	// 10 seconds provides sufficient time for metric flushing while preventing hang scenarios.
	ShutdownGracePeriod = 10 * time.Second
)

func main() {
	var exitCode int = 0
	var configFile string
	flag.StringVar(&configFile, "config", configFile, "Path to the configuration file")
	flag.Parse()

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		log.Fatal().Err(err).Msg("configuration file does not exist")
	}

	settings, err := config.NewSettings(configFile)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load settings")
	}

	ctx := context.Background()

	// create logging opts
	loggingOpts := make([]logging.LoggerOpt, 0)
	loggingOpts = append(loggingOpts,
		logging.WithLevel(settings.Logging.Level),
		logging.WithSink(logging.NewFieldFilterWriter(os.Stdout, []string{"spanId", "parentSpanId"})),
	)

	// TODO -- add log capture code

	logger, err := logging.NewLogger(loggingOpts...)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create the logger")
	}
	zerolog.DefaultContextLogger = logger
	ctx = logger.WithContext(ctx)

	store, err := disk.NewDiskStore(settings.Database)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize database")
	}

	// Start a monitor that can pickup secrets changes and update the settings
	m := monitor.NewSecretMonitor(ctx, settings)
	defer func() {
		if err = m.Shutdown(); err != nil {
			logger.Err(err).Msg("failed to shutdown secret monitor")
		}
	}()
	if err = m.Run(); err != nil {
		logger.Err(err).Msg("failed to run secret monitor")
		exitCode = 1
		return
	}

	// Create the shipper and start in a thread
	domain, err := shipper.NewMetricShipper(ctx, settings, store)
	if err != nil {
		log.Err(err).Msg("failed to create the metric shipper")
		exitCode = 1
		return
	}

	// MUST BE AFTER DOMAIN
	go func() {
		HandleShutdownEvents(ctx, settings, domain)
		os.Exit(0)
	}()
	go func() {
		if err := domain.Run(); err != nil {
			logger.Err(err).Msg("failed to run metric shipper")
		}
	}()

	defer func() {
		if r := recover(); r != nil {
			logger.Panic().Interface("panic", r).Msg("application panicked, exiting")
		}
	}()

	middleware := []server.Middleware{
		middleware.LoggingMiddlewareWrapper,
		middleware.PromHTTPMiddleware,
	}

	apis := []server.API{
		handlers.NewShipperAPI("/", domain),
	}

	if settings.Server.Profiling {
		apis = append(apis, handlers.NewProfilingAPI("/debug/pprof/"))
	}

	logger.Info().Msg("Starting service")
	server.New(build.Version()).
		WithAddress(fmt.Sprintf(":%d", settings.Server.Port)).
		WithMiddleware(middleware...).
		WithAPIs(apis...).
		WithListener(server.HTTPListener()).
		Run(ctx)
	logger.Info().Msg("Service stopping")

	defer func() {
		os.Exit(exitCode)
	}()
}

// waitForCollectorShutdown monitors for the collector's shutdown completion marker file.
// This function implements the coordination mechanism that ensures the shipper waits
// for the collector to complete shutdown and flush all metrics before proceeding
// with its own shutdown sequence.
//
// Parameters:
//   - ctx: Context for logging
//   - shutdownFile: Path to the collector's shutdown marker file
//   - maxWait: Maximum duration to wait for the shutdown marker
//
// Returns:
//   - bool: true if shutdown marker was detected, false if timeout occurred
//
// Implementation:
//   The function polls for the shutdown marker file every 100ms until either:
//   1. The file is detected (collector has completed shutdown)
//   2. The timeout period expires (failsafe to prevent indefinite blocking)
//
// This coordination ensures proper shutdown sequencing where:
//   1. Collector receives shutdown signal and begins flushing metrics
//   2. Collector creates shutdown marker file when flush is complete
//   3. Shipper detects marker and proceeds with its own shutdown
//   4. Shipper performs final upload of any remaining metrics
//
// The timeout mechanism prevents shipper deadlock if collector fails to
// create the shutdown marker due to crashes or other failures.
func waitForCollectorShutdown(ctx context.Context, shutdownFile string, maxWait time.Duration) bool {
	// Timeline example for 10s timeout:
	// t=0ms:    deadline=10000ms, check file, sleep 100ms
	// t=100ms:  compare 100ms < 10000ms ✓, check file, sleep 100ms
	// t=200ms:  compare 200ms < 10000ms ✓, check file, sleep 100ms
	// ...continues until...
	// t=9900ms: compare 9900ms < 10000ms ✓, check file, sleep 100ms
	// t=10000ms: compare 10000ms < 10000ms ✗, exit loop (no extra sleep)
	deadline := time.Now().Add(maxWait)

	for {
		if _, err := os.Stat(shutdownFile); err == nil {
			log.Ctx(ctx).Info().Str("file", shutdownFile).Msg("collector shutdown marker detected")
			return true
		}

		if time.Now().After(deadline) {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	log.Ctx(ctx).Warn().Str("file", shutdownFile).Dur("timeout", maxWait).Msg("collector shutdown marker not found within timeout")
	return false
}

func HandleShutdownEvents(ctx context.Context, settings *config.Settings, domain *shipper.MetricShipper) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-signalChan

	log.Ctx(ctx).Info().Str("signal", sig.String()).Msg("Received signal, waiting for collector shutdown")

	// Wait for collector to complete shutdown with timeout protection
	shutdownFile := filepath.Join(settings.Database.StoragePath, config.ShutdownMarkerFilename)

	if waitForCollectorShutdown(ctx, shutdownFile, ShutdownGracePeriod) {
		log.Ctx(ctx).Info().Msg("collector shutdown confirmed, proceeding with shipper shutdown")
	} else {
		log.Ctx(ctx).Warn().Msg("proceeding with shipper shutdown without collector confirmation")
	}

	if err := domain.Shutdown(); err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to shutdown metric shipper")
	}

	log.Ctx(ctx).Info().Str("signal", sig.String()).Msg("shipper shutdown complete")
}
