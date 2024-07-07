// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package cz

import (
	"context"
	"fmt"
	net "net/http"

	"github.com/sirupsen/logrus"

	"github.com/cloudzero/cloudzero-agent-validator/pkg/config"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/diagnostic"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/http"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/logging"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/status"
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
	url := fmt.Sprintf("%s/v2/insights", c.cfg.Cloudzero.Host)
	_, err := http.Do(
		ctx, client, net.MethodGet,
		map[string]string{
			http.HeaderAuthorization:  fmt.Sprintf("Bearer %s", c.cfg.Cloudzero.Credential),
			http.HeaderAcceptEncoding: http.ContentTypeJSON,
		},
		map[string]string{
			http.QueryParamAccountID:   c.cfg.Deployment.AccountID,
			http.QueryParamRegion:      c.cfg.Deployment.Region,
			http.QueryParamClusterName: c.cfg.Deployment.ClusterName,
		}, url, nil,
	)
	if err == nil {
		accessor.AddCheck(&status.StatusCheck{Name: DiagnosticAPIKey, Passing: true})
		return nil
	}

	accessor.AddCheck(&status.StatusCheck{Name: DiagnosticAPIKey, Passing: false, Error: err.Error()})
	return nil
}
