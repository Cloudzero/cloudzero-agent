// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"slices"

	"github.com/pkg/errors"
)

type Prometheus struct {
	Executable                      string   `yaml:"executable" default:"/bin/prometheus" env:"PROMETHEUS_EXECUTABLE" env-description:"Prometheus Executable Path"`
	KubeStateMetricsServiceEndpoint string   `yaml:"kube_state_metrics_service_endpoint" env:"KMS_EP_URL" env-description:"Kube State Metrics Service Endpoint"`
	Configurations                  []string `yaml:"configurations"`
	KubeMetrics                     []string `yaml:"kube_metrics"`
}

func (s *Prometheus) Validate() error {
	// The KSM endpoint is optional when the KubeState plugin is enabled
	// (components.agent.kubeState.enabled), since the plugin provides
	// metrics directly inside the Alloy process without a separate KSM
	// service. When the endpoint is omitted from the validator config,
	// we skip reachability validation.
	if s.KubeStateMetricsServiceEndpoint != "" {
		if !isValidURL(s.KubeStateMetricsServiceEndpoint) {
			return fmt.Errorf("invalid %s", s.KubeStateMetricsServiceEndpoint)
		}
	}

	if len(s.Configurations) == 0 {
		s.Configurations = []string{
			"/etc/prometheus/prometheus.yml",
			"/etc/config/prometheus/configmaps/prometheus.yml",
		}
	} else {
		cleanedPaths := []string{}
		for _, location := range s.Configurations {
			if location == "" {
				continue
			}
			absLocation, err := absFilePath(location)
			if err != nil {
				return err
			}
			if slices.Contains(cleanedPaths, absLocation) {
				continue
			}
			cleanedPaths = append(cleanedPaths, absLocation)
		}
		s.Configurations = cleanedPaths
	}

	if len(s.KubeMetrics) == 0 {
		return errors.New("no KubeMetrics provided")
	}

	return nil
}
