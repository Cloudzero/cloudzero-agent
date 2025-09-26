// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package types defines common lifecycle management interfaces for CloudZero Agent components.
package types

// Runnable defines the lifecycle contract for long-running components in the CloudZero Agent.
// This interface provides standardized start, status check, and shutdown operations
// for services like collectors, shippers, webhooks, and monitoring components.
type Runnable interface {
	// Run starts the runnable component and blocks until completion or error.
	// Implementations should handle graceful startup and be prepared for shutdown signals.
	Run() error

	// IsRunning returns true if the component is currently active and processing.
	// This method provides status information for health checks and monitoring.
	IsRunning() bool

	// Shutdown initiates graceful shutdown of the component.
	// Implementations should clean up resources, flush pending data, and return promptly.
	Shutdown() error
}
