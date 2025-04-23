// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package core_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/cloudzero/cloudzero-agent/app/storage/core"
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

func TestUnit_Storage_Core_Raw(t *testing.T) {
	// create the db
	db, err := core.NewDriver(sqlite.Open("file:memory?mode=memory&cache=shared"))
	require.NoError(t, err, "failed to get the new driver")
	repo := core.NewRawBaseRepoImpl(db)
	db2 := repo.DB(t.Context())
	require.NotNil(t, db2, "failed to get the db")

	err = repo.Tx(t.Context(), func(ctxTx context.Context) error {
		return nil
	})
	require.NoError(t, err, "failed to run the context")
}

func TestUnit_Storage_Core_Base(t *testing.T) {
	// create the db
	db, err := core.NewDriver(sqlite.Open("file:memory?mode=memory&cache=shared"))
	require.NoError(t, err, "failed to get the new driver")
	repo := core.NewBaseRepoImpl(db, nil)
	db2 := repo.DB(t.Context())
	require.NotNil(t, db2, "failed to get the db")

	err = repo.Tx(t.Context(), func(ctxTx context.Context) error {
		return nil
	})
	require.NoError(t, err, "failed to run the context")
}
