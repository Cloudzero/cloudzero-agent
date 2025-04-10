// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package core_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/cloudzero/cloudzero-agent/app/storage/core"
)

func TestNewID(t *testing.T) {
	id := core.NewID()
	assert.NotEmpty(t, id, "NewID should not return an empty string")

	_, err := uuid.Parse(id)
	assert.NoError(t, err, "NewID should return a valid UUID")
}

func TestNewIDUniqueness(t *testing.T) {
	id1 := core.NewID()
	id2 := core.NewID()
	assert.NotEqual(t, id1, id2, "NewID should return unique IDs")
}
