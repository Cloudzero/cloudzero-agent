// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package types

//go:generate mockgen -destination=mocks/runnable_mock.go -package=mocks . Runnable

type Runnable interface {
	// Run starts the runnable.
	Run() error
	// IsRunning returns true if the runnable is running.
	IsRunning() bool
	// Shutdown shuts down the runnable.
	Shutdown() error
}
