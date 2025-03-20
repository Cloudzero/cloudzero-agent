package instr

import (
	"bytes"
	"context"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestUnit_Instr_Span_PrometheusMetrics(t *testing.T) {
	defer _testResentSync()

	// create the prom struct
	p, err := NewPrometheusMetrics(
		WithPromMetrics(testMetric),
		WithNoGoMetrics(),
	)
	require.NoError(t, err)

	// prom handler
	srv := httptest.NewServer(p.Handler())

	// basic
	err = p.Span("test_function_1", func(id string) error {
		time.Sleep(time.Second)
		return nil
	})
	require.NoError(t, err)

	// with error
	err = p.Span("test_function_2", func(id string) error {
		time.Sleep(time.Second / 2)
		return fmt.Errorf("function failed")
	})
	require.Error(t, err)
	require.Equal(t, "function failed", err.Error())

	body := _testSendPromRequest(t, srv.URL)
	require.Contains(t, body, "function_name=\"test_function_1\"")
	require.Contains(t, body, "function_name=\"test_function_2\"")
	require.Contains(t, body, "error=\"function failed\"")
}

func TestUnit_Instr_Span_Context(t *testing.T) {
	defer _testResentSync()

	// Create the PrometheusMetrics struct.
	p, err := NewPrometheusMetrics(
		WithPromMetrics(testMetric),
		WithNoGoMetrics(),
	)
	require.NoError(t, err)

	// Create a Prometheus handler server.
	srv := httptest.NewServer(p.Handler())
	defer srv.Close()

	// Basic span execution.
	err = p.SpanCtx(context.Background(), "test_function_1", func(ctx context.Context, id string) error {
		// Verify that the context contains the span id.
		val := ctx.Value(spanIDKey)
		require.NotNil(t, val, "expected span id value in context")
		require.Equal(t, id, val, "span id in context does not match")
		// Optionally, log using the span logger.
		log := p.StartSpan(ctx, "dummy").Logger()
		log.Info().Msg("inside test_function_1")

		time.Sleep(time.Second)
		return nil
	})
	require.NoError(t, err)

	// Span execution that returns an error.
	err = p.SpanCtx(context.Background(), "test_function_2", func(ctx context.Context,
		id string) error {
		time.Sleep(time.Second / 2)
		return fmt.Errorf("function failed")
	})
	require.Error(t, err)
	require.Equal(t, "function failed", err.Error())

	// Retrieve and check the Prometheus metrics export.
	body := _testSendPromRequest(t, srv.URL)
	require.Contains(t, body, "function_name=\"test_function_1\"")
	require.Contains(t, body, "function_name=\"test_function_2\"")
	require.Contains(t, body, "error=\"function failed\"")
}

func TestUnit_Instr_Span_ParentSpanID(t *testing.T) {
	// Prepare a buffer to capture log output.
	var buf bytes.Buffer

	// Create a zerolog logger that writes to the buffer.
	testLogger := zerolog.New(&buf).With().Timestamp().Logger()

	// Create a context with the test logger attached.
	ctx := testLogger.WithContext(context.Background())

	// Set a parent span id in the context.
	parentID := "parent-1234"
	ctx = context.WithValue(ctx, spanIDKey, parentID)

	// Define a child span id.
	childID := "child-5678"

	// Call the SpanLogger which is expected to:
	// - embed the current span id as "spanId"
	// - search the context for a parent span id and embed it as "parentSpanId"
	spanLogger := SpanLogger(ctx, childID)

	// Emit a test log message.
	spanLogger.Info().Msg("test log")

	// Retrieve output from the buffer.
	logOutput := buf.String()
	// For debugging, you might print the logOutput:
	fmt.Println("Logged output:", logOutput)

	// Verify that the log output contains both span id and parent span id.
	require.Contains(t, logOutput, `"spanId":"child-5678"`, "logger output should contain the child span id")
	require.Contains(t, logOutput, `"parentSpanId":"parent-1234"`, "logger output should contain the parent span id")
}
