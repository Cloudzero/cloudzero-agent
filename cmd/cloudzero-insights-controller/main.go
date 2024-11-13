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
	"github.com/cloudzero/cloudzero-insights-controller/pkg/k8s"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/storage"
)

func main() {
	var configFiles config.Files
	flag.Var(&configFiles, "config", "Path to the configuration file(s)")
	flag.Parse()

	log.Info().Msgf("Starting CloudZero Insights Controller %s", build.GetVersion())
	if len(configFiles) == 0 {
		log.Fatal().Msg("No configuration files provided")
	}

	settings, err := config.NewSettings(configFiles...)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load settings")
	}
	log.Info().Msgf("Loaded settings from %v", configFiles)
	log.Info().Msgf("Settings are %v", settings)
	log.Info().Msgf("cloud account Id is %v", settings.CloudAccountID)
	log.Info().Msgf("host name is %v", settings.Host)

	// setup database
	db := storage.SetupDatabase()
	writer := storage.NewWriter(db)
	reader := storage.NewReader(db, settings)
	rmw := http.NewRemoteWriter(writer, reader, settings)

	// error channel
	errChan := make(chan error)

	// setup k8s client
	k8sClient, err := k8s.BuildKubeClient(settings.K8sClient.KubeConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to build k8s client")
	}
	// create scraper
	scraper := k8s.NewScraper(k8sClient, writer, settings)

	server := http.NewServer(settings,
		[]http.RouteSegment{
			{Route: "/scrape", Hook: handler.NewScraperHandler(scraper, settings)},
		},
		[]http.AdmissionRouteSegment{
			{Route: "/validate/pod", Hook: handler.NewPodHandler(writer, settings, errChan)},
			{Route: "/validate/deployment", Hook: handler.NewDeploymentHandler(writer, settings, errChan)},
			{Route: "/validate/statefulset", Hook: handler.NewStatefulsetHandler(writer, settings, errChan)},
			{Route: "/validate/namespace", Hook: handler.NewNamespaceHandler(writer, settings, errChan)},
			{Route: "/validate/node", Hook: handler.NewNodeHandler(writer, settings, errChan)},
			{Route: "/validate/job", Hook: handler.NewJobHandler(writer, settings, errChan)},
			{Route: "/validate/cronjob", Hook: handler.NewCronJobHandler(writer, settings, errChan)},
			{Route: "/validate/daemonset", Hook: handler.NewDaemonSetHandler(writer, settings, errChan)},
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
