// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package utils_test

import (
	"testing"
	"time"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func TestFormatForStorage(t *testing.T) {
	tests := []struct {
		name     string
		input    time.Time
		expected string
	}{
		{
			name:     "UTC time",
			input:    time.Date(2023, 10, 10, 10, 10, 10, 0, time.UTC),
			expected: "2023-10-10 10:10:10 +0000 UTC",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.FormatForStorage(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
func TestGetCurrentTime(t *testing.T) {
	clock := &utils.Clock{}

	// Allow a margin of error for the time difference
	margin := 2 * time.Second

	startTime := time.Now().UTC()
	currentTime := clock.GetCurrentTime()
	endTime := time.Now().UTC()

	assert.WithinDuration(t, startTime, currentTime, margin, "currentTime should be within the margin of startTime")
	assert.WithinDuration(t, endTime, currentTime, margin, "currentTime should be within the margin of endTime")
}
