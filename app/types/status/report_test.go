// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package status_test

import (
	"testing"

	"github.com/cloudzero/cloudzero-agent/app/types/status"
	"github.com/stretchr/testify/assert"
)

func TestStatus_NewBuilder(t *testing.T) {
	tests := []struct {
		name   string
		report *status.ClusterStatus
	}{
		{
			"TestNewBuilder with empty report",
			&status.ClusterStatus{},
		},
	}
	for _, tt := range tests {
		tt := tt // Create a new variable with the same name to avoid copying the lock
		t.Run(tt.name, func(t *testing.T) {
			monitorCount := 0
			builder := status.NewAccessor(tt.report, func(s *status.ClusterStatus) {
				monitorCount++
			})

			builder.AddCheck(&status.StatusCheck{})

			builder.WriteToReport(func(s *status.ClusterStatus) {
				s.State = status.StatusType_STATUS_TYPE_POD_STARTED
			})

			builder.ReadFromReport(func(s *status.ClusterStatus) {
				assert.Equal(t, s.State, status.StatusType_STATUS_TYPE_POD_STARTED, "expected state to be STATUS_TYPE_POD_STARTED")
				checkCount := len(s.Checks)
				assert.Equal(t, checkCount, 1, "expected 1 check in report")
			})

			assert.Equal(t, monitorCount, 2, "expected monitor to be called once")
		})
	}
}
