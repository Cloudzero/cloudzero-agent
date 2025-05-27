// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package settings contains code for checking the supplied configs
package settings

import (
	"context"
	"encoding/base64"
	"fmt"

	net "net/http"

	"github.com/cloudzero/cloudzero-agent/app/config"
	cfg_gator "github.com/cloudzero/cloudzero-agent/app/config/gator"
	cfg_validator "github.com/cloudzero/cloudzero-agent/app/config/validator"
	cfg_webhook "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/diagnostic"
	logging "github.com/cloudzero/cloudzero-agent/app/logging/validator"
	"github.com/cloudzero/cloudzero-agent/app/types/status"
	"github.com/sirupsen/logrus"
)

const DiagnosticAgentSettings = cfg_validator.DiagnosticAgentSettings

type checker struct {
	logger *logrus.Entry

	// all configs for all applications
	settingsValidatorPath []string
	settingsWebhookPath   []string
	settingsGatorPath     []string
}

func NewProvider(
	ctx context.Context,
	configsValidator []string,
	configsWebhook []string,
	configsAggregator []string,
) diagnostic.Provider {
	return &checker{
		settingsValidatorPath: configsValidator,
		settingsWebhookPath:   configsWebhook,
		settingsGatorPath:     configsAggregator,
		logger: logging.NewLogger().
			WithContext(ctx).WithField(logging.OpField, "settings"),
	}
}

func (c *checker) Check(_ context.Context, _ *net.Client, accessor status.Accessor) error {
	// ensure for no nil
	if len(c.settingsValidatorPath) == 0 || len(c.settingsWebhookPath) == 0 || len(c.settingsGatorPath) == 0 {
		accessor.AddCheck(&status.StatusCheck{
			Name:  DiagnosticAgentSettings,
			Error: "there were no settings provided",
		})
		return nil
	}

	// create the configurations
	settingsValidator, err := cfg_validator.NewSettings(c.settingsValidatorPath...)
	if err != nil {
		e := fmt.Errorf("failed to create the validator config: %w", err)
		c.logger.Error(e.Error())
		accessor.AddCheck(&status.StatusCheck{Name: DiagnosticAgentSettings, Error: e.Error()})
		return nil
	}

	settingsWebhook, err := cfg_webhook.NewSettings(c.settingsWebhookPath...)
	if err != nil {
		e := fmt.Errorf("failed to create the webhook config: %w", err)
		c.logger.Error(e.Error())
		accessor.AddCheck(&status.StatusCheck{Name: DiagnosticAgentSettings, Error: e.Error()})
		return nil
	}

	settingsGator, err := cfg_gator.NewSettings(c.settingsGatorPath...)
	if err != nil {
		e := fmt.Errorf("failed to create the gator config: %w", err)
		c.logger.Error(e.Error())
		accessor.AddCheck(&status.StatusCheck{Name: DiagnosticAgentSettings, Error: e.Error()})
		return nil
	}

	if err := process(settingsValidator, accessor, func(s string) {
		accessor.WriteToReport(func(cs *status.ClusterStatus) {
			cs.ConfigValidatorBase64 = s
		})
	}); err != nil {
		accessor.AddCheck(&status.StatusCheck{Name: DiagnosticAgentSettings, Error: err.Error()})
	}
	if err := process(settingsWebhook, accessor, func(s string) {
		accessor.WriteToReport(func(cs *status.ClusterStatus) {
			cs.ConfigWebhookServerBase64 = s
		})
	}); err != nil {
		accessor.AddCheck(&status.StatusCheck{Name: DiagnosticAgentSettings, Error: err.Error()})
	}
	if err := process(settingsGator, accessor, func(s string) {
		accessor.WriteToReport(func(cs *status.ClusterStatus) {
			cs.ConfigAggregatorBase64 = s
		})
	}); err != nil {
		accessor.AddCheck(&status.StatusCheck{Name: DiagnosticAgentSettings, Error: err.Error()})
	}

	accessor.AddCheck(&status.StatusCheck{Name: DiagnosticAgentSettings, Passing: true})

	return nil
}

func process(
	serializable config.Serializable,
	accessor status.Accessor,
	write func(string),
) error {
	// read the raw bytes
	raw, err := serializable.ToBytes()
	if err != nil {
		e := fmt.Errorf("failed to read the config file: %w", err)
		accessor.AddCheck(&status.StatusCheck{Name: DiagnosticAgentSettings, Error: e.Error()})
		return e
	}

	// convert to base64
	write(base64.StdEncoding.EncodeToString([]byte(raw)))
	return nil
}
