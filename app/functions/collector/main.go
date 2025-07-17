// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
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
	"sync"
	"syscall"
	"time"

	"github.com/go-obvious/server"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/cloudzero/cloudzero-agent/app/build"
	config "github.com/cloudzero/cloudzero-agent/app/config/gator"
	"github.com/cloudzero/cloudzero-agent/app/domain"
	"github.com/cloudzero/cloudzero-agent/app/handlers"
	"github.com/cloudzero/cloudzero-agent/app/http/middleware"
	"github.com/cloudzero/cloudzero-agent/app/logging"
	"github.com/cloudzero/cloudzero-agent/app/storage/disk"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/cloudzero/cloudzero-agent/app/utils"
)

const (
	// apiBasePath is the base path for the custom metrics API
	apiBasePath = "/apis/custom.metrics.k8s.io/v1beta1"
)

// startCollectorServer starts an HTTP server with core APIs (collector, metrics, profiling)
func startCollectorServer(ctx context.Context, logger *zerolog.Logger, settings *config.Settings, domain *domain.MetricCollector, k8sClient kubernetes.Interface, includeCustomMetrics bool) {
	// HTTP server always serves core APIs
	apis := []server.API{
		handlers.NewRemoteWriteAPI("/collector", domain),
		handlers.NewPromMetricsAPI("/metrics"),
	}

	// Add custom metrics API only if not served by separate HTTPS server
	if includeCustomMetrics {
		apis = append(apis, handlers.NewCustomMetricsAPI(apiBasePath, domain, k8sClient))
	}

	if settings.Server.Profiling {
		apis = append(apis, handlers.NewProfilingAPI("/debug/pprof/"))
	}

	// HTTP server uses all middleware including prometheus metrics
	srv := server.New(build.Version()).
		WithAddress(fmt.Sprintf(":%d", settings.Server.Port)).
		WithMiddleware(middleware.LoggingMiddlewareWrapper, middleware.PromHTTPMiddleware).
		WithAPIs(apis...)

	logger.Info().Uint("port", settings.Server.Port).Msg("Starting HTTP server")
	srv.WithListener(server.HTTPListener()).Run(ctx)
}

// startAutscalerServer starts an HTTPS server that serves only the custom metrics API
func startAutscalerServer(ctx context.Context, logger *zerolog.Logger, settings *config.Settings, domain *domain.MetricCollector, k8sClient kubernetes.Interface) {
	// HTTPS server serves only custom metrics API with discovery endpoints
	apis := []server.API{
		handlers.NewCustomMetricsWithDiscoveryAPI(apiBasePath, domain, k8sClient),
	}

	// Use only logging middleware (no prometheus to avoid duplication with HTTP server)
	srv := server.New(build.Version()).
		WithAddress(fmt.Sprintf(":%d", settings.Server.TLSPort)).
		WithMiddleware(middleware.LoggingMiddlewareWrapper).
		WithAPIs(apis...)

	// TLS certificate paths for dual mode
	certPath := settings.Certificate.Cert
	keyPath := settings.Certificate.Key

	// Validate TLS certificates exist
	if certPath == "" || keyPath == "" {
		logger.Fatal().Msg("TLS certificate paths not configured, cannot start HTTPS server")
	}

	if _, err := os.Stat(certPath); err != nil {
		logger.Fatal().Err(err).Msg("TLS certificate file not found, cannot start HTTPS server")
	}

	if _, err := os.Stat(keyPath); err != nil {
		logger.Fatal().Err(err).Msg("TLS key file not found, cannot start HTTPS server")
	}

	logger.Info().Uint("port", settings.Server.TLSPort).Msg("Starting HTTPS server (custom metrics API only)")
	tlsProvider := func() *tls.Config {
		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			logger.Fatal().Err(err).Msg("Failed to load TLS certificate")
		}
		return &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}
	}

	srv.WithListener(server.TLSListener(30*time.Second, 30*time.Second, 120*time.Second, tlsProvider)).Run(ctx)
}

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

	clock := &utils.Clock{}

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

	costMetricStore, err := disk.NewDiskStore(
		settings.Database,
		disk.WithContentIdentifier(disk.CostContentIdentifier),
		disk.WithMaxInterval(settings.Database.CostMaxInterval),
	)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize database")
	}
	defer func() {
		if innerErr := costMetricStore.Flush(); innerErr != nil {
			logger.Err(innerErr).Msg("failed to flush Parquet store")
		}
		if r := recover(); r != nil {
			logger.Panic().Interface("panic", r).Msg("application panicked, exiting")
		}
	}()

	observabilityMetricStore, err := disk.NewDiskStore(
		settings.Database,
		disk.WithContentIdentifier(disk.ObservabilityContentIdentifier),
		disk.WithMaxInterval(settings.Database.ObservabilityMaxInterval),
	)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize database")
	}
	defer func() {
		if innerErr := observabilityMetricStore.Flush(); innerErr != nil {
			logger.Err(innerErr).Msg("failed to flush Parquet store")
		}
		if r := recover(); r != nil {
			logger.Panic().Interface("panic", r).Msg("application panicked, exiting")
		}
	}()

	// create the metric collector service interface
	domain, err := domain.NewMetricCollector(settings, clock, costMetricStore, observabilityMetricStore)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize metric collector")
	}
	defer domain.Close()

	// Create Kubernetes client for custom metrics API (only when running in cluster)
	var k8sClient kubernetes.Interface
	config, err := rest.InClusterConfig()
	if err != nil {
		// Silently skip Kubernetes client setup when not running in cluster
		// Custom metrics API will work with fallback behavior (only looking at
		// the current pod).
		k8sClient = nil
	} else {
		k8sClient, err = kubernetes.NewForConfig(config)
		if err != nil {
			logger.Warn().Err(err).Msg("failed to create Kubernetes client, custom metrics API will have limited functionality")
			k8sClient = nil
		}
	}

	// Create a cancellable context for server shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Handle shutdown events gracefully
	go func() {
		HandleShutdownEvents(ctx, costMetricStore, observabilityMetricStore)
		logger.Info().Msg("Shutdown signal received, cancelling context")
		cancel()
	}()

	// Expose the service
	logger.Info().Msg("Starting service")

	wg := sync.WaitGroup{}

	if settings.Server.Mode == "https" || settings.Server.Mode == "dual" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			startAutscalerServer(ctx, logger, settings, domain, k8sClient)
		}()
	}

	if settings.Server.Mode == "http" || settings.Server.Mode == "dual" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Include custom metrics API only in HTTP-only mode (not in dual mode)
			includeCustomMetrics := settings.Server.Mode == "http"
			startCollectorServer(ctx, logger, settings, domain, k8sClient, includeCustomMetrics)
		}()
	}

	wg.Wait()

	logger.Info().Msg("Service stopping")
}

func HandleShutdownEvents(ctx context.Context, appendables ...types.WritableStore) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	sig := <-signalChan

	log.Ctx(ctx).Info().Str("signal", sig.String()).Msg("Received signal, service stopping")
	for _, appendable := range appendables {
		appendable.Flush()
	}
}
