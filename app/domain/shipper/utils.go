// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

import "github.com/cloudzero/cloudzero-agent-validator/app/types"

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

func GetRemoteFileID(file types.File) string {
	return file.UniqueID() + remoteFileExtension
}
