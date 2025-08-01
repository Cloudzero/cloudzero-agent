// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package build_test

import (
	"fmt"
	"testing"

	"github.com/cloudzero/cloudzero-agent/app/build"
	"github.com/stretchr/testify/assert"
)

func TestGetVersion(t *testing.T) {
	Rev := "abc123"
	Tag := "v1.0.0"
	Time := "2022-01-01T00:00:00Z"

	build.Rev = Rev
	build.Tag = Tag
	build.Time = Time

	expectedVersion := fmt.Sprintf("%s.%s-%s", Rev, Tag, Time)
	actualVersion := build.GetVersion()

	assert.Equal(t, expectedVersion, actualVersion)
}
