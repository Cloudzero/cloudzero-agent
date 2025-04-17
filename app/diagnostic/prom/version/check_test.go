// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package version_test

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	config "github.com/cloudzero/cloudzero-agent/app/config/validator"
	"github.com/cloudzero/cloudzero-agent/app/diagnostic/prom/version"
	"github.com/cloudzero/cloudzero-agent/app/types/status"
)

func makeReport() status.Accessor {
	return status.NewAccessor(&status.ClusterStatus{})
}

func TestChecker_GetVersion(t *testing.T) {
	tests := []struct {
		name       string
		executable string
		expected   bool
	}{
		{
			name:       "ExecutableNotFound",
			executable: "/path/to/nonexistent/prometheus",
			expected:   false,
		},
		{
			name:       "ExecutableEmpty",
			executable: "",
			expected:   false,
		},
		{
			name:       "Success",
			executable: getPromExecutablePath(),
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Settings{
				Prometheus: config.Prometheus{
					Executable: tt.executable,
				},
			}
			provider := version.NewProvider(context.Background(), cfg)
			accessor := makeReport()

			err := provider.Check(context.Background(), nil, accessor)
			assert.NoError(t, err)

			accessor.ReadFromReport(func(s *status.ClusterStatus) {
				assert.Len(t, s.Checks, 1)
				for _, c := range s.Checks {
					assert.Equal(t, tt.expected, c.Passing)
				}
				if tt.expected {
					assert.NotEmpty(t, s.AgentVersion)
				}
			})
		})
	}
}

func getPromExecutablePath() string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return wd + "/testdata/prometheus"
}
