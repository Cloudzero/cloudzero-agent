package logging_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/cloudzero/cloudzero-insights-controller/app/logging"
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
