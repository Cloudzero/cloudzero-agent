// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

//coverage:ignore
package types

//go:generate mockgen -destination=./mocks/mock_store.go -package=mocks github.com/cloudzero/cloudzero-insights-controller/app/types Store

import (
	"context"
	"os"
	"path/filepath"
)

// WritableStore represents an interface for append-only storage
// with controlled flushing and monitoring of buffered entries.
type WritableStore interface {
	// All retrieves all metrics. It takes a context and an optional string pointer as parameters.
	// Returns a MetricRange and an error.
	All(context.Context, string) (MetricRange, error)

	// Put appends one or more metrics to the storage, handling buffering internally.
	Put(context.Context, ...Metric) error

	// Flush writes all buffered data to persistent storage.
	// This can be used to force a write without reaching the row limit.
	Flush() error

	// Pending returns the number of rows currently buffered and not yet written to disk.
	// This can be used to monitor when a flush may be needed.
	Pending() int
}

// ReadableStore is for performing read operations against the store
// It is understood that this will only read to the store, and not write against it.
//
// In addition, this implements the StoreMonitor interface to monitor the store.
type ReadableStore interface {
	StoreMonitor

	// GetFiles returns the list of files in the store. `paths` can be used to add a specific location
	GetFiles(paths ...string) ([]string, error)

	// ListFiles gives a list of `[]os.DirEntry` for a given store. `paths` can be used to add a specific location
	ListFiles(paths ...string) ([]os.DirEntry, error)

	// Walk runs a `proccess` on the file loc of the implementation store
	Walk(loc string, process filepath.WalkFunc) error
}

// Store represents a storage interface that provides methods to interact with metrics.
// It allows for writing and reading from the store
type Store interface {
	WritableStore
	ReadableStore
}
