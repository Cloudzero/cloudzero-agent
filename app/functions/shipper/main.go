// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

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

	go func() {
		HandleShutdownEvents(ctx)
		os.Exit(0)
	}()

	// Create the shipper and start in a thread
	domain, err := shipper.NewMetricShipper(ctx, settings, store)
	if err != nil {
		log.Err(err).Msg("failed to create the metric shipper")
		exitCode = 1
		return
	}

	defer func() {
		if err := domain.Shutdown(); err != nil {
			logger.Err(err).Msg("failed to shutdown metric shipper")
		}
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

func HandleShutdownEvents(ctx context.Context) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-signalChan

	log.Ctx(ctx).Info().Str("signal", sig.String()).Msg("Received signal, service stopping")
}
