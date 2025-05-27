// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package namespace contains code for checking the Kubernetes configuration.
package namespace

import (
	"context"
	"net/http"
	"os"

	config "github.com/cloudzero/cloudzero-agent/app/config/validator"
	"github.com/cloudzero/cloudzero-agent/app/domain/diagnostic"
	logging "github.com/cloudzero/cloudzero-agent/app/logging/validator"
	"github.com/cloudzero/cloudzero-agent/app/types/status"
	"github.com/sirupsen/logrus"
)

const DiagnosticK8sNamespace = config.DiagnosticK8sNamespace

type checker struct {
	cfg    *config.Settings
	logger *logrus.Entry
}

func NewProvider(ctx context.Context, cfg *config.Settings) diagnostic.Provider {
	return &checker{
		cfg:    cfg,
		logger: logging.NewLogger().WithContext(ctx).WithField(logging.OpField, "k8s_namespace"),
	}
}

func (c *checker) Check(ctx context.Context, client *http.Client, accessor status.Accessor) error {
	// check the environment var set
	val, exists := os.LookupEnv("NAMESPACE")
	if !exists {
		accessor.AddCheck(&status.StatusCheck{Name: DiagnosticK8sNamespace, Error: "the env variable `NAMESPACE` must exist"})
		return nil
	}

	// add to the report
	accessor.WriteToReport(func(cs *status.ClusterStatus) {
		cs.Namespace = val
	})
	accessor.AddCheck(&status.StatusCheck{Name: DiagnosticK8sNamespace, Passing: true})
	return nil
}
