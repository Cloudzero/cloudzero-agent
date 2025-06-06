// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package loader provides code to load all the different config types
package loader

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	net "net/http"

	"github.com/cloudzero/cloudzero-agent/app/domain/k8s"
	http "github.com/cloudzero/cloudzero-agent/app/http/client"
	"github.com/cloudzero/cloudzero-agent/app/types/clusterconfig"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"google.golang.org/protobuf/proto"

	cfg_gator "github.com/cloudzero/cloudzero-agent/app/config/gator"
	cfg_validator "github.com/cloudzero/cloudzero-agent/app/config/validator"
	cfg_webhook "github.com/cloudzero/cloudzero-agent/app/config/webhook"
)

const (
	FlagAccount          = "account"
	FlagRegion           = "region"
	FlagClusterName      = "cluster-name"
	FlagReleaseName      = "release-name"
	FlagChartVersion     = "chart-version"
	FlagAgentVersion     = "agent-version"
	FlagValuesB64        = "values-b64"
	FlagConfigValidator  = "config-validator"
	FlagconfigWebhook    = "config-webhook"
	FlagConfigAggregator = "config-aggregator"

	ConfigPayloadHeaderKey   = "X-Cloudzero-Config-Status"
	ConfigPayloadHeaderValue = "configLoad"
	URLPath                  = "/v1/container-metrics/status"
)

func NewCommand() *cli.Command {
	cmd := &cli.Command{
		Name:    "load",
		Usage:   "load the configuration",
		Aliases: []string{"l"},
		Flags: []cli.Flag{
			&cli.StringFlag{Name: FlagAccount, Usage: "account name", Required: true},
			&cli.StringFlag{Name: FlagRegion, Usage: "region", Required: true},
			&cli.StringFlag{Name: FlagClusterName, Usage: "cluster name", Required: true},
			&cli.StringFlag{Name: FlagReleaseName, Usage: "release name", Required: true},
			&cli.StringFlag{Name: FlagChartVersion, Usage: "current chart version", Required: true},
			&cli.StringFlag{Name: FlagAgentVersion, Usage: "current agent version", Required: true},
			&cli.StringFlag{Name: FlagValuesB64, Usage: "rendered values file encoded into base64", Required: true},
			&cli.StringSliceFlag{Name: FlagConfigValidator, Usage: "list of validator config files", Required: true},
			&cli.StringSliceFlag{Name: FlagconfigWebhook, Usage: "list of webhook config files", Required: true},
			&cli.StringSliceFlag{Name: FlagConfigAggregator, Usage: "list of aggregator config files", Required: true},
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
	settingsValidator, err := cfg_validator.NewSettings(c.StringSlice(FlagConfigValidator)...)
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
	settingsWebhook, err := cfg_webhook.NewSettings(c.StringSlice(FlagconfigWebhook)...)
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
	settingsAggregator, err := cfg_gator.NewSettings(c.StringSlice(FlagConfigAggregator)...)
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
	cfg := clusterconfig.ClusterConfig{
		Account:                   c.String(FlagAccount),
		Region:                    c.String(FlagRegion),
		Namespace:                 ns,
		ProviderId:                providerID,
		ClusterName:               c.String(FlagClusterName),
		K8SVersion:                k8sVersion,
		DeploymentName:            c.String(FlagReleaseName),
		ChartVersion:              c.String(FlagChartVersion),
		AgentVersion:              c.String(FlagAgentVersion),
		ConfigValuesBase64:        c.String(FlagValuesB64),
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

	// ship to the remote
	if err := post(c.Context, settingsValidator, &cfg); err != nil {
		log.Ctx(c.Context).Err(err).Msg("failed to ship the config to the remote")
	}

	// should not return an error in most cases
	return nil
}

// post TODO -- refactor this into a shared interface with other similar code
func post(
	ctx context.Context,
	cfg *cfg_validator.Settings,
	cc *clusterconfig.ClusterConfig,
) error {
	if ctx == nil {
		ctx = context.Background()
	}

	if cc == nil {
		return errors.New("nil clusterConfig")
	}

	if cfg.Cloudzero.Host == "" {
		return errors.New("missing cloudzero host")
	}

	if cfg.Cloudzero.Credential == "" {
		return errors.New("missing cloudzero api key")
	}

	// quietly exit
	if cfg.Cloudzero.DisableTelemetry {
		return nil
	}

	// create an http client
	client := retryablehttp.NewClient()

	// encode the data
	data, err := proto.Marshal(cc)
	if err != nil {
		return err
	}

	if len(data) == 0 {
		return errors.New("no data to post")
	}

	// write data into a buffer
	var buf bytes.Buffer
	if _, err := buf.Write(data); err != nil { //nolint:govet // die
		return fmt.Errorf("failed to write the data into a buffer: %w", err)
	}

	log.Info().Int("len", buf.Len()).Msg("compressed size")

	endpoint := fmt.Sprintf("%s%s?configStatus=%s", cfg.Cloudzero.Host, URLPath, ConfigPayloadHeaderValue)
	_, err = http.Do(ctx, client.StandardClient(), net.MethodPost,
		map[string]string{
			http.HeaderAuthorization: "Bearer " + cfg.Cloudzero.Credential,
			http.HeaderContentType:   http.ContentTypeProtobuf,
			ConfigPayloadHeaderKey:   ConfigPayloadHeaderValue,
		},
		map[string]string{
			http.QueryParamAccountID:   cfg.Deployment.AccountID,
			http.QueryParamRegion:      cfg.Deployment.Region,
			http.QueryParamClusterName: cfg.Deployment.ClusterName,
		},
		endpoint,
		&buf,
	)

	return err
}
