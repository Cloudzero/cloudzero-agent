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
	"github.com/cloudzero/cloudzero-insights-controller/pkg/storage"
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
	// error channel
	errChan := make(chan error)

	// setup database
	db := storage.SetupDatabase()
	writer := storage.NewWriter(db)
	reader := storage.NewReader(db, settings)
	rmw := http.NewRemoteWriter(writer, reader, settings)
	server := http.NewServer(settings,
		[]http.RouteSegment{
			{Route: "/validate/pod", Hook: handler.NewPodHandler(writer, settings, errChan)},
			{Route: "/validate/deployment", Hook: handler.NewDeploymentHandler(writer, settings, errChan)},
			{Route: "/validate/statefulset", Hook: handler.NewStatefulsetHandler(writer, settings, errChan)},
			{Route: "/validate/namespace", Hook: handler.NewNamespaceHandler(writer, settings, errChan)},
			{Route: "/validate/node", Hook: handler.NewNodeHandler(writer, settings, errChan)},
			// TODO: Add others
		}..., // variadic arguments expansion
	)

	go func() {
		// listen shutdown signal
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
		sig := <-signalChan
		log.Error().Msgf("Received %s signal; shutting down...", sig)
		// flush database before shutdown
		rmw.Flush()
		if err := server.Shutdown(context.Background()); err != nil {
			log.Error().Err(err).Msg("Error shutting down server")
		}
	}()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Info().Msgf("Recovered from panic in remote write: %v", r)
			}
		}()
		ticker := rmw.StartRemoteWriter()
		defer ticker.Stop()
	}()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Info().Msgf("Recovered from panic in stale data removal: %v", r)
			}
		}()
		hk := storage.NewHouseKeeper(writer, settings)
		hk.StartHouseKeeper()
	}()

	if settings.Certificate.Cert == "" || settings.Certificate.Key == "" {
		log.Info().Msg("Starting server without TLS")
		err := server.ListenAndServe()
		if err != nil {
			log.Fatal().Err(err).Msgf("Failed to listen and serve: %v", err)
		}
	} else {
		log.Info().Msg("Starting server with TLS")
		err := server.ListenAndServeTLS(settings.Certificate.Cert, settings.Certificate.Key)
		if err != nil {
			log.Fatal().Err(err).Msgf("Failed to listen and serve: %v", err)
		}
	}
	// Print a message when the server is stopped.
	log.Info().Msg("Server stopped")
}
