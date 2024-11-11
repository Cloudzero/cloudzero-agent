//coverage:ignore
package types

//go:generate mockgen -destination=./mocks/mock_store.go -package=mocks github.com/cloudzero/cirrus-remote-write/app/types Store

import (
	"context"
)

// Appendable represents an interface for append-only storage
// with controlled flushing and monitoring of buffered entries.
type Appendable interface {
	// Put appends one or more metrics to the storage, handling buffering internally.
	Put(context.Context, ...Metric) error

	// Flush writes all buffered data to persistent storage.
	// This can be used to force a write without reaching the row limit.
	Flush() error

	// Pending returns the number of rows currently buffered and not yet written to disk.
	// This can be used to monitor when a flush may be needed.
	Pending() int
}

type AppendableFiles interface {
	GetFiles() ([]string, error)
}

type AppendableReader interface {
	All(context.Context, string) (MetricRange, error)
}

// Store represents a storage interface that provides methods to interact with metrics.
// It embeds the Appendable interface and includes methods to retrieve, delete, and list all metrics.
type Store interface {
	Appendable
	// All retrieves all metrics. It takes a context and an optional string pointer as parameters.
	// Returns a MetricRange and an error.
	All(context.Context, *string) (MetricRange, error)

	// Get retrieves a specific metric by its identifier. It takes a context and a string identifier as parameters.
	// Returns a pointer to a Metric and an error.
	Get(context.Context, string) (*Metric, error)

	// Delete removes a specific metric by its identifier. It takes a context and a string identifier as parameters.
	// Returns an error.
	Delete(context.Context, string) error
}
