// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"time"

	"github.com/cloudzero/cloudzero-agent-validator/pkg/inspector"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type serverConfig struct {
	destinationURL string
	listenPort     uint16
	logLevel       zerolog.Level
}

const (
	httpServerReadHeaderTimeout = 15 * time.Second
)

func runServer(cfg *serverConfig) error {
	logger := log.Output(zerolog.ConsoleWriter{Out: os.Stdout}).Level(cfg.logLevel)

	targetURL, err := url.Parse(cfg.destinationURL)
	if err != nil {
		return fmt.Errorf("failed to parse target URL: %w", err)
	}

	czInspector := inspector.New()

	proxy := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			logger.Debug().
				Str("method", pr.In.Method).
				Str("destination", fmt.Sprintf("%s://%s/%s", targetURL.Scheme, targetURL.Host, pr.Out.URL.Path)).
				Int64("length", int64(pr.In.ContentLength)).
				Msg("rewrite request")

			pr.SetURL(targetURL)
			pr.Out.Host = targetURL.Host
		},
		ModifyResponse: func(resp *http.Response) error {
			return czInspector.Inspect(context.Background(), resp, logger)
		},
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, err error) {
			logger.Error().Err(err).Msg("proxy error")
			w.WriteHeader(http.StatusBadGateway)
		},
	}

	server := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.listenPort),
		Handler:           proxy,
		ReadHeaderTimeout: httpServerReadHeaderTimeout,
	}

	logger.Info().
		Str("log-level", cfg.logLevel.String()).
		Str("addr", server.Addr).
		Str("target", targetURL.String()).
		Msg("starting inspector server")

	if err := server.ListenAndServe(); err != nil {
		return fmt.Errorf("failed run server: %w", err)
	}

	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
