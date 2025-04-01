// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package types

import "time"

//go:generate mockgen -destination=mocks/time_provider_mock.go -package=mocks . TimeProvider

type TimeProvider interface {
	// GetCurrentTime returns the current time.
	GetCurrentTime() time.Time
}
