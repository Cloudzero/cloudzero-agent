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

	appendable, err := store.NewParquetStore(settings.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize database")
	}

	ctx := context.Background()

	// Handle shutdown events gracefully
	go HandleShutdownEvents(appendable)

	// create the metric collector service interface
	domain := domain.NewMetricCollector(settings, appendable)
	defer domain.Close()

	// Expose the service
	log.Info().Msg("Starting service")
	server.New(build.Version(), handlers.NewRemoteWriteAPI("/metrics", domain)).Run(ctx)
	log.Info().Msg("Service stopping")
}

func HandleShutdownEvents(appendable types.Appendable) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	log.Info().Msg("Service stopping")
	appendable.Flush()
	os.Exit(0)
}
