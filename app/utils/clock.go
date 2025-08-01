// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package utils contains utility functions and types used throughout the application.
package utils

import (
	"time"

	"github.com/cloudzero/cloudzero-agent/app/types"
)

var _ types.TimeProvider = (*Clock)(nil)

type Clock struct{}

func (c *Clock) GetCurrentTime() time.Time {
	return time.Now().UTC()
}

// FormatForStorage formats a time.Time value to the ISO 8601 format
func FormatForStorage(t time.Time) string {
	return t.Format("2006-01-02 15:04:05.999999999 -0700 MST")
}
