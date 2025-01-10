// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"os"
	"testing"

	"github.com/cloudzero/cloudzero-agent-validator/pkg/config"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestPrometheus_Validate(t *testing.T) {
	wd, err := os.Getwd()
	assert.NoError(t, err)
	scrapeConfigFile := wd + "/testdata/prometheus.yml"
	tests := []struct {
		name     string
		prom     config.Prometheus
		expected error
	}{
		{
			name: "ValidPrometheus",
			prom: config.Prometheus{
				KubeStateMetricsServiceEndpoint: kmsServiceEndpoint,
				Configurations:                  []string{scrapeConfigFile},
				KubeMetrics:                     []string{"kube_node_info", "kube_pod_info"},
			},
			expected: nil,
		},
		{
			name: "MissingKubeStateMetricsServiceEndpoint",
			prom: config.Prometheus{
				Configurations: []string{scrapeConfigFile},
				KubeMetrics:    []string{"kube_node_info", "kube_pod_info"},
			},
			expected: errors.New(config.ErrNoKubeStateMetricsServiceEndpointMsg),
		},
		{
			name: "MissingScrapeConfigLocation",
			prom: config.Prometheus{
				KubeStateMetricsServiceEndpoint: kmsServiceEndpoint,
				KubeMetrics:                     []string{"kube_node_info", "kube_pod_info"},
			},
			expected: nil,
		},
		{
			name: "MissingKubeMetrics",
			prom: config.Prometheus{
				KubeStateMetricsServiceEndpoint: kmsServiceEndpoint,
				Configurations:                  []string{scrapeConfigFile},
			},
			expected: errors.New("no KubeMetrics provided"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.prom.Validate()
			if tt.expected == nil {
				assert.NoError(t, err)
				return
			}
			assert.Equal(t, tt.expected.Error(), err.Error())
		})
	}
}
