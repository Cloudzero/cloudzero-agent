// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/build"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/handler"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/http"
)

func main() {
	var configFile string
	flag.StringVar(&configFile, "config", configFile, "Path to the configuration file")
	flag.Parse()

	log.Info().Msgf("Starting CloudZero Insights Controller %s", build.GetVersion())
	if configFile == "" {
		log.Fatal().Msg("No configuration file provided")
	}

	settings, err := config.NewSettings(configFile)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load settings")
	}

	server := http.NewServer(settings,
		[]http.RouteSegment{
			{Route: "/validate/deployment", Hook: handler.NewDeploymentHandler(settings)},
			// TODO: Add others
		}..., // variadic arguments expansion
	)

	go func() {
		// listen shutdown signal
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
		sig := <-signalChan
		log.Error().Msgf("Received %s signal; shutting down...", sig)
		if err := server.Shutdown(context.Background()); err != nil {
			log.Error().Err(err).Msg("Error shutting down server")
		}
	}()

	if err := server.ListenAndServeTLS(settings.Certificate.Cert, settings.Certificate.Key); err != nil {
		log.Fatal().Err(err).Msg("Failed to listen and serve")
	}
	// Print a message when the server is stopped.
	log.Info().Msg("Server stopped")
}
