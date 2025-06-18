// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package version contains code for checking the Kubernetes configuration.
package version

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	config "github.com/cloudzero/cloudzero-agent/app/config/validator"
	"github.com/cloudzero/cloudzero-agent/app/domain/diagnostic"
	"github.com/cloudzero/cloudzero-agent/app/domain/diagnostic/common"
	logging "github.com/cloudzero/cloudzero-agent/app/logging/validator"
	"github.com/cloudzero/cloudzero-agent/app/types/status"
)

const DiagnosticK8sVersion = config.DiagnosticK8sVersion

type checker struct {
	cfg    *config.Settings
	logger *logrus.Entry
}

func NewProvider(ctx context.Context, cfg *config.Settings) diagnostic.Provider {
	return &checker{
		cfg: cfg,
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
	cfg, err := c.getConfig()
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

// getConfig returns a k8s config based on the environment
// detecting if we are on the prometheus pod or running
// on a machine with a kubeconfig file
func (c *checker) getConfig() (*rest.Config, error) {
	if common.InPod() {
		return rest.InClusterConfig()
	}

	kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
	return clientcmd.BuildConfigFromFlags("", kubeconfig)
}
