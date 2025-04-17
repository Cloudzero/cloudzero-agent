// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package logging_test

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/cloudzero/cloudzero-agent/app/logging/validator"
	"github.com/cloudzero/cloudzero-agent/test"
)

func TestSetUpLoggingSequenceLogger(t *testing.T) {
	logging.SetUpLogging("info", logging.LogFormatText)
	logger := logrus.StandardLogger()
	capture := test.NewLogCaptureWithCurrentFormatter(logger)

	logger.Info("line1")
	logger.Info("line2")
	assert.Equal(t, "1", capture.Extract(0, logging.LogSequence))
	assert.Equal(t, "2", capture.Extract(1, logging.LogSequence))
}
