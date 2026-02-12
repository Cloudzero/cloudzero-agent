// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package types defines file abstraction interfaces for CloudZero Agent storage operations.
package types

import "io"

// File defines a comprehensive interface for file operations in the CloudZero Agent storage system.
// This abstraction enables testing with mock files and provides a consistent interface for
// different storage backends while supporting the metric file lifecycle management.
type File interface {
	// io.ReadWriteCloser provides standard I/O operations for reading, writing, and resource cleanup.
	// Implementations must handle proper buffering and flushing for performance optimization.
	io.ReadWriteCloser

	// UniqueID returns a stable identifier for this file across operations.
	// Used for tracking files through the collection, compression, and upload pipeline.
	// Must remain consistent even if the file is moved or renamed.
	UniqueID() string

	// Location returns the current full path of the file in the filesystem.
	// This path may change if the file is moved during processing operations.
	Location() string

	// Rename moves the file to a new location in the filesystem.
	// Used during the metric processing pipeline to move files between stages
	// (e.g., from temporary to processed to uploaded directories).
	// Returns error if the rename operation fails due to permissions or filesystem issues.
	Rename(new string) error

	// Size returns the current size of the file in bytes.
	// Used for storage monitoring, upload progress tracking, and batch processing decisions.
	// Returns error if the file size cannot be determined (e.g., file deleted, permissions).
	Size() (int64, error)
}
