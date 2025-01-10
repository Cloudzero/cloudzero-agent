// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package egress

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

const DiagnosticEgressAccess = config.DiagnosticEgressAccess

type checker struct {
	cfg    *config.Settings
	logger *logrus.Entry
}

func NewProvider(ctx context.Context, cfg *config.Settings) diagnostic.Provider {
	return &checker{
		cfg: cfg,
		logger: logging.NewLogger().
			WithContext(ctx).WithField(logging.OpField, "egress"),
	}
}

func (c *checker) Check(ctx context.Context, client *net.Client, accessor status.Accessor) error {
	// simple unuathenticated check for egress access
	url := fmt.Sprintf("%s", c.cfg.Cloudzero.Host)
	_, err := http.Do(ctx, client, net.MethodGet, nil, nil, url, nil)
	if err == nil {
		accessor.AddCheck(&status.StatusCheck{Name: DiagnosticEgressAccess, Passing: true})
		return nil
	}
	accessor.AddCheck(&status.StatusCheck{Name: DiagnosticEgressAccess, Passing: false, Error: err.Error()})
	return nil
}
