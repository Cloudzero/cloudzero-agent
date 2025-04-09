// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/cloudzero/cloudzero-agent/app/build"
	config "github.com/cloudzero/cloudzero-agent/app/config/insights-controller"
	"github.com/cloudzero/cloudzero-agent/app/domain/backfiller"
	"github.com/cloudzero/cloudzero-agent/app/domain/housekeeper"
	"github.com/cloudzero/cloudzero-agent/app/domain/k8s"
	"github.com/cloudzero/cloudzero-agent/app/domain/monitor"
	"github.com/cloudzero/cloudzero-agent/app/domain/pusher"
	"github.com/cloudzero/cloudzero-agent/app/http"
	"github.com/cloudzero/cloudzero-agent/app/http/handler"
	"github.com/cloudzero/cloudzero-agent/app/logging"
	"github.com/cloudzero/cloudzero-agent/app/storage/repo"
	"github.com/cloudzero/cloudzero-agent/app/utils"
)

func main() {
	var configFiles config.Files
	var backfill bool
	flag.Var(&configFiles, "config", "Path to the configuration file(s)")
	flag.BoolVar(&backfill, "backfill", false, "Enable backfill mode")
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

	// create a logger
	logger, err := logging.NewLogger(
		logging.WithLevel(settings.Logging.Level),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create the logger")
	}
	zerolog.DefaultContextLogger = logger

	// print settings on debug
	if logger.GetLevel() <= zerolog.DebugLevel {
		enc, err := json.MarshalIndent(settings, "", "  ") //nolint:govet // I actively and vehemently disagree with `shadowing` of `err` in golang
		if err != nil {
			logger.Fatal().Err(err).Msg("failed to encode the config")
		}
		fmt.Println(string(enc))
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
	if err = secretMon.Run(); err != nil {
		log.Fatal().Err(err).Msg("failed to run secret monitor") //nolint:gocritic // It's okay if the `defer cancel()` doesn't run since we're exiting.
	}
	defer func() {
		if innerErr := secretMon.Shutdown(); innerErr != nil {
			log.Err(innerErr).Msg("failed to shut down secret monitor")
		}
	}()

	// create remote metrics writer
	dataPusher := pusher.New(ctx, store, clock, settings)
	if err = dataPusher.Run(); err != nil {
		log.Fatal().Err(err).Msg("failed to start remote metrics writer")
	}
	defer func() {
		log.Ctx(ctx).Debug().Msg("Starting main shutdown process")
		if innerErr := dataPusher.Shutdown(); innerErr != nil {
			log.Err(innerErr).Msg("failed to flush data")
			// Exit with a non-zero status code to indicate failure because we
			// are potentially losing data.
			os.Exit(1)
		}
	}()

	// start the housekeeper to delete old data
	hk := housekeeper.New(ctx, store, clock, settings)
	if err = hk.Run(); err != nil {
		log.Fatal().Err(err).Msg("failed to start database housekeeper")
	}
	defer func() {
		if innerErr := hk.Shutdown(); innerErr != nil {
			log.Err(innerErr).Msg("failed to shut down database housekeeper")
		}
	}()

	if backfill {
		log.Ctx(ctx).Info().Msg("Starting backfill mode")
		// setup k8s client
		k8sClient, err := k8s.NewClient(settings.K8sClient.KubeConfig)
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to build k8s client")
		}
		backfiller.NewBackfiller(k8sClient, store, settings).Start(context.Background())
		return
	}

	// error channel
	errChan := make(chan error)

	server := http.NewServer(settings,
		nil,
		[]http.AdmissionRouteSegment{
			{Route: "/validate/pod", Hook: handler.NewPodHandler(store, settings, clock, errChan)},
			{Route: "/validate/deployment", Hook: handler.NewDeploymentHandler(store, settings, clock, errChan)},
			{Route: "/validate/statefulset", Hook: handler.NewStatefulsetHandler(store, settings, clock, errChan)},
			{Route: "/validate/namespace", Hook: handler.NewNamespaceHandler(store, settings, clock, errChan)},
			{Route: "/validate/node", Hook: handler.NewNodeHandler(store, settings, clock, errChan)},
			{Route: "/validate/job", Hook: handler.NewJobHandler(store, settings, clock, errChan)},
			{Route: "/validate/cronjob", Hook: handler.NewCronJobHandler(store, settings, clock, errChan)},
			{Route: "/validate/daemonset", Hook: handler.NewDaemonSetHandler(store, settings, clock, errChan)},
		}..., // variadic arguments expansion
	)

	go func() {
		// listen shutdown signal
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
		sig := <-signalChan

		log.Ctx(ctx).Error().Str("signal", sig.String()).Msg("Received signal; shutting down...")
		if err := server.Shutdown(ctx); err != nil {
			log.Err(err).Msg("Error shutting down server")
		}
	}()

	if settings.Certificate.Cert == "" || settings.Certificate.Key == "" {
		log.Ctx(ctx).Info().Msg("Starting server without TLS")
		err := server.ListenAndServe()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to listen and serve")
		}
	} else {
		log.Ctx(ctx).Info().Msg("Starting server with TLS")
		// Register a signup handler
		sigc := make(chan os.Signal, 1)
		defer close(sigc)
		signal.Notify(sigc, syscall.SIGHUP)

		// Options
		sig := monitor.WithSIGHUPReload(sigc)
		certs := monitor.WithCertificatesPaths(settings.Certificate.Cert, settings.Certificate.Key, "")
		verify := monitor.WithVerifyConnection()
		cb := monitor.WithOnReload(func(_ *tls.Config) {
			log.Ctx(ctx).Info().Msg("TLS certificates rotated !!")
		})
		server.TLSConfig = monitor.TLSConfig(sig, certs, verify, cb)

		err := server.ListenAndServeTLS("", "")
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to listen and serve")
		}
	}
	// Print a message when the server is stopped.
	log.Ctx(ctx).Info().Msg("Server stopped")
}
