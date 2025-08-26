// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package version contains code for checking the Kubernetes configuration.
package version

import (
	"context"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/discovery"

	config "github.com/cloudzero/cloudzero-agent/app/config/validator"
	"github.com/cloudzero/cloudzero-agent/app/domain/diagnostic"
	"github.com/cloudzero/cloudzero-agent/app/domain/k8s"
	logging "github.com/cloudzero/cloudzero-agent/app/logging/validator"
	"github.com/cloudzero/cloudzero-agent/app/types/status"
)

const DiagnosticK8sVersion = config.DiagnosticK8sVersion

type checker struct {
	cfg            *config.Settings
	logger         *logrus.Entry
	configProvider k8s.ConfigProvider
}

func NewProvider(ctx context.Context, cfg *config.Settings) diagnostic.Provider {
	return &checker{
		cfg:            cfg,
		configProvider: k8s.NewConfigProvider(),
		logger: logging.NewLogger().
			WithContext(ctx).WithField(logging.OpField, "k8s_version"),
	}
}

func (c *checker) Check(_ context.Context, client *http.Client, accessor status.Accessor) error {
	version, err := c.getK8sVersion(client)
	if err != nil {
		accessor.AddCheck(
			&status.StatusCheck{Name: DiagnosticK8sVersion, Passing: false, Error: err.Error()},
		)
		return nil
	}

	accessor.WriteToReport(func(s *status.ClusterStatus) {
		s.K8SVersion = string(version)
	})
	accessor.AddCheck(
		&status.StatusCheck{Name: DiagnosticK8sVersion, Passing: true},
	)
	return nil
}

// getK8sVersion returns the k8s version of the cluster
func (c *checker) getK8sVersion(_ *http.Client) ([]byte, error) {
	cfg, err := c.configProvider.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "read config")
	}

	// TODO: Improve the HTTPMock to allow us to override the client
	// To Control the response

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "discovery client")
	}

	information, err := discoveryClient.ServerVersion()
	if err != nil {
		return nil, errors.Wrap(err, "server version")
	}

	return []byte(fmt.Sprintf("%s.%s", information.Major, information.Minor)), nil
}
