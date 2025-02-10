// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-obvious/server"
	"github.com/rs/zerolog/log"

	"github.com/cloudzero/cloudzero-insights-controller/app/config"
	"github.com/cloudzero/cloudzero-insights-controller/app/domain/shipper"
	"github.com/cloudzero/cloudzero-insights-controller/app/handlers"
	"github.com/cloudzero/cloudzero-insights-controller/app/store"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/build"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/domain/monitor"
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

	appendable, err := store.NewParquetStore(settings.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize database")
	}

	ctx := context.Background()

	// Start a monitor that can pickup secrets changes and update the settings
	m := monitor.NewSecretMonitor(ctx, settings)
	defer func() {
		if err = m.Shutdown(); err != nil {
			log.Err(err).Msg("failed to shutdown secret monitor")
		}
	}()
	if err = m.Start(); err != nil {
		log.Err(err).Msg("failed to run secret monitor")
		exitCode = 1
		return
	}

	go func() {
		HandleShutdownEvents()
		os.Exit(0)
	}()

	// Create the shipper and start in a thread
	domain, err := shipper.NewMetricShipper(ctx, settings, appendable)
	if err != nil {
		log.Err(err).Msg("failed to create the metric shipper")
		exitCode = 1
		return
	}

	defer func() {
		if err := domain.Shutdown(); err != nil {
			log.Err(err).Msg("failed to shutdown metric shipper")
		}
	}()
	go func() {
		if err := domain.Run(); err != nil {
			log.Err(err).Msg("failed to run metric shipper")
		}
	}()

	defer func() {
		if r := recover(); r != nil {
			log.Panic().Interface("panic", r).Msg("application panicked, exiting")
		}
	}()

	log.Info().Msg("Starting service")
	server.New(build.Version(), nil, handlers.NewShipperAPI("/", domain)).Run(context.Background())
	log.Info().Msg("Service stopping")

	defer func() {
		os.Exit(exitCode)
	}()
}

func HandleShutdownEvents() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	log.Info().Msg("Service stopping")
}
