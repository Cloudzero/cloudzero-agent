// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package main implements the CloudZero metric collector service.
//
// The collector is responsible for receiving Prometheus remote write requests,
// processing and filtering metrics, and persisting them to local storage for
// later shipment to CloudZero. It serves as the primary ingestion point for
// metrics data in the CloudZero agent architecture.
//
// Key responsibilities:
//   - HTTP server hosting Prometheus remote write endpoints
//   - Metric collection and filtering (cost vs observability metrics)
//   - Local storage management using disk-based storage
//   - Graceful shutdown coordination with shipper component
//   - Prometheus metrics exposition for monitoring
//   - Optional performance profiling endpoints
//
// Service lifecycle:
//   1. Configuration loading and validation from YAML files
//   2. Logger initialization with configurable levels
//   3. Disk storage setup for cost and observability metrics
//   4. MetricCollector domain service initialization
//   5. HTTP server startup with middleware and API endpoints
//   6. Signal handling for graceful shutdown
//   7. Shutdown marker creation for shipper coordination
//
// Command-line usage:
//   collector -config /path/to/config.yaml
//
// The collector creates a shutdown marker file upon graceful termination
// to coordinate orderly shutdown with the shipper component, ensuring
// all metrics are properly flushed before system shutdown.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/go-obvious/server"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/cloudzero/cloudzero-agent/app/build"
	config "github.com/cloudzero/cloudzero-agent/app/config/gator"
	"github.com/cloudzero/cloudzero-agent/app/domain"
	"github.com/cloudzero/cloudzero-agent/app/handlers"
	"github.com/cloudzero/cloudzero-agent/app/http/middleware"
	"github.com/cloudzero/cloudzero-agent/app/logging"
	"github.com/cloudzero/cloudzero-agent/app/storage/disk"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/cloudzero/cloudzero-agent/app/utils"
)

func main() {
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

	clock := &utils.Clock{}

	ctx := context.Background()
	logger, err := logging.NewLogger(
		logging.WithLevel(settings.Logging.Level),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create the logger")
	}
	zerolog.DefaultContextLogger = logger
	ctx = logger.WithContext(ctx)

	// print settings on debug
	if logger.GetLevel() <= zerolog.DebugLevel {
		enc, err := json.MarshalIndent(settings, "", "  ") //nolint:govet // I actively and vehemently disagree with `shadowing` of `err` in golang
		if err != nil {
			logger.Fatal().Err(err).Msg("failed to encode the config")
		}
		fmt.Println(string(enc))
	}

	costMetricStore, err := disk.NewDiskStore(
		settings.Database,
		disk.WithContentIdentifier(disk.CostContentIdentifier),
		disk.WithMaxInterval(settings.Database.CostMaxInterval),
	)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize database")
	}
	defer func() {
		if innerErr := costMetricStore.Flush(); innerErr != nil {
			logger.Err(innerErr).Msg("failed to flush Parquet store")
		}
		if r := recover(); r != nil {
			logger.Panic().Interface("panic", r).Msg("application panicked, exiting")
		}
	}()

	observabilityMetricStore, err := disk.NewDiskStore(
		settings.Database,
		disk.WithContentIdentifier(disk.ObservabilityContentIdentifier),
		disk.WithMaxInterval(settings.Database.ObservabilityMaxInterval),
	)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize database")
	}
	defer func() {
		if innerErr := observabilityMetricStore.Flush(); innerErr != nil {
			logger.Err(innerErr).Msg("failed to flush Parquet store")
		}
		if r := recover(); r != nil {
			logger.Panic().Interface("panic", r).Msg("application panicked, exiting")
		}
	}()

	// Handle shutdown events gracefully
	go func() {
		HandleShutdownEvents(ctx, settings, costMetricStore, observabilityMetricStore)
		os.Exit(0)
	}()

	// create the metric collector service interface
	domain, err := domain.NewMetricCollector(settings, clock, costMetricStore, observabilityMetricStore)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize metric collector")
	}
	defer domain.Close()

	mw := []server.Middleware{
		middleware.LoggingMiddlewareWrapper,
		middleware.PromHTTPMiddleware,
	}

	apis := []server.API{
		handlers.NewRemoteWriteAPI("/collector", domain),
		handlers.NewPromMetricsAPI("/metrics"),
	}

	if settings.Server.Profiling {
		apis = append(apis, handlers.NewProfilingAPI("/debug/pprof/"))
	}

	// Expose the service
	logger.Info().Msg("Starting service")
	server.New(build.Version()).
		WithAddress(fmt.Sprintf(":%d", settings.Server.Port)).
		WithMiddleware(mw...).
		WithAPIs(apis...).
		WithListener(server.HTTPListener()).
		Run(ctx)
	logger.Info().Msg("Service stopping")
}

// HandleShutdownEvents manages graceful shutdown of the collector service.
// It listens for OS signals (SIGINT, SIGTERM) and performs orderly shutdown
// including metric store flushing and shutdown coordination with the shipper.
//
// Parameters:
//   - ctx: Context for logging and cancellation
//   - settings: Configuration containing storage paths for shutdown coordination
//   - appendables: Variable list of storage interfaces that need flushing
//
// Shutdown sequence:
//   1. Wait for OS termination signal
//   2. Log shutdown initiation with signal details
//   3. Flush all provided storage interfaces to persist pending metrics
//   4. Create shutdown marker file to signal completion to shipper
//   5. Log successful shutdown coordination
//
// The shutdown marker file is critical for proper agent coordination - it tells
// the shipper component that the collector has completed shutdown and flushed
// all metrics, allowing the shipper to safely proceed with its own shutdown
// and final metric upload operations.
//
// Error handling:
//   - Storage flush errors are logged but don't prevent shutdown
//   - Shutdown marker creation errors are logged for debugging
//   - Process continues even if coordination mechanisms fail
func HandleShutdownEvents(ctx context.Context, settings *config.Settings, appendables ...types.WritableStore) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-signalChan

	log.Ctx(ctx).Info().Str("signal", sig.String()).Msg("Received signal, service stopping")
	for _, appendable := range appendables {
		appendable.Flush()
	}

	// Signal shutdown completion to shipper via file marker
	shutdownFile := filepath.Join(settings.Database.StoragePath, config.ShutdownMarkerFilename)
	if err := os.WriteFile(shutdownFile, []byte("done"), config.ShutdownMarkerFileMode); err != nil {
		log.Ctx(ctx).Err(err).Str("file", shutdownFile).Msg("failed to write shutdown marker file")
	} else {
		log.Ctx(ctx).Info().Str("file", shutdownFile).Msg("wrote shutdown completion marker")
	}
}
