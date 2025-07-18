// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package types

import "time"

type TimeProvider interface {
	// GetCurrentTime returns the current time.
	GetCurrentTime() time.Time
}
