// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
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
	"github.com/cloudzero/cloudzero-agent/app/logging"
	"github.com/cloudzero/cloudzero-agent/app/store"
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

	store, err := store.NewDiskStore(settings.Database)
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
		handlers.NewPromMetricsAPI("/metrics"),
		handlers.NewShipperAPI("/", domain),
	}

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
