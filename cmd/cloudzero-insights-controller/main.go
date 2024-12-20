// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"crypto/tls"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/build"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/domain/housekeeper"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/domain/k8s"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/domain/monitor"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/domain/pusher"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/domain/scraper"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/http"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/http/handler"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/storage/repo"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/utils"
)

func main() {
	var configFiles config.Files
	flag.Var(&configFiles, "config", "Path to the configuration file(s)")
	flag.Parse()

	clock := &utils.Clock{}

	log.Info().
		Str("app_name", build.AppName).
		Str("version", build.GetVersion()).
		Str("build_time", build.Time).
		Str("rev", build.Rev).
		Str("tag", build.Tag).
		Str("author", build.AuthorName).
		Str("copyright", build.Copyright).
		Str("author_email", build.AuthorEmail).
		Str("charts_repo", build.ChartsRepo).
		Str("platform_endpoint", build.PlatformEndpoint).
		Interface("config_files", configFiles).
		Msg("Starting CloudZero Insights Controller")
	if len(configFiles) == 0 {
		log.Fatal().Msg("No configuration files provided")
	}

	settings, err := config.NewSettings(configFiles...)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load settings")
	}

	// setup database
	store, err := repo.NewInMemoryResourceRepository(clock)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create in-memory resource repository")
	}

	// Start a monitor that can pickup secrets changes and update the settings
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	secretMon := monitor.NewSecretMonitor(ctx, settings)
	if err = secretMon.Start(); err != nil {
		log.Fatal().Err(err).Msg("failed to run secret monitor") //nolint:gocritic // It's okay if the `defer cancel()` doesn't run since we're exiting.
	}
	defer func() { _ = secretMon.Shutdown() }()

	// create remote metrics writer
	dataPusher := pusher.New(ctx, store, clock, settings)
	if err = dataPusher.Start(); err != nil {
		log.Fatal().Err(err).Msg("failed to start remote metrics writer")
	}
	defer func() { _ = dataPusher.Shutdown() }()

	// start the housekeeper to delete old data
	hk := housekeeper.New(ctx, store, clock, settings)
	if err = hk.Start(); err != nil {
		log.Fatal().Err(err).Msg("failed to start database housekeeper")
	}
	defer func() { _ = hk.Shutdown() }()

	// error channel
	errChan := make(chan error)

	// setup k8s client
	k8sClient, err := k8s.NewClient(settings.K8sClient.KubeConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to build k8s client")
	}
	// create scraper
	scraper := scraper.NewScraper(k8sClient, store, settings)

	server := http.NewServer(settings,
		[]http.RouteSegment{
			{Route: "/scrape", Hook: handler.NewScraperHandler(scraper, settings)},
		},
		[]http.AdmissionRouteSegment{
			{Route: "/validate/pod", Hook: handler.NewPodHandler(store, settings, errChan)},
			{Route: "/validate/deployment", Hook: handler.NewDeploymentHandler(store, settings, errChan)},
			{Route: "/validate/statefulset", Hook: handler.NewStatefulsetHandler(store, settings, errChan)},
			{Route: "/validate/namespace", Hook: handler.NewNamespaceHandler(store, settings, errChan)},
			{Route: "/validate/node", Hook: handler.NewNodeHandler(store, settings, errChan)},
			{Route: "/validate/job", Hook: handler.NewJobHandler(store, settings, errChan)},
			{Route: "/validate/cronjob", Hook: handler.NewCronJobHandler(store, settings, errChan)},
			{Route: "/validate/daemonset", Hook: handler.NewDaemonSetHandler(store, settings, errChan)},
		}..., // variadic arguments expansion
	)

	go func() {
		// listen shutdown signal
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
		sig := <-signalChan

		log.Error().Str("signal", sig.String()).Msg("Received signal; shutting down...")
		if err := server.Shutdown(ctx); err != nil {
			log.Err(err).Msg("Error shutting down server")
		}

		// Shutdown after disabling the exposed endpoints
		_ = dataPusher.Shutdown() // flush database content to remote endpoint
	}()

	if settings.Certificate.Cert == "" || settings.Certificate.Key == "" {
		log.Info().Msg("Starting server without TLS")
		err := server.ListenAndServe()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to listen and serve")
		}
	} else {
		log.Info().Msg("Starting server with TLS")
		// Register a signup handler
		sigc := make(chan os.Signal, 1)
		defer close(sigc)
		signal.Notify(sigc, syscall.SIGHUP)

		// Options
		sig := monitor.WithSIGHUPReload(sigc)
		certs := monitor.WithCertificatesPaths(settings.Certificate.Cert, settings.Certificate.Key, "")
		verify := monitor.WithVerifyConnection()
		cb := monitor.WithOnReload(func(_ *tls.Config) {
			log.Info().Msg("TLS certificates rotated !!")
		})
		server.TLSConfig = monitor.TLSConfig(sig, certs, verify, cb)

		err := server.ListenAndServeTLS("", "")
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to listen and serve")
		}
	}
	// Print a message when the server is stopped.
	log.Info().Msg("Server stopped")
}
