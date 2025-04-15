// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package egress contains code for checking egress access.
package egress

import (
	"context"
	net "net/http"

	ping "github.com/prometheus-community/pro-bing"
	"github.com/sirupsen/logrus"

	"github.com/cloudzero/cloudzero-agent/pkg/config"
	"github.com/cloudzero/cloudzero-agent/pkg/diagnostic"
	"github.com/cloudzero/cloudzero-agent/pkg/logging"
	"github.com/cloudzero/cloudzero-agent/pkg/status"
	"github.com/cloudzero/cloudzero-agent/pkg/util"
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
	domain, err := util.ExtractHostnameFromURL(c.cfg.Cloudzero.Host)
	if err != nil {
		accessor.AddCheck(&status.StatusCheck{Name: DiagnosticEgressAccess, Passing: false, Error: err.Error()})
		return nil
	}

	pinger, err := ping.NewPinger(domain)
	if err != nil {
		accessor.AddCheck(&status.StatusCheck{Name: DiagnosticEgressAccess, Passing: false, Error: err.Error()})
		return nil
	}
	pinger.SetNetwork("ip4")
	pinger.SetPrivileged(false)
	pinger.Count = 1
	if err := pinger.RunWithContext(ctx); err != nil {
		accessor.AddCheck(&status.StatusCheck{Name: DiagnosticEgressAccess, Passing: false, Error: err.Error()})
		return nil
	}
	accessor.AddCheck(&status.StatusCheck{Name: DiagnosticEgressAccess, Passing: true})
	return nil
}
