// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package config contains a CLI for managing configuration files.
package config

import (
	"context"
	_ "embed"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"github.com/urfave/cli/v2"

	config "github.com/cloudzero/cloudzero-agent/app/config/validator"
	"github.com/cloudzero/cloudzero-agent/app/utils/k8s"
)

//go:embed internal/scrape_config.tmpl
var scrapeConfigTemplate string

var configAlias = []string{"f"}

type ScrapeConfigData struct {
	Targets        []string
	ClusterName    string
	CloudAccountID string
	Region         string
	Host           string
	SecretPath     string
}

func NewCommand(ctx context.Context) *cli.Command {
	cmd := &cli.Command{
		Name:  "config",
		Usage: "configuration utility commands",
		Subcommands: []*cli.Command{
			{
				Name:  "generate",
				Usage: "generates a generic config file",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: config.FlagAccountID, Aliases: []string{"a"}, Usage: config.FlagDescAccountID, Required: true},
					&cli.StringFlag{Name: config.FlagClusterName, Aliases: []string{"c"}, Usage: config.FlagDescClusterName, Required: true},
					&cli.StringFlag{Name: config.FlagRegion, Aliases: []string{"r"}, Usage: config.FlagDescRegion, Required: true},
					&cli.StringFlag{Name: "kubeconfig", Usage: "absolute path to the kubeconfig file", Required: false},
					&cli.StringFlag{Name: "namespace", Usage: "namespace of the cloudzero-agent pod", Required: true},
					&cli.StringFlag{Name: "configmap", Usage: "name of the ConfigMap", Required: true},
					&cli.StringFlag{Name: "pod", Usage: "name of the cloudzero-agent pod", Required: true},
					&cli.StringFlag{Name: "host", Usage: "host for the prometheus remote write endpoint", Required: true},
					&cli.StringFlag{Name: "secret-path", Usage: "path to the secret file", Value: "/etc/config/prometheus/secrets/", Required: false},
				},
				Action: func(c *cli.Context) error {
					kubeconfigPath := c.String("kubeconfig")
					namespace := c.String("namespace")
					configMapName := c.String("configmap")
					host := c.String("host")
					secretPath := c.String("secret-path")

					clientset, err := k8s.BuildKubeClient(kubeconfigPath)
					if err != nil {
						return err
					}

					kubeStateMetricsURL, err := k8s.GetKubeStateMetricsURL(ctx, clientset)
					if err != nil {
						return err
					}

					targets := []string{kubeStateMetricsURL}
					scrapeConfigData := ScrapeConfigData{
						Targets:        targets,
						ClusterName:    c.String(config.FlagClusterName),
						CloudAccountID: c.String(config.FlagAccountID),
						Region:         c.String(config.FlagRegion),
						Host:           host,
						SecretPath:     secretPath,
					}

					configContent, err := Generate(scrapeConfigData)
					if err != nil {
						return err
					}

					configMapData := map[string]string{
						"prometheus.yml": configContent,
					}

					err = k8s.UpdateConfigMap(ctx, clientset, namespace, configMapName, configMapData)
					if err != nil {
						return err
					}

					return nil
				},
			},
			{
				Name:  "validate",
				Usage: "validates the config file",
				Flags: []cli.Flag{
					&cli.StringSliceFlag{
						Name: config.FlagConfigFile, Aliases: configAlias,
						Usage: "input " + config.FlagDescConfFile, Required: true,
					},
				},
				Action: func(c *cli.Context) error {
					configs := c.StringSlice(config.FlagConfigFile)
					if len(configs) == 0 {
						return errors.New("no configuration files specified")
					}

					cfg, err := config.NewSettings(configs...)
					if err != nil {
						return errors.Wrap(err, "config read")
					}
					err = cfg.Validate()
					if err != nil {
						return errors.Wrap(err, "config validation")
					}
					return nil
				},
			},
		},
	}
	return cmd
}

func Generate(data ScrapeConfigData) (string, error) {
	t, err := template.New("scrape_config").Parse(scrapeConfigTemplate)
	if err != nil {
		return "", errors.Wrap(err, "template parser")
	}

	var result strings.Builder
	err = t.Execute(&result, data)
	if err != nil {
		return "", errors.Wrap(err, "executing template")
	}

	return result.String(), nil
}
