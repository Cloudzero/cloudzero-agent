// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	"github.com/cloudzero/cloudzero-agent-validator/pkg/build"
	configcmd "github.com/cloudzero/cloudzero-agent-validator/pkg/cmd/config"
	diagcmd "github.com/cloudzero/cloudzero-agent-validator/pkg/cmd/diagnose"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/logging"
)

func main() {
	ctx := ctrlCHandler()

	app := &cli.App{
		Name:     build.AppName,
		Version:  fmt.Sprintf("%s/%s-%s", build.GetVersion(), runtime.GOOS, runtime.GOARCH),
		Compiled: time.Now(),
		Authors: []*cli.Author{
			{Name: build.AuthorName, Email: build.AuthorEmail},
		},
		Copyright:            build.Copyright,
		Usage:                "a tool for validating cloudzero-agent deployments",
		EnableBashCompletion: true,
		Before: func(_ *cli.Context) (err error) {
			logging.SetUpLogging(logging.DefaultLogLevel, logging.LogFormatTextColorful)
			return nil
		},
	}

	app.Commands = append(app.Commands,
		configcmd.NewCommand(ctx),
		diagcmd.NewCommand(),
	)

	err := app.Run(os.Args)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to run command")
	}
}

func ctrlCHandler() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt)
	go func() {
		<-stopCh
		cancel()
		os.Exit(0)
	}()
	return ctx
}
