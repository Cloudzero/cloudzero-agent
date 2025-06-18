// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package shipper provides domain logic for the shipper.
package shipper

import (
	"path/filepath"
	"strings"

	"github.com/cloudzero/cloudzero-agent/app/types"
)

// Chunk splits a list into a matrix of elements with a size of `n`
func Chunk[T any](list []T, n int) [][]T {
	if n <= 0 {
		return [][]T{list}
	}

	var chunks [][]T
	for i := 0; i < len(list); i += n {
		end := min(i+n, len(list))
		chunks = append(chunks, list[i:end])
	}

	return chunks
}

// GetRemoteFileID creates the remote file id for the transposed file
func GetRemoteFileID(file types.File) string {
	return file.UniqueID() + remoteFileExtension
}

// GetRootFileID returns the file id with no file extensions or path information
func GetRootFileID(file string) string {
	parts := strings.SplitN(filepath.Base(file), ".", 2)
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}
