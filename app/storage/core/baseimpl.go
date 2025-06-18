// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package core provides core functionalities for database repository implementations.
// This package includes base implementations for repositories that can be extended
// to fit specific use cases. It supports transaction management and context-based
// database operations.
package core

import (
	"context"

	"gorm.io/gorm"
)

// RawBaseRepoImpl adds core behaviors applicable to any database repository implementation
// and does not assume a simple table model.
type RawBaseRepoImpl struct {
	db *gorm.DB
}

// NewRawBaseRepoImpl creates a new RawBaseRepoImpl for use in a concrete instance of a repository.
func NewRawBaseRepoImpl(db *gorm.DB) RawBaseRepoImpl {
	return RawBaseRepoImpl{
		db: db,
	}
}

// DB returns a *gorm.DB for use in any database calls. It first looks for any *gorm.DB
// in the context, which allows ongoing transactions to be used. Otherwise, it uses the
// default *gorm.DB passed into NewRawBaseRepoImpl. In both cases, the found *gorm.DB
// is augmented using WithContext(ctx).
func (b *RawBaseRepoImpl) DB(ctx context.Context) *gorm.DB {
	if tx, found := FromContext(ctx); found {
		return tx.WithContext(ctx)
	}

	return b.db.WithContext(ctx)
}

// Tx starts a new database transaction and saves it in a new context, ctxTx. This context is
// passed into the block function. Any calls to repository methods that take this context and
// are built on this RawBaseRepoImpl will operate in the context of this transaction. If the
// block returns an error, the transaction is rolled back. If no error is returned, the
// transaction is committed. Note: this mechanism allows for nesting of transactions.
func (b *RawBaseRepoImpl) Tx(ctx context.Context, block func(ctxTx context.Context) error) error {
	db := b.DB(ctx)
	err := db.Transaction(func(tx *gorm.DB) error {
		ctxTx := NewContext(ctx, tx)
		return block(ctxTx)
	})
	return err
}

// BaseRepoImpl adds core behaviors applicable to any database repository implementation.
// Repositories should be defined as follows:
//
//	   type MyAwesomeRepoImpl struct {
//		     core.BaseRepoImpl
//	   }
//
// And constructed as follows:
//
//	   func NewMyAwesomeRepoImpl(db *gorm.DB) *MyAwesomeRepoImpl {
//		     return &MyAwesomeRepoImpl{BaseRepoImpl:
//		       core.NewBaseRepoImpl(db, &MyAwesomeModel)}
//	   }
//
// Any database operations should be invoked using the DB() function:
//
//	r.DB(ctx).Where(...)
//
// Which ensures the proper *gorm.DB instance is used (this enables transaction support).
type BaseRepoImpl struct {
	RawBaseRepoImpl
	model interface{}
}

// NewBaseRepoImpl creates a new BaseRepoImpl for use in a concrete instance of a repository,
// as shown above. The `model` parameter is used to specify the struct that this repository is
// associated with. It is used, for example, in the Count() function.
func NewBaseRepoImpl(db *gorm.DB, model interface{}) BaseRepoImpl {
	return BaseRepoImpl{
		RawBaseRepoImpl: NewRawBaseRepoImpl(db),
		model:           model,
	}
}

// Count returns the number of rows in the table.
func (b *BaseRepoImpl) Count(ctx context.Context) (int, error) {
	var count int64
	err := b.DB(ctx).Model(b.model).Count(&count).Error
	return int(count), TranslateError(err)
}

// DeleteAll deletes all rows in the table (useful in testing).
func (b *BaseRepoImpl) DeleteAll(ctx context.Context) error {
	return TranslateError(b.DB(ctx).Where("1 = 1").Delete(b.model).Error)
}

// key is an unexported type for keys defined in this package.
// This prevents collisions with keys defined in other packages.
type key int

// dbKey is the key for *gorm.DB values in Contexts. It is
// unexported; clients use core.NewContext and core.FromContext
// instead of using this key directly.
var dbKey key

// NewContext returns a new Context that carries value db.
func NewContext(ctx context.Context, db *gorm.DB) context.Context {
	return context.WithValue(ctx, dbKey, db)
}

// FromContext returns the gorm.DB value stored in ctx, if any.
func FromContext(ctx context.Context) (*gorm.DB, bool) {
	db, ok := ctx.Value(dbKey).(*gorm.DB)
	return db, ok
}
