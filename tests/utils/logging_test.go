// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package utils_test

import (
	"testing"

	"github.com/cloudzero/cloudzero-agent/tests/utils"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestLogCapture_Extract(t *testing.T) {
	capture := utils.NewLogCapture(logrus.New())
	capture.Lines = []string{
		"a=one b=two.3.4A",
		`c="quote, with space", d="another quote"`,
	}

	assert.Equal(t, "one", capture.Extract(0, "a"))
	assert.Equal(t, "two.3.4A", capture.Extract(0, "b"))
	assert.Equal(t, `quote, with space`, capture.Extract(1, "c"))
	assert.Equal(t, `another quote`, capture.Extract(1, "d"))
	assert.Equal(t, "", capture.Extract(1, "e"))
}
