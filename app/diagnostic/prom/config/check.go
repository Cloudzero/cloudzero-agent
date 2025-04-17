// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package config contains a diagnostic provider for checking the Prometheus configuration.
package config

import (
	"context"
	"fmt"
	net "net/http"
	"os"

	config "github.com/cloudzero/cloudzero-agent/app/config/validator"
	"github.com/cloudzero/cloudzero-agent/app/diagnostic"
	logging "github.com/cloudzero/cloudzero-agent/app/logging/validator"
	"github.com/cloudzero/cloudzero-agent/app/types/status"
	"github.com/sirupsen/logrus"
)

const DiagnosticScrapeConfig = config.DiagnosticScrapeConfig

type checker struct {
	cfg    *config.Settings
	logger *logrus.Entry
}

func NewProvider(ctx context.Context, cfg *config.Settings) diagnostic.Provider {
	return &checker{
		cfg: cfg,
		logger: logging.NewLogger().
			WithContext(ctx).WithField(logging.OpField, "prom"),
	}
}

func (c *checker) Check(_ context.Context, _ *net.Client, accessor status.Accessor) error {
	if len(c.cfg.Prometheus.Configurations) == 0 {
		accessor.AddCheck(&status.StatusCheck{
			Name:  DiagnosticScrapeConfig,
			Error: "no prometheus scrape config locations specified in configuration file",
		})
		return nil
	}

	for _, location := range c.cfg.Prometheus.Configurations {
		if _, err := os.Stat(location); os.IsNotExist(err) {
			accessor.AddCheck(
				&status.StatusCheck{Name: DiagnosticScrapeConfig, Error: "find scrape configuration failed: " + location})
			continue
		}
		data, err := os.ReadFile(location)
		if err != nil {
			accessor.AddCheck(
				&status.StatusCheck{Name: DiagnosticScrapeConfig, Error: "failed to read: " + location})
			continue
		}
		accessor.WriteToReport(func(s *status.ClusterStatus) {
			if s.ScrapeConfig != "" {
				s.ScrapeConfig = fmt.Sprintf("%s\n%s", s.ScrapeConfig, string(data))
			} else {
				s.ScrapeConfig = string(data)
			}
			s.Checks = append(s.Checks, &status.StatusCheck{Name: DiagnosticScrapeConfig, Passing: true})
		})
	}
	return nil
}
