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

	"github.com/cloudzero/cloudzero-insights-controller/app/config"
	"github.com/cloudzero/cloudzero-insights-controller/app/domain"
	"github.com/cloudzero/cloudzero-insights-controller/app/handlers"
	"github.com/cloudzero/cloudzero-insights-controller/app/store"
	"github.com/cloudzero/cloudzero-insights-controller/app/types"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/build"
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

	ctx := context.Background()
	var logger zerolog.Logger
	{
		logLevel, parseErr := zerolog.ParseLevel(settings.Logging.Level)
		if parseErr != nil {
			log.Fatal().Err(parseErr).Msg("failed to parse log level")
		}
		logger = zerolog.New(os.Stdout).Level(logLevel).With().Timestamp().Logger()
		ctx = logger.WithContext(ctx)
		zerolog.DefaultContextLogger = &logger
	}

	appendable, err := store.NewDiskStore(settings.Database)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize database")
	}
	defer func() {
		if innerErr := appendable.Flush(); innerErr != nil {
			logger.Err(innerErr).Msg("failed to flush Parquet store")
		}
		if r := recover(); r != nil {
			logger.Panic().Interface("panic", r).Msg("application panicked, exiting")
		}
	}()

	// Handle shutdown events gracefully
	go func() {
		HandleShutdownEvents(ctx, appendable)
		os.Exit(0)
	}()

	// create the metric collector service interface
	domain := domain.NewMetricCollector(settings, appendable)
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

	// Expose the service
	logger.Info().Msg("Starting service")
	server.New(
		build.Version(),
		[]server.Middleware{loggerMiddleware},
		handlers.NewRemoteWriteAPI("/metrics", domain),
	).Run(ctx)
	logger.Info().Msg("Service stopping")
}

func HandleShutdownEvents(ctx context.Context, appendable types.Appendable) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-signalChan

	log.Ctx(ctx).Info().Str("signal", sig.String()).Msg("Received signal, service stopping")
	appendable.Flush()
}
