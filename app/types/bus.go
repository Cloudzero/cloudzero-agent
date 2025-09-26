// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package types defines event bus abstractions for inter-component communication in the CloudZero Agent.
package types

import "github.com/wagoodman/go-partybus"

// Type aliases for the underlying event bus library types.
// These aliases provide semantic clarity while maintaining compatibility with
// the partybus library used for asynchronous communication between components.
type (
	// Event represents a message or notification sent through the event bus.
	// Used throughout the agent for decoupled communication between services.
	Event = partybus.Event

	// Subscription represents an active subscription to event bus notifications.
	// Consumers use subscriptions to receive events from publishers without direct coupling.
	Subscription = partybus.Subscription
)

// Bus defines the event communication interface for asynchronous messaging between CloudZero Agent components.
// This pattern enables loose coupling between services like file monitoring, metric processing, and storage operations.
type Bus interface {
	// Subscribe creates a new subscription to receive all events published to the bus.
	// Returns a subscription that can be used to receive events asynchronously.
	Subscribe() *Subscription

	// Unsubscribe removes an active subscription from the bus and cleans up resources.
	// Should be called when a component no longer needs to receive events.
	Unsubscribe(*Subscription) error

	// Publish sends an event to all active subscribers on the bus.
	// Events are delivered asynchronously to prevent blocking the publisher.
	Publish(event Event)
}

// FileCreated represents a file system event when a new file is created.
// Used by file monitoring services to notify other components about new metric files
// or configuration changes that require processing.
type FileCreated struct {
	// Name is the full path of the newly created file.
	Name string
}

// FileChanged represents a file system event when an existing file is modified.
// Triggers reprocessing or reloading of configuration and metric files.
type FileChanged struct {
	// Name is the full path of the modified file.
	Name string
}

// FileDeleted represents a file system event when a file is removed.
// Used to clean up references and trigger garbage collection of associated resources.
type FileDeleted struct {
	// Name is the full path of the deleted file.
	Name string
}

// FileRenamed represents a file system event when a file is moved or renamed.
// Important for tracking metric files through the collection and processing pipeline.
type FileRenamed struct {
	// Name is the new full path of the renamed file.
	Name string
}
