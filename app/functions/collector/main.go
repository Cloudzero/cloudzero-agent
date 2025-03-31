// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-obvious/server"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/cloudzero/cloudzero-agent-validator/app/build"
	"github.com/cloudzero/cloudzero-agent-validator/app/config/gator"
	"github.com/cloudzero/cloudzero-agent-validator/app/domain"
	"github.com/cloudzero/cloudzero-agent-validator/app/handlers"
	"github.com/cloudzero/cloudzero-agent-validator/app/logging"
	"github.com/cloudzero/cloudzero-agent-validator/app/store"
	"github.com/cloudzero/cloudzero-agent-validator/app/types"
	"github.com/cloudzero/cloudzero-agent-validator/app/utils"
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

	costMetricStore, err := store.NewDiskStore(settings.Database, store.WithContentIdentifier(store.CostContentIdentifier))
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

	observabilityMetricStore, err := store.NewDiskStore(settings.Database, store.WithContentIdentifier(store.ObservabilityContentIdentifier))
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
		HandleShutdownEvents(ctx, costMetricStore, observabilityMetricStore)
		os.Exit(0)
	}()

	// create the metric collector service interface
	domain, err := domain.NewMetricCollector(settings, clock, costMetricStore, observabilityMetricStore)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize metric collector")
	}
	defer domain.Close()

	loggerMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestLogger := log.Ctx(r.Context()).With().
				Str("path", r.URL.Path).
				Str("method", r.Method).
				Str("remote_addr", r.RemoteAddr).
				Logger()

			requestLogger.Trace().Msg("received request")

			next.ServeHTTP(w, r.WithContext(requestLogger.WithContext(r.Context())))
		})
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
	server.New(
		build.Version(),
		[]server.Middleware{
			loggerMiddleware,
			handlers.PromHTTPMiddleware,
		},
		apis...,
	).Run(ctx)
	logger.Info().Msg("Service stopping")
}

func HandleShutdownEvents(ctx context.Context, appendables ...types.WritableStore) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-signalChan

	log.Ctx(ctx).Info().Str("signal", sig.String()).Msg("Received signal, service stopping")
	for _, appendable := range appendables {
		appendable.Flush()
	}
}
