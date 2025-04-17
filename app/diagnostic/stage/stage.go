// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package stage contains a diagnostic provider for checking the stage.
package stage

import (
	"context"
	net "net/http"

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
	stage  status.StatusType
}

func NewProvider(ctx context.Context, cfg *config.Settings, stage status.StatusType) diagnostic.Provider {
	return &checker{
		cfg: cfg,
		logger: logging.NewLogger().
			WithContext(ctx).
			WithField(logging.OpField, "stage"),
		stage: stage,
	}
}

func (c *checker) Check(_ context.Context, _ *net.Client, accessor status.Accessor) error {
	accessor.WriteToReport(func(s *status.ClusterStatus) {
		s.State = c.stage
	})
	return nil
}
