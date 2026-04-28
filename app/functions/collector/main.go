// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/go-obvious/server"
	"github.com/go-obvious/server/healthz"
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

// Error-rate health-check parameters for the /collector endpoint. These are
// starting points chosen to match the analysis in CP-40749 and should be
// revisited if real traffic patterns call for different thresholds.
//
//   - errorRateWindow: how much recent /collector traffic to consider.
//   - errorRateThreshold: the 5xx fraction above which the health check trips
//     (0.20 = 20%).
//   - errorRateMinFailures: absolute floor on failures required before the
//     threshold is evaluated, so a single transient blip doesn't trip the
//     probe.
//   - errorRateLivenessCooldown: how long /livez stays latched unhealthy
//     after the last unhealthy evaluation. Must be at least as long as the
//     liveness probe's total budget (failureThreshold × periodSeconds) so
//     the probe actually fires a restart instead of the latch releasing
//     first; the chart default of failureThreshold=10 × periodSeconds=30s
//     = 5m matches this constant.
const (
	errorRateWindow           = 60 * time.Second
	errorRateThreshold        = 0.20
	errorRateMinFailures      = 3
	errorRateLivenessCooldown = 5 * time.Minute
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

	// Track recent /collector response statuses so two different probes can
	// react on different timescales:
	//
	//   - /healthz (readiness): instantaneous — fails fast, recovers fast.
	//     When the rate crosses the threshold, Kubernetes removes the pod
	//     from the Service endpoint list. If the pod subsequently stops
	//     seeing traffic and the window ages out, /healthz goes back to 200
	//     and the pod rejoins the Service.
	//
	//   - /livez (liveness): sticky latch — once the tracker has observed an
	//     unhealthy evaluation, /livez stays 503 for errorRateLivenessCooldown
	//     even if /healthz has since returned to 200. This prevents a
	//     readiness-eviction → no-traffic → window-ages-out → readmit flap
	//     loop and guarantees that a persistently bad pod is eventually
	//     restarted rather than bouncing between Ready and NotReady forever.
	collectorErrorRate := middleware.NewErrorRateTracker(errorRateWindow)
	healthz.Register("collector-error-rate", func() error {
		if !collectorErrorRate.Healthy(errorRateThreshold, errorRateMinFailures) {
			return errors.New("/collector 5xx rate exceeds threshold")
		}
		return nil
	})

	apis := []server.API{
		handlers.NewRemoteWriteAPI("/collector", domain, handlers.WithErrorRateTracker(collectorErrorRate)),
		handlers.NewPromMetricsAPI("/metrics"),
		handlers.NewLivezAPI("/livez", collectorErrorRate, errorRateThreshold, errorRateMinFailures, errorRateLivenessCooldown),
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
