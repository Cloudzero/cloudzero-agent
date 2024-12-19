package types

import (
	"context"
)

// StorageCommon defines common methods all repos implement by virtue of using BaseRepoImpl.
type StorageCommon interface {
	Tx(ctx context.Context, block func(ctxTx context.Context) error) error
	Count(ctx context.Context) (int, error)
	DeleteAll(ctx context.Context) error
}

// CRUD is an interface that defines the minimal methods that must be
// implemented by a storage that provides access to records. It is
// a combination of the Creator, Reader, Updater, and Deleter interfaces.
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
