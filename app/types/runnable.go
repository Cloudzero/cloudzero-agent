// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package types

type Runnable interface {
	// Run starts the runnable.
	Run() error
	// Shutdown stops the runnable.
	Shutdown() error
}
