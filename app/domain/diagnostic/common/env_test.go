// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package common_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cloudzero/cloudzero-agent/app/domain/diagnostic/common"
)

func TestInPod(t *testing.T) {
	original := os.Getenv("HOSTNAME")
	defer func() {
		if original != "" {
			_ = os.Setenv("HOSTNAME", original)
		}
	}()

	// Test case: HOSTNAME environment variable is set
	os.Setenv("HOSTNAME", "cloudzero-agent-server-56c5764cbf-ltnqh")
	assert.True(t, common.InPod())

	// Test case: HOSTNAME environment variable is not set
	os.Unsetenv("HOSTNAME")
	assert.False(t, common.InPod())
}
