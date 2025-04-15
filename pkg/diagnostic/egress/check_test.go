// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package egress_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cloudzero/cloudzero-agent/pkg/config"
	"github.com/cloudzero/cloudzero-agent/pkg/diagnostic/egress"
	"github.com/cloudzero/cloudzero-agent/pkg/status"
)

func TestChecker_CheckPing(t *testing.T) {
	tests := []struct {
		name          string
		host          string
		expectPassing bool
		expectError   bool
	}{
		{
			name:          "PingSuccess",
			host:          "https://localhost",
			expectPassing: true,
			expectError:   false,
		},
		{
			name:          "InvalidDomain",
			host:          "invalid-url",
			expectPassing: false,
			expectError:   true,
		},
		{
			name:          "PingFailure",
			host:          "http://nonexistent.domain",
			expectPassing: false,
			expectError:   true,
		},
		{
			name:          "NoHost",
			host:          "",
			expectPassing: false,
			expectError:   true,
		},
	}
	if testing.Short() {
		// RE: https://github.com/actions/runner-images/issues/1519#issuecomment-683790054
		t.Skip("Skipping test in short mode as ping is not supported in GitHub Actions CI")
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Settings{
				Cloudzero: config.Cloudzero{
					Host: tt.host,
				},
			}

			provider := egress.NewProvider(context.Background(), cfg)

			accessor := status.NewAccessor(&status.ClusterStatus{})
			err := provider.Check(context.Background(), nil, accessor)
			assert.NoError(t, err)

			accessor.ReadFromReport(func(s *status.ClusterStatus) {
				assert.Len(t, s.Checks, 1)
				assert.Equal(t, tt.expectPassing, s.Checks[0].Passing)
				if tt.expectError {
					assert.NotEmpty(t, s.Checks[0].Error)
				} else {
					assert.Empty(t, s.Checks[0].Error)
				}
			})
		})
	}
}
