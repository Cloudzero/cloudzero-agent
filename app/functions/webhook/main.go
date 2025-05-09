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
	"strings"
	"syscall"
	"time"

	"github.com/go-obvious/server"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/cloudzero/cloudzero-agent/app/build"
	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/housekeeper"
	"github.com/cloudzero/cloudzero-agent/app/domain/monitor"
	"github.com/cloudzero/cloudzero-agent/app/domain/pusher"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/backfiller"
	"github.com/cloudzero/cloudzero-agent/app/handlers"
	"github.com/cloudzero/cloudzero-agent/app/http/middleware"
	"github.com/cloudzero/cloudzero-agent/app/logging"
	"github.com/cloudzero/cloudzero-agent/app/storage/repo"
	"github.com/cloudzero/cloudzero-agent/app/utils"
	"github.com/cloudzero/cloudzero-agent/app/utils/k8s"
)

func main() {
	var configFiles config.Files
	var backfill bool
	flag.Var(&configFiles, "config", "Path to the configuration file(s)")
	flag.BoolVar(&backfill, "backfill", false, "Enable backfill mode")
	flag.Parse()

	clock := &utils.Clock{}

	settings, err := config.NewSettings(configFiles...)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load settings")
	}

	log.Info().
		Str("version", build.GetVersion()).
		Str("buildTime", build.Time).
		Str("rev", build.Rev).
		Str("tag", build.Tag).
		Str("author", build.AuthorName).
		Str("copyright", build.Copyright).
		Str("authorEmail", build.AuthorEmail).
		Str("chartsRepo", build.ChartsRepo).
		Str("platformEndpoint", settings.Destination).
		Interface("configFiles", configFiles).
		Msg("Starting CloudZero Insights Controller")
	if len(configFiles) == 0 {
		log.Fatal().Msg("No configuration files provided")
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

	wd, err := webhook.NewWebhookFactory(store, settings, clock)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create webhook domain controller")
	}

	if backfill {
		log.Ctx(ctx).Info().Msg("Starting backfill mode")
		// setup k8s client
		k8sClient, err2 := k8s.NewClient(settings.K8sClient.KubeConfig)
		if err2 != nil {
			log.Fatal().Err(err2).Msg("Failed to build k8s client")
		}
		if err3 := backfiller.NewKubernetesObjectEnumerator(k8sClient, wd, settings).Start(context.Background()); err3 != nil {
			log.Fatal().Err(err3).Msg("Failed to start Kubernetes object enumerator")
		}
		return
	}

	defer func() {
		if r := recover(); r != nil {
			logger.Panic().Interface("panic", r).Msg("application panicked, exiting")
		}
	}()

	mw := []server.Middleware{
		middleware.LoggingMiddlewareWrapper,
		middleware.PromHTTPMiddleware,
	}

	apis := []server.API{
		handlers.NewValidationWebhookAPI("/validate", wd),
		handlers.NewPromMetricsAPI("/metrics"),
	}
	if settings.Server.Profiling {
		apis = append(apis, handlers.NewProfilingAPI("/debug/pprof/"))
	}

	// Register a signup handler
	sigc := make(chan os.Signal, 1)
	defer close(sigc)
	signal.Notify(sigc, syscall.SIGHUP)

	// Allow TLS key rotation monitoring if in HTTPS mode,
	// otherwise we are running in HTTP mode - likely to support
	// istio mTLS cluster configurations
	listener := server.HTTPListener()
	if strings.ToLower(settings.Server.Mode) != "http" {
		// Options
		sig := monitor.WithSIGHUPReload(sigc)
		certs := monitor.WithCertificatesPaths(settings.Certificate.Cert, settings.Certificate.Key, "")
		verify := monitor.WithVerifyConnection()
		cb := monitor.WithOnReload(func(_ *tls.Config) {
			log.Ctx(ctx).Info().Msg("TLS certificate initialized")
		})
		listener = server.TLSListener(
			time.Duration(settings.Server.ReadTimeout),
			time.Duration(settings.Server.WriteTimeout),
			time.Duration(settings.Server.IdleTimeout),
			func() *tls.Config { return monitor.TLSConfig(sig, certs, verify, cb) },
		)
	}

	log.Ctx(ctx).Info().Msg("Starting service")
	server.New(build.Version()).
		WithAddress(fmt.Sprintf(":%d", settings.Server.Port)).
		WithMiddleware(mw...).
		WithAPIs(apis...).
		WithListener(listener).
		Run(ctx)
	log.Ctx(ctx).Info().Msg("Server stopped")
}
