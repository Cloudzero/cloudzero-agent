// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package types_test

import (
	"testing"

	"github.com/cloudzero/cloudzero-agent/app/types"
)

func TestStorageWarning_Thresholds(t *testing.T) {
	tests := []struct {
		name     string
		percent  float64
		expected types.StoreWarning
	}{
		{"None-0", 0.0, types.StoreWarningNone},
		{"None-49", 49.0, types.StoreWarningNone},
		{"None-49.9", 49.9, types.StoreWarningNone},
		{"Low-50", 50.0, types.StoreWarningLow},
		{"Low-50.5", 50.5, types.StoreWarningLow},
		{"Low-64.9", 64.9, types.StoreWarningLow},
		{"Med-65", 65.0, types.StoreWarningMed},
		{"Med-75.1", 75.1, types.StoreWarningMed},
		{"Med-79.9", 79.9, types.StoreWarningMed},
		{"High-80", 80.0, types.StoreWarningHigh},
		{"High-85.5", 85.5, types.StoreWarningHigh},
		{"High-89.9", 89.9, types.StoreWarningHigh},
		{"Crit-90", 90.0, types.StoreWarningCrit},
		{"Crit-95.1", 95.1, types.StoreWarningCrit},
		{"Crit-100", 100.0, types.StoreWarningCrit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			du := &types.StoreUsage{PercentUsed: tt.percent}
			got := du.GetStorageWarning()
			if got != tt.expected {
				t.Errorf("For %.2f%%, expected %v (%d), got %v (%d)",
					tt.percent,
					tt.expected, uint(tt.expected),
					got, uint(got),
				)
			}
		})
	}
}
