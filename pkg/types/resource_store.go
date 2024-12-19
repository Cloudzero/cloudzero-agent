// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
//
//nolint:gofmt
package types

import (
	"context"
)

type ResourceStore interface {
	StorageCommon
	Storage[ResourceTags, string]

	// Custom methods
	FindFirstBy(ctx context.Context, conds ...interface{}) (*ResourceTags, error)
	FindAllBy(ctx context.Context, conds ...interface{}) ([]*ResourceTags, error)
}
