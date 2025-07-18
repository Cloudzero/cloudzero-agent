// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"context"
)

type ResourceStore interface {
	StorageCommon
	Storage[ResourceTags, string]

	// FindFirstBy returns the first resource tag that matches the given conditions.
	FindFirstBy(ctx context.Context, conds ...interface{}) (*ResourceTags, error)
	// FindAllBy returns all resource tags that match the given conditions.
	FindAllBy(ctx context.Context, conds ...interface{}) ([]*ResourceTags, error)
}
