// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package types

import "io"

type File interface {
	io.ReadWriteCloser

	// a unique identifier for the file
	UniqueID() string

	// location of the file
	Location() string

	// change the name / location of the file in the environment
	Rename(new string) error

	// get the size of the file in bytes
	Size() (int64, error)
}
