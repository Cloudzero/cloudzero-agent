// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package loader

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/cloudzero/cloudzero-agent/app/domain/k8s"
	"github.com/cloudzero/cloudzero-agent/app/types/cluster_config"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"

	cfg_gator "github.com/cloudzero/cloudzero-agent/app/config/gator"
	cfg_validator "github.com/cloudzero/cloudzero-agent/app/config/validator"
	cfg_webhook "github.com/cloudzero/cloudzero-agent/app/config/webhook"
)

const (
	FLAG_ACCOUNT           = "account"           //nolint:stylecheck // const value
	FLAG_REGION            = "region"            //nolint:stylecheck // const value
	FLAG_CLUSTER_NAME      = "cluster-name"      //nolint:stylecheck // const value
	FLAG_DEPLOY_NAME       = "deploy-name"       //nolint:stylecheck // const value
	FLAG_CHART_VERSION     = "chart-version"     //nolint:stylecheck // const value
	FLAG_AGENT_VERSION     = "agent-version"     //nolint:stylecheck // const value
	FLAG_VALUES_B64        = "values-b64"        //nolint:stylecheck // const value
	FLAG_CONFIG_VALIDATOR  = "config-validator"  //nolint:stylecheck // const value
	FLAG_CONFIG_WEBHOOK    = "config-webhook"    //nolint:stylecheck // const value
	FLAG_CONFIG_AGGREGATOR = "config-aggregator" //nolint:stylecheck // const value
)

func NewCommand() *cli.Command {
	cmd := &cli.Command{
		Name:    "load",
		Usage:   "load the configuration",
		Aliases: []string{"l"},
		Flags: []cli.Flag{
			&cli.StringFlag{Name: FLAG_ACCOUNT, Usage: "account name", Required: true},
			&cli.StringFlag{Name: FLAG_REGION, Usage: "region", Required: true},
			&cli.StringFlag{Name: FLAG_CLUSTER_NAME, Usage: "cluster name", Required: true},
			&cli.StringFlag{Name: FLAG_DEPLOY_NAME, Usage: "deploy name", Required: true},
			&cli.StringFlag{Name: FLAG_CHART_VERSION, Usage: "current chart version", Required: true},
			&cli.StringFlag{Name: FLAG_AGENT_VERSION, Usage: "current agent version", Required: true},
			&cli.StringFlag{Name: FLAG_VALUES_B64, Usage: "rendered values file encoded into base64", Required: true},
			&cli.StringSliceFlag{Name: FLAG_CONFIG_VALIDATOR, Usage: "list of validator config files", Required: true},
			&cli.StringSliceFlag{Name: FLAG_CONFIG_WEBHOOK, Usage: "list of webhook config files", Required: true},
			&cli.StringSliceFlag{Name: FLAG_CONFIG_AGGREGATOR, Usage: "list of aggregator config files", Required: true},
		},
		Action: run,
	}
	return cmd
}

func run(c *cli.Context) error {
	// create an errors array
	errs := make([]string, 0)

	// get the namespace
	ns, err := k8s.GetNamespace()
	if err != nil {
		log.Ctx(c.Context).Err(err).Msg("failed to get the k8s namespace")
		errs = append(errs, err.Error())
	}

	// get the provider id
	providerID, err := k8s.GetProviderID(c.Context)
	if err != nil {
		log.Ctx(c.Context).Err(err).Msg("failed to get the providerID")
		errs = append(errs, err.Error())
	}

	// get the k8s version
	k8sVersion, err := k8s.GetVersion()
	if err != nil {
		errs = append(errs, err.Error())
	}

	// parse the validator config
	settingsValidatorB64 := ""
	settingsValidator, err := cfg_validator.NewSettings(c.StringSlice(FLAG_CONFIG_VALIDATOR)...)
	if err != nil {
		log.Ctx(c.Context).Err(err).Msg("failed to create the validator config")
		errs = append(errs, err.Error())
	} else {
		enc, err2 := settingsValidator.ToBytes()
		if err2 != nil {
			log.Ctx(c.Context).Err(err2).Msg("failed to encode the validator config to bytes")
			errs = append(errs, fmt.Errorf("failed to encode the settings: %w", err2).Error())
		} else {
			settingsValidatorB64 = base64.StdEncoding.EncodeToString(enc)
		}
	}

	// parse the validator config
	settingsWebhookB64 := ""
	settingsWebhook, err := cfg_webhook.NewSettings(c.StringSlice(FLAG_CONFIG_WEBHOOK)...)
	if err != nil {
		log.Ctx(c.Context).Err(err).Msg("failed to create the webhook config")
		errs = append(errs, err.Error())
	} else {
		enc, err2 := settingsWebhook.ToBytes()
		if err2 != nil {
			log.Ctx(c.Context).Err(err2).Msg("failed to encode the webhook config to bytes")
			errs = append(errs, fmt.Errorf("failed to encode the settings: %w", err2).Error())
		} else {
			settingsWebhookB64 = base64.StdEncoding.EncodeToString(enc)
		}
	}

	// parse the validator config
	settingsAggregatorB64 := ""
	settingsAggregator, err := cfg_gator.NewSettings(c.StringSlice(FLAG_CONFIG_AGGREGATOR)...)
	if err != nil {
		log.Ctx(c.Context).Err(err).Msg("failed to create the aggregator config")
		errs = append(errs, err.Error())
	} else {
		enc, err2 := settingsAggregator.ToBytes()
		if err2 != nil {
			log.Ctx(c.Context).Err(err2).Msg("failed to encode the gator config to bytes")
			errs = append(errs, fmt.Errorf("failed to encode the settings: %w", err2).Error())
		} else {
			settingsAggregatorB64 = base64.StdEncoding.EncodeToString(enc)
		}
	}

	// create a new cluster config object
	cfg := cluster_config.ClusterConfig{
		Account:                   c.String(FLAG_ACCOUNT),
		Region:                    c.String(FLAG_REGION),
		Namespace:                 ns,
		ProviderId:                providerID,
		ClusterName:               c.String(FLAG_CLUSTER_NAME),
		K8SVersion:                k8sVersion,
		DeploymentName:            c.String(FLAG_DEPLOY_NAME),
		ChartVersion:              c.String(FLAG_CHART_VERSION),
		AgentVersion:              c.String(FLAG_AGENT_VERSION),
		ConfigValuesBase64:        c.String(FLAG_VALUES_B64),
		ConfigValidatorBase64:     settingsValidatorB64,
		ConfigWebhookServerBase64: settingsWebhookB64,
		ConfigAggregatorBase64:    settingsAggregatorB64,
		Errors:                    errs,
	}

	// print out the rendered cluster config
	enc, err := json.MarshalIndent(&cfg, "", "  ")
	if err == nil {
		log.Ctx(c.Context).Debug().Msg("Rendered ClusterConfig:")
		fmt.Println(string(enc))
	}

	// TODO -- ship to the remote

	// should not return an error in most cases
	return nil
}
