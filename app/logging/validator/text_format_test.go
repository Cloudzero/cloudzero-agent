// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package logging

import (
	"bytes"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestPlainTextFormatter_Format(t *testing.T) {
	formatter := &PlainTextFormatter{
		DisableTimestamp: true,
		DisableSorting:   true,
	}

	entry := &logrus.Entry{
		Message: "test message",
		Level:   logrus.InfoLevel,
		Time:    time.Now(),
		Data: logrus.Fields{
			"key1": "value1",
			"key2": "value2",
		},
	}

	buffer := &bytes.Buffer{}
	output, err := formatter.Format(entry)
	assert.NoError(t, err)
	buffer.Write(output)
	assert.NotEmpty(t, buffer.String())
}
