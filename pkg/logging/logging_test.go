// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package logging_test

import (
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/cloudzero/cloudzero-agent/pkg/logging"
)

func TestSetUpLogging(t *testing.T) {
	// Test with LogFormatText
	logging.SetUpLogging("info", logging.LogFormatText)
	logger := logrus.StandardLogger()
	assert.IsType(t, &logging.SequenceLogger{}, logger.Formatter)

	// Test with LogFormatTextColorful
	logging.SetUpLogging("debug", logging.LogFormatTextColorful)
	assert.IsType(t, &logging.SequenceLogger{}, logger.Formatter)

	// Test with LogFormatJSON
	logging.SetUpLogging("warn", logging.LogFormatJSON)
	assert.IsType(t, &logging.SequenceLogger{}, logger.Formatter)

	// Test with unknown log format
	logging.SetUpLogging("error", "unknown")
	assert.IsType(t, &logging.SequenceLogger{}, logger.Formatter)
}

func TestLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    string
		expected logrus.Level
	}{
		{"PanicLevel", "panic", logrus.PanicLevel},
		{"FatalLevel", "fatal", logrus.FatalLevel},
		{"ErrorLevel", "error", logrus.ErrorLevel},
		{"WarnLevel", "warn", logrus.WarnLevel},
		{"InfoLevel", "info", logrus.InfoLevel},
		{"DebugLevel", "debug", logrus.DebugLevel},
		{"TraceLevel", "trace", logrus.TraceLevel},
		{"UnknownLevel", "unknown", logrus.InfoLevel},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := logging.LogLevel(tt.level)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewFileLogger(t *testing.T) {
	file := "test.log"
	logger := logging.LogToFile(file)
	assert.NotNil(t, logger.Out)
	defer func() { _ = os.Remove(file) }()
}
