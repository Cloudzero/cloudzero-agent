// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package status provides thread-safe access to CloudZero Agent cluster health reporting.
// This package implements a builder pattern for constructing and accessing ClusterStatus
// protobuf messages used for agent health monitoring and diagnostic reporting.
//
// The ClusterStatus report contains comprehensive health information about the
// CloudZero Agent installation, including connectivity checks, configuration validation,
// and operational status for all agent components. This data is used by:
//   - CloudZero platform for agent health monitoring and alerting
//   - Customer operations teams for troubleshooting installation issues
//   - Agent components for self-diagnostic and validation reporting
//
// The package provides thread-safe read/write access to shared status data,
// enabling multiple agent components to contribute health information concurrently
// without data races or corruption.
package status

import (
	"sync"
)

// Compile-time interface compliance verification for builder type.
// Ensures the builder type correctly implements all Accessor interface methods,
// preventing runtime errors due to missing or incorrectly implemented methods.
var _ interface {
	Accessor
} = (*builder)(nil)

// Accessor defines the interface for thread-safe read/write access to CloudZero Agent cluster status reports.
// This interface enables multiple agent components to concurrently update and read status information
// without requiring external synchronization, simplifying status management across the agent architecture.
//
// The Accessor pattern provides controlled access to the shared ClusterStatus protobuf,
// ensuring data consistency during concurrent status updates from different agent services.
// Used by agent components including webhook handlers, metric collectors, and health check services.
type Accessor interface {
	// AddCheck appends one or more status checks to the cluster status report.
	// This method enables agent components to contribute health check results to the
	// comprehensive cluster status without requiring full report reconstruction.
	//
	// Common usage patterns:
	//   - Webhook services adding connectivity and certificate validation results
	//   - Metric collectors reporting Prometheus endpoint health status
	//   - Storage services contributing disk space and persistence health checks
	//   - Network services adding CloudZero API connectivity validation
	//
	// The method is thread-safe and triggers registered write callbacks to notify
	// listeners of status updates, enabling real-time monitoring and alerting.
	//
	// Multiple checks can be added atomically in a single call for efficiency.
	AddCheck(...*StatusCheck)

	// WriteToReport provides exclusive write access to the cluster status report.
	// The provided function receives a pointer to the ClusterStatus protobuf and can
	// make arbitrary modifications while holding the write lock.
	//
	// Used for:
	//   - Bulk status updates requiring consistency across multiple fields
	//   - Complex status calculations that need to read and modify multiple values
	//   - Status reset or initialization operations during agent startup
	//   - Atomic updates to prevent partial status states during report generation
	//
	// The write lock ensures no other goroutines can read or write the status
	// during the function execution, maintaining data consistency and preventing races.
	// Write callbacks are triggered after the function completes and the lock is released.
	WriteToReport(func(*ClusterStatus))

	// ReadFromReport provides shared read access to the cluster status report.
	// The provided function receives a read-only pointer to the ClusterStatus protobuf
	// while holding a read lock that allows concurrent readers but blocks writers.
	//
	// Used for:
	//   - Status report generation for HTTP endpoints and diagnostic tools
	//   - Health check aggregation across multiple agent components
	//   - Status monitoring and alerting that needs consistent snapshots
	//   - Diagnostic logging that requires stable status values during output
	//
	// Multiple goroutines can read simultaneously, but write operations are blocked
	// during read access to ensure the function receives a consistent view of the status.
	// The function should not retain references to the ClusterStatus pointer after return.
	ReadFromReport(func(*ClusterStatus))
}

// builder provides the concrete implementation of the Accessor interface for cluster status management.
// This struct encapsulates the thread-safe access pattern using read-write mutex synchronization
// and supports event-driven status update notifications through callback functions.
//
// The builder maintains both the status data and synchronization mechanisms required for
// safe concurrent access across multiple agent components, while providing flexibility
// for status change notifications and monitoring integrations.
type builder struct {
	// report holds the shared ClusterStatus protobuf that contains all health and diagnostic information.
	// This is the central data structure modified by all agent components through the Accessor interface.
	// The protobuf format enables efficient serialization for network transmission and storage.
	report *ClusterStatus

	// lock provides read-write mutex synchronization for concurrent access to the report.
	// Read locks allow multiple concurrent readers for status queries and monitoring,
	// while write locks ensure exclusive access during status updates and modifications.
	// This pattern optimizes for the common case of frequent status reads with occasional writes.
	lock *sync.RWMutex

	// onWrite contains callback functions executed after each write operation to the report.
	// These callbacks enable event-driven status processing, such as:
	//   - Triggering status report uploads to CloudZero platform
	//   - Updating local status caches for HTTP endpoints
	//   - Notifying monitoring systems of status changes
	//   - Logging significant status transitions for debugging
	onWrite []func(*ClusterStatus)
}

// NewAccessor creates a new thread-safe Accessor for cluster status management.
// This constructor initializes the builder with the provided ClusterStatus protobuf
// and optional write callback functions for status change notifications.
//
// Parameters:
//   - s: The ClusterStatus protobuf to manage (usually initialized with default values)
//   - onWrite: Optional callback functions executed after each write operation
//
// The returned Accessor enables multiple agent components to safely read and write
// status information concurrently. Write callbacks are executed in the order provided
// after each write operation completes and the write lock is released.
//
// Common usage patterns:
//   - Agent startup: Create accessor with status upload and logging callbacks
//   - Component initialization: Pass accessor to services for health reporting
//   - Monitoring integration: Add callbacks for metric collection and alerting
//
// Returns an Accessor interface that provides thread-safe access to the cluster status.
func NewAccessor(s *ClusterStatus, onWrite ...func(*ClusterStatus)) Accessor {
	return &builder{
		report:  s,
		lock:    &sync.RWMutex{},
		onWrite: onWrite,
	}
}

// onWriteEvent executes all registered write callback functions after status modifications.
// This method provides the event-driven notification mechanism that enables status change
// listeners to respond to report updates without polling or external coordination.
//
// Callbacks are executed in registration order after write locks are released,
// ensuring they cannot interfere with concurrent status operations. If a callback
// function panics, it may prevent subsequent callbacks from executing.
//
// Used internally by WriteToReport and AddCheck to trigger status change notifications.
func (b builder) onWriteEvent() {
	for _, fn := range b.onWrite {
		fn(b.report)
	}
}

// WriteToReport implements the Accessor interface for exclusive write access to the cluster status.
// This method provides thread-safe write access by acquiring an exclusive lock, executing the
// provided function with status modification permissions, and triggering write event callbacks.
//
// The function parameter receives direct access to the ClusterStatus protobuf and can make
// arbitrary modifications during the locked execution window. This enables complex status
// updates that require consistency across multiple fields or calculations based on existing values.
//
// Lock acquisition order:
//  1. Acquire exclusive write lock (blocks all readers and writers)
//  2. Execute provided function with status modification access
//  3. Release write lock
//  4. Execute write event callbacks to notify status change listeners
//
// Used for bulk status updates, atomic multi-field modifications, and status initialization.
func (b builder) WriteToReport(fn func(*ClusterStatus)) {
	b.lock.Lock()
	defer b.lock.Unlock()

	fn(b.report)
	b.onWriteEvent()
}

// ReadFromReport implements the Accessor interface for shared read access to the cluster status.
// This method provides thread-safe read access by acquiring a shared lock that allows multiple
// concurrent readers while blocking write operations during the read execution.
//
// The function parameter receives read-only access to the ClusterStatus protobuf and should
// not retain references to the status pointer after the function returns. Multiple goroutines
// can read simultaneously, optimizing for the common pattern of frequent status queries.
//
// Lock acquisition pattern:
//  1. Acquire shared read lock (allows concurrent readers, blocks writers)
//  2. Execute provided function with read-only status access
//  3. Release read lock (allows blocked writers to proceed)
//
// Used for status report generation, health check aggregation, and monitoring data collection.
func (b builder) ReadFromReport(fn func(*ClusterStatus)) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	fn(b.report)
}

// AddCheck implements the Accessor interface for appending status checks to the cluster report.
// This method provides a convenient way for agent components to add health check results
// without requiring full write access or manual status manipulation.
//
// Multiple status checks can be added atomically in a single operation for efficiency.
// The method acquires an exclusive write lock, appends all provided checks to the report,
// releases the lock, and triggers write event callbacks to notify status change listeners.
//
// Common usage patterns:
//   - agent.AddCheck(connectivityCheck, certificateCheck) // Multiple checks atomically
//   - webhook.AddCheck(admissionValidationCheck) // Single component health
//   - collector.AddCheck(prometheusEndpointCheck) // Service-specific validation
//
// Lock acquisition sequence:
//  1. Acquire exclusive write lock to prevent concurrent modifications
//  2. Append all provided StatusCheck objects to the report's Checks slice
//  3. Release write lock to allow other operations
//  4. Execute write callbacks to notify listeners of the status update
//
// This method is thread-safe and can be called concurrently from multiple agent components.
func (b builder) AddCheck(c ...*StatusCheck) {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.report.Checks = append(b.report.Checks, c...)
	b.onWriteEvent()
}
