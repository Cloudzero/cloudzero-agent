// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package types

import (
	"context"
)

//go:generate mockgen -destination=mocks/storage_mock.go -package=mocks . StorageCommon,Storage,Creator,Reader,Updater,Deleter
//go:generate patch -si mocks/resource_store_mock.diff mocks/resource_store_mock.go

// StorageCommon defines common methods all repos implement by virtue of using BaseRepoImpl.
type StorageCommon interface {
	// Tx runs block in a transaction.
	Tx(ctx context.Context, block func(ctxTx context.Context) error) error
	// Count returns the number of records.
	Count(ctx context.Context) (int, error)
	// DeleteAll deletes all records.
	DeleteAll(ctx context.Context) error
}

// Storage is a CRUD interface that defines the minimal methods that must be
// implemented by a storage that provides access to records. It is a combination
// of the Creator, Reader, Updater, and Deleter interfaces.
type Storage[Model any, ID comparable] interface {
	Creator[Model]
	Reader[Model, ID]
	Updater[Model]
	Deleter[ID]
}

// Creator is an interface that defines the method that must be implemented by a
// repository that provides access to records that can be created.
type Creator[Model any] interface {
	// Create creates a new record in the database. It may modify the input Model
	// along the way (e.g. to set the ID). It returns an error if there was a
	// problem creating the record.
	Create(ctx context.Context, it *Model) error
}

// Reader is an interface that defines the method that must be implemented by a
// repository that provides access to records that can be read.
type Reader[Model any, ID comparable] interface {
	// Get retrieves a record from the database by ID. It returns an error if there
	// was a problem retrieving the record.
	Get(ctx context.Context, id ID) (*Model, error)
}

// Updater is an interface that defines the method that must be implemented by a
// repository that provides access to records that can be updated.
type Updater[Model any] interface {
	// Update updates an existing record in the database. It may modify the input
	// Model along the way (e.g. to set the updated_at timestamp). It returns an
	// error if there was a problem updating the record.
	Update(ctx context.Context, it *Model) error
}

// Deleter is an interface that defines the method that must be implemented by a
// repository that provides access to records that can be deleted.
type Deleter[ID comparable] interface {
	// Delete deletes a record from the database by ID. It returns an error if
	// there was a problem deleting the record.
	Delete(ctx context.Context, id ID) error
}
