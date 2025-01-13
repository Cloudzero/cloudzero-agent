// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cloudzero/cloudzero-agent-validator/pkg/config"
	promcfg "github.com/cloudzero/cloudzero-agent-validator/pkg/diagnostic/prom/config"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/status"
)

func makeReport() status.Accessor {
	return status.NewAccessor(&status.ClusterStatus{})
}

func TestChecker_CheckOK(t *testing.T) {
	wd, err := os.Getwd()
	assert.NoError(t, err)
	scrapeConfigFile := wd + "/testdata/prometheus.yml"

	cfg := &config.Settings{
		Prometheus: config.Prometheus{
			Configurations: []string{scrapeConfigFile},
		},
	}
	provider := promcfg.NewProvider(context.Background(), cfg)

	accessor := makeReport()

	err = provider.Check(context.Background(), nil, accessor)
	assert.NoError(t, err)

	accessor.ReadFromReport(func(s *status.ClusterStatus) {
		assert.Len(t, s.Checks, 1)
		for _, c := range s.Checks {
			assert.True(t, c.Passing)
		}
		assert.NotEmpty(t, s.ScrapeConfig)
	})
}

func TestChecker_NotSet(t *testing.T) {
	tcase := []struct {
		name string
		cfg  *config.Settings
	}{
		{
			name: "empty",
			cfg: &config.Settings{
				Prometheus: config.Prometheus{},
			},
		},
		{
			name: "missing",
			cfg: &config.Settings{
				Prometheus: config.Prometheus{
					Configurations: []string{"/file/not/found"},
				},
			},
		},
	}

	for _, tc := range tcase {
		t.Run(tc.name, func(t *testing.T) {
			provider := promcfg.NewProvider(context.Background(), tc.cfg)
			accessor := makeReport()
			err := provider.Check(context.Background(), nil, accessor)
			assert.NoError(t, err)

			accessor.ReadFromReport(func(s *status.ClusterStatus) {
				assert.Len(t, s.Checks, 1)
				for _, c := range s.Checks {
					assert.False(t, c.Passing)
					assert.NotEmpty(t, c.Error)
				}
			})
		})
	}
}
