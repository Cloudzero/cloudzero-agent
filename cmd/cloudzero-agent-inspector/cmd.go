// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"runtime"

	"github.com/cloudzero/cloudzero-agent-validator/pkg/build"
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

const (
	cliServerConfigListenPort      = 9376
	cliServerConfigDestinationURL  = "https://api.cloudzero.com"
	cliServerConfigLogLevelDefault = zerolog.InfoLevel
)

var cliServerConfig = serverConfig{
	listenPort:     cliServerConfigListenPort,
	destinationURL: cliServerConfigDestinationURL,
	logLevel:       cliServerConfigLogLevelDefault,
}

type zerologLevel struct{}

func (l *zerologLevel) Set(value string) error {
	level, err := zerolog.ParseLevel(value)
	if err != nil {
		return err
	}
	cliServerConfig.logLevel = level
	return nil
}

func (l *zerologLevel) Type() string {
	return "level"
}

func (l *zerologLevel) String() string {
	return cliServerConfig.logLevel.String()
}

var cliParamLogLevel = zerologLevel{}

var rootCmd = &cobra.Command{
	Use:   "cloudzero-agent-inspector",
	Short: "A proxy server for CloudZero API requests",
	Long:  `cloudzero-agent-inspector acts as a proxy server for CloudZero API requests, allowing inspection and debugging of API traffic.`,
	RunE: func(_ *cobra.Command, _ []string) error {
		return runServer(&cliServerConfig)
	},
	Version: fmt.Sprintf("%s.%s/%s-%s", build.Rev, build.Tag, runtime.GOOS, runtime.GOARCH),
}

func init() {
	rootCmd.PersistentFlags().Uint16VarP(&cliServerConfig.listenPort, "port", "p", cliServerConfigListenPort, "Port to listen on")
	rootCmd.PersistentFlags().StringVarP(&cliServerConfig.destinationURL, "destination", "d", cliServerConfigDestinationURL, "Destination URL to proxy requests to")
	rootCmd.PersistentFlags().VarP(&cliParamLogLevel, "log-level", "l", "Log level (panic, fatal, error, warn, info, debug, trace)")
}
