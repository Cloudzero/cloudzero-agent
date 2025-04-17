// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package cz contains code for checking a CloudZero API token.
package cz

import (
	"context"
	net "net/http"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/cloudzero/cloudzero-agent/app/config/validator"
	"github.com/cloudzero/cloudzero-agent/app/diagnostic"
	"github.com/cloudzero/cloudzero-agent/app/http/client"
	"github.com/cloudzero/cloudzero-agent/app/logging/validator"
	"github.com/cloudzero/cloudzero-agent/app/status"
)

const DiagnosticAPIKey = config.DiagnosticAPIKey

type checker struct {
	cfg    *config.Settings
	logger *logrus.Entry
}

func NewProvider(ctx context.Context, cfg *config.Settings) diagnostic.Provider {
	return &checker{
		cfg: cfg,
		logger: logging.NewLogger().
			WithContext(ctx).WithField(logging.OpField, "cz"),
	}
}

func (c *checker) Check(ctx context.Context, client *net.Client, accessor status.Accessor) error {
	// Hit an authenticated API to verify the API token
	url := c.cfg.Cloudzero.Host + "/v2/insights"
	_, err := http.Do(
		ctx, client, net.MethodGet,
		map[string]string{
			http.HeaderAuthorization:  strings.TrimSpace(c.cfg.Cloudzero.Credential),
			http.HeaderAcceptEncoding: http.ContentTypeJSON,
		},
		nil,
		// TODO: Add HEAD endpoint for container-metrics/status and pass these to check the API key
		// map[string]string{
		// 	http.QueryParamAccountID:   c.cfg.Deployment.AccountID,
		// 	http.QueryParamRegion:      c.cfg.Deployment.Region,
		// 	http.QueryParamClusterName: c.cfg.Deployment.ClusterName,
		// },
		url, nil,
	)
	if err == nil {
		accessor.AddCheck(&status.StatusCheck{Name: DiagnosticAPIKey, Passing: true})
		return nil
	}

	accessor.AddCheck(&status.StatusCheck{Name: DiagnosticAPIKey, Passing: false, Error: err.Error()})
	return nil
}
