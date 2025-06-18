// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package diagnose contains a CLI for running diagnostics.
package diagnose

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"google.golang.org/protobuf/encoding/protojson"

	config "github.com/cloudzero/cloudzero-agent/app/config/validator"
	"github.com/cloudzero/cloudzero-agent/app/domain/diagnostic/catalog"
	"github.com/cloudzero/cloudzero-agent/app/domain/diagnostic/runner"
	logging "github.com/cloudzero/cloudzero-agent/app/logging/validator"
	"github.com/cloudzero/cloudzero-agent/app/types/status"
	"github.com/cloudzero/cloudzero-agent/app/utils/telemetry"
)

const (
	configFileDesc = "input " + config.FlagDescConfFile
)

var configAlias = []string{"f"}

func NewCommand() *cli.Command {
	cmd := &cli.Command{
		Name:    "diagnose",
		Usage:   "diagnostic commands",
		Aliases: []string{"diag", "d"},
		Subcommands: []*cli.Command{
			{
				Name:  "get-available",
				Usage: "lists the available diagnostic checks",
				Flags: []cli.Flag{},
				Action: func(c *cli.Context) error {
					registry := catalog.NewCatalog(c.Context, &config.Settings{})
					for _, check := range registry.List() {
						fmt.Println("- " + check)
					}
					return nil
				},
			},
			{
				Name:  "run",
				Usage: "run a specific check or checks",
				Flags: []cli.Flag{
					&cli.StringSliceFlag{Name: "check", Usage: "comma seperated or multi-value list of check(s) to run", Required: true},
					&cli.StringSliceFlag{Name: config.FlagConfigFile, Aliases: configAlias, Usage: configFileDesc, Required: true},
					&cli.BoolFlag{Name: "post", Usage: "if set to true, telemetry will be pushed", Required: false},
				},
				Action: func(c *cli.Context) error {
					requestedChecks := c.StringSlice("check")
					if len(requestedChecks) == 0 {
						return nil
					}
					configs := c.StringSlice(config.FlagConfigFile)
					if len(configs) == 0 {
						return errors.New("no configuration files specified")
					}

					ctx := c.Context

					cfg, err := config.NewSettings(configs...)
					if err != nil {
						logrus.WithError(err).Fatal("Failed to load configuration")
					}
					if err = cfg.Validate(); err != nil {
						logrus.WithError(err).Fatal("Invalid configuration")
					}

					// modify the stages
					cfg.Diagnostics.Stages = []config.Stage{
						{
							Name:    config.ContextStageInit,
							Enforce: false,
							Checks:  requestedChecks,
						},
					}

					engine := runner.NewRunner(cfg, catalog.NewCatalog(ctx, cfg), config.ContextStageInit)

					report, err := engine.Run(ctx)
					if err != nil {
						logrus.WithError(err).Fatal("Failed to run diagnostics")
					}

					report.ReadFromReport(func(cs *status.ClusterStatus) {
						if b, err := protojson.Marshal(cs); err == nil {
							fmt.Println(string(b))
						}
					})

					if c.Bool("post") {
						client := http.DefaultClient
						if err := telemetry.Post(ctx, client, cfg, report); err != nil {
							logrus.WithError(err).Warn("failed to post status")
						}
					}
					return nil
				},
			},
			{
				Name:  config.ContextStageInit,
				Usage: "runs pre-start diagnostic tests",
				Flags: []cli.Flag{
					&cli.StringSliceFlag{Name: config.FlagConfigFile, Aliases: configAlias, Usage: configFileDesc, Required: true},
				},
				Action: func(c *cli.Context) error {
					return runDiagnostics(c, config.ContextStageInit)
				},
			},
			{
				Name:  config.ContextStageStart,
				Usage: "runs post-start diagnostic tests",
				Flags: []cli.Flag{
					&cli.StringSliceFlag{Name: config.FlagConfigFile, Aliases: configAlias, Usage: configFileDesc, Required: true},
				},
				Action: func(c *cli.Context) error {
					return runDiagnostics(c, config.ContextStageStart)
				},
			},
			{
				Name:  config.ContextStateConfigLoad,
				Usage: "checks current configs",
				Flags: []cli.Flag{
					&cli.StringSliceFlag{Name: config.FlagConfigFile, Aliases: configAlias, Usage: configFileDesc, Required: true},
					&cli.StringSliceFlag{Name: config.FlagConfigFileWebhook, Usage: "List of locations for webhook config files", Required: true},
					&cli.StringSliceFlag{Name: config.FlagConfigFileAggregator, Usage: "List of locations for aggregator config files", Required: true},
				},
				Action: func(c *cli.Context) error {
					return runDiagnostics(c, config.ContextStateConfigLoad)
				},
			},
			{
				Name:  config.ContextStageStop,
				Usage: "runs pre-stop diagnostic tests",
				Flags: []cli.Flag{
					&cli.StringSliceFlag{Name: config.FlagConfigFile, Aliases: configAlias, Usage: configFileDesc, Required: true},
				},
				Action: func(c *cli.Context) error {
					return runDiagnostics(c, config.ContextStageStop)
				},
			},
		},
	}
	return cmd
}

func runDiagnostics(c *cli.Context, stage string) error {
	ctx := c.Context
	configs := c.StringSlice(config.FlagConfigFile)
	if len(configs) == 0 {
		return errors.New("no configuration files specified")
	}

	cfg, err := config.NewSettings(configs...)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to load configuration")
	}
	if err = cfg.Validate(); err != nil {
		logrus.WithError(err).Fatal("Invalid configuration")
	}
	if cfg.Logging.Location != "" {
		logging.SetUpLogging(cfg.Logging.Level, logging.LogFormatJSON)
		_ = logging.LogToFile(cfg.Logging.Location)
	}

	// parse registry based on the stage
	registry := catalog.NewCatalog(ctx, cfg)

	engine := runner.NewRunner(cfg, registry, stage)

	report, err := engine.Run(ctx)
	if err != nil {
		logrus.WithError(err).Fatal("Failed to run diagnostics")
	}

	report.ReadFromReport(func(cs *status.ClusterStatus) {
		printNonEmptyClusterStatus(cs)
		if b, err := protojson.Marshal(cs); err == nil {
			if cfg.Logging.Location != "" {
				logrus.WithField("report", string(b)).Info("reporting status")
			}
		}
	})

	if !cfg.Cloudzero.DisableTelemetry {
		client := http.DefaultClient
		if err := telemetry.Post(ctx, client, cfg, report); err != nil {
			logrus.WithError(err).Warn("failed to post status")
		}
	}
	return nil
}

func printNonEmptyClusterStatus(cs *status.ClusterStatus) {
	if cs == nil || len(cs.Checks) == 0 {
		return
	}

	printClusterStatusHeader()
	for _, check := range cs.Checks {
		printClusterStatusRow(check)
	}
}

func printClusterStatusHeader() {
	fmt.Println("Checks:")
	fmt.Printf("%-30s %-10s %-50s\n", "Name", "Passing", "Error")
	//revive:disable-next-line
	fmt.Printf("%-30s %-10s %-50s\n", strings.Repeat("-", 30), strings.Repeat("-", 10), strings.Repeat("-", 50))
}

func printClusterStatusRow(check *status.StatusCheck) {
	if check.Name != "" || check.Passing || check.Error != "" {
		fmt.Printf("%-30s %-10v %-50s\n", check.Name, check.Passing, check.Error)
	}
}
