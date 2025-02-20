// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package types_test

import (
	"testing"

	"github.com/cloudzero/cloudzero-insights-controller/app/types"
)

func TestStorageWarning_Thresholds(t *testing.T) {
	tests := []struct {
		name     string
		percent  float64
		expected types.StoreWarning
	}{
		{"None-0", 0.0, types.StorageWarningNone},
		{"None-49", 49.0, types.StorageWarningNone},
		{"None-49.9", 49.9, types.StorageWarningNone},
		{"Low-50", 50.0, types.StorageWarningLow},
		{"Low-50.5", 50.5, types.StorageWarningLow},
		{"Low-74.9", 74.9, types.StorageWarningLow},
		{"Med-75", 75.0, types.StorageWarningMed},
		{"Med-75.1", 75.1, types.StorageWarningMed},
		{"Med-89.9", 89.9, types.StorageWarningMed},
		{"High-90", 90.0, types.StorageWarningHigh},
		{"High-90.5", 90.5, types.StorageWarningHigh},
		{"High-94.9", 94.9, types.StorageWarningHigh},
		{"Crit-95", 95.0, types.StorageWarningCrit},
		{"Crit-95.1", 95.1, types.StorageWarningCrit},
		{"Crit-100", 100.0, types.StorageWarningCrit},
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
