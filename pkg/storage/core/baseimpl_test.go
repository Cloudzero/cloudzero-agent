// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package core_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/storage/core"
)

func TestBaseRepoImpl_Context(t *testing.T) {
	db := &gorm.DB{}
	ctx := context.Background()

	// empty context, not found
	from, found := core.FromContext(context.Background())
	assert.Nil(t, from)
	assert.False(t, found)

	// context with tx, found
	ctxTx := core.NewContext(ctx, db)
	from, found = core.FromContext(ctxTx)
	assert.Same(t, from, db)
	assert.True(t, found)
}
