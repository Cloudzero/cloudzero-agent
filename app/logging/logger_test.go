// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package logging_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cloudzero/cloudzero-agent/app/logging"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestUnit_Logging_NewLoggerOptions(t *testing.T) {
	// create the logger with a buffer sink
	var buf bytes.Buffer
	logger, err := logging.NewLogger(
		logging.WithLevel("debug"),
		logging.WithSink(&buf),
	)
	require.NoError(t, err, "failed to create logger")

	// check for default context logger
	require.NotNil(t, zerolog.DefaultContextLogger, "default context logger was not set")

	// make log entries
	logger.Debug().Msg("test debug")
	logger.Info().Msg("test info")

	// Ensure the expected output is in our buffer.
	output := buf.String()
	if !strings.Contains(output, "test debug") ||
		!strings.Contains(output, "test info") {
		t.Errorf("expected log messages not found in output: %s", output)
	}
}

func TestUnit_Logging_WithAttrs(t *testing.T) {
	// Create a logger with a custom buffer sink and custom attributes.
	var buf bytes.Buffer
	logger, err := logging.NewLogger(
		logging.WithSink(&buf),
		// Set custom attributes.
		logging.WithAttrs(
			func(ctx zerolog.Context) zerolog.Context {
				return ctx.Str("foo", "bar")
			},
			func(ctx zerolog.Context) zerolog.Context {
				return ctx.Int("num", 42)
			},
		),
	)
	require.NoError(t, err, "failed to create logger with attributes")

	// Make a log entry.
	logger.Info().Msg("hello world")

	// Get the output and assert that our attributes are there.
	output := buf.String()
	require.Contains(t, output, `"foo":"bar"`, "expected attribute foo=bar not found in log output")
	require.Contains(t, output, `"num":42`, "expected attribute num=42 not found in log output")
	require.Contains(t, output, "hello world", "expected log message not found in output")
}
