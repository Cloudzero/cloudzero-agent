// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package types

import "time"

type DatabaseWriter interface {
	WriteData(data ResourceTags, isCreate bool) error
	UpdateSentAtForRecords(data []ResourceTags, ct time.Time) (int64, error)
	PurgeStaleData(rt time.Duration) error
}

type DatabaseReader interface {
	ReadData(time.Time) ([]ResourceTags, error)
}
