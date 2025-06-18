// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package core

import "github.com/google/uuid"

// NewID generates a new unique identifier string using UUID version 4.
// It returns the UUID as a string.
func NewID() string {
	return uuid.New().String()
}
