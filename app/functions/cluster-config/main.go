// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"time"

	"github.com/cloudzero/cloudzero-agent/app/build"
	"github.com/cloudzero/cloudzero-agent/app/functions/cluster-config/loader"
	"github.com/cloudzero/cloudzero-agent/app/logging"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
)

const (
	FlagLogLevel = "log-level"
)

func main() {
	ctx := ctrlCHandler()

	app := &cli.App{
		Name:     "cloudzero-agent-cluster-config",
		Version:  fmt.Sprintf("%s/%s-%s", build.GetVersion(), runtime.GOOS, runtime.GOARCH),
		Compiled: time.Now(),
		Authors: []*cli.Author{
			{Name: build.AuthorName, Email: build.AuthorEmail},
		},
		Copyright:            build.Copyright,
		Usage:                "a tool for loading and validating cloudzero config files",
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: FlagLogLevel, Usage: "the log level", Required: false, Value: "debug"},
		},
		Before: func(c *cli.Context) (err error) {
			// setu the logger
			logger, err := logging.NewLogger(logging.WithVersion(c.String(FlagLogLevel)))
			if err != nil {
				return fmt.Errorf("failed to create the logger: %w", err)
			}

			ctx = logger.WithContext(ctx)

			return nil
		},
	}

	// add commands
	app.Commands = append(
		app.Commands,
		loader.NewCommand(),
	)

	if err := app.Run(os.Args); err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to run the config loader")
		os.Exit(1)
	}

	log.Ctx(ctx).Info().Msg("Successfully ran cloudzero config handler")
}

func ctrlCHandler() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt)
	go func() {
		<-stopCh
		cancel()
		os.Exit(0) //nolint:revive // need to verify that cancelling the context is sufficient to exit
	}()
	return ctx
}
