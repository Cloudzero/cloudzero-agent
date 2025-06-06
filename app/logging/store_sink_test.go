// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package logging_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/cloudzero/cloudzero-agent/app/logging"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockStore is a thread-safe mock implementation of types.Store
type mockStore struct {
	mu        sync.Mutex
	putCalls  int
	metrics   []types.Metric
	putDelay  time.Duration // Optional: to simulate work and increase contention window
	putError  error         // Optional: to simulate errors from Put
	lastError error         // Stores the last error encountered in Put
}

func newMockStore() *mockStore {
	return &mockStore{
		metrics: make([]types.Metric, 0),
	}
}

func (ms *mockStore) Put(ctx context.Context, m ...types.Metric) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if ms.putError != nil {
		ms.lastError = ms.putError
		return ms.putError
	}

	ms.putCalls++
	ms.metrics = append(ms.metrics, m...)
	if ms.putDelay > 0 {
		time.Sleep(ms.putDelay)
	}
	ms.lastError = nil // Clear last error on success
	return nil
}

func (ms *mockStore) All(ctx context.Context, file string) (types.MetricRange, error) {
	return types.MetricRange{
		Metrics: ms.metrics,
		Next:    nil,
	}, nil
}

func (ms *mockStore) Flush() error {
	return nil
}

func (ms *mockStore) Pending() int {
	return 0
}

func (ms *mockStore) GetPutCalls() int {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	return ms.putCalls
}

func (ms *mockStore) GetMetrics() []types.Metric {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	// Return a copy to avoid race conditions if the caller modifies the slice
	copiedMetrics := make([]types.Metric, len(ms.metrics))
	copy(copiedMetrics, ms.metrics)
	return copiedMetrics
}

func (ms *mockStore) GetLastError() error {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	return ms.lastError
}

// TestUnit_Logging_StoreWriter_Write_Basic tests basic functionality of a single write.
func TestUnit_Logging_StoreWriter_Write_Basic(t *testing.T) {
	store := newMockStore()
	ctx := context.Background()
	clusterName := "test-cluster"
	cloudAccountID := "test-account"

	writer := logging.StoreWriter(ctx, store, clusterName, cloudAccountID)

	logTime := time.Now().UTC()
	logMsg := "hello world"
	logData := map[string]interface{}{
		zerolog.TimestampFieldName: logTime.Format(zerolog.TimeFieldFormat),
		zerolog.LevelFieldName:     "info",
		zerolog.MessageFieldName:   logMsg,
		"custom_field":             "custom_value",
		"number_field":             123.45,
		"bool_field":               true,
		"null_field":               nil,
		"obj_field":                map[string]string{"a": "b"},
		"arr_field":                []int{1, 2, 3},
	}

	jsonData, err := json.Marshal(logData)
	require.NoError(t, err)

	n, err := writer.Write(jsonData)
	require.NoError(t, err)
	assert.Equal(t, len(jsonData), n)

	assert.Equal(t, 1, store.GetPutCalls())
	metrics := store.GetMetrics()
	require.Len(t, metrics, 1)

	metric := metrics[0]
	assert.Equal(t, clusterName, metric.ClusterName)
	assert.Equal(t, cloudAccountID, metric.CloudAccountID)
	assert.Equal(t, "log", metric.MetricName)
	assert.Equal(t, logMsg, metric.Value)
	assert.WithinDuration(t, logTime, metric.TimeStamp, time.Second) // zerolog time format might lose sub-second precision

	assert.Equal(t, logTime.Format(zerolog.TimeFieldFormat), metric.Labels[zerolog.TimestampFieldName])
	assert.Equal(t, "info", metric.Labels[zerolog.LevelFieldName])
	assert.Equal(t, logMsg, metric.Labels[zerolog.MessageFieldName])
	assert.Equal(t, "custom_value", metric.Labels["custom_field"])
	assert.Equal(t, "123.45", metric.Labels["number_field"]) // %g format
	assert.Equal(t, "true", metric.Labels["bool_field"])
	assert.Equal(t, "null", metric.Labels["null_field"])
	assert.Equal(t, `{"a":"b"}`, metric.Labels["obj_field"])
	assert.Equal(t, `[1,2,3]`, metric.Labels["arr_field"])

	// Test unmarshal error - should still consume log
	n, err = writer.Write([]byte("this is not json"))
	require.NoError(t, err) // Write itself doesn't error
	assert.Equal(t, len("this is not json"), n)
	assert.Equal(t, 1, store.GetPutCalls()) // No new Put call
}

// TestUnit_Logging_StoreWriter_Concurrency ensures no deadlocks and all logs are processed
// when writing concurrently from multiple goroutines.
func TestUnit_Logging_StoreWriter_Concurrency_NoLockingIssues(t *testing.T) {
	store := newMockStore()
	// store.putDelay = 1 * time.Millisecond // Optional: uncomment to simulate more work in store

	ctx := context.Background()
	writer := logging.StoreWriter(ctx, store, "concurrent-cluster", "concurrent-account")

	// Configure a logger to use our storeWriter
	// We use io.Discard for one of the sinks if NewLogger adds a default stdout,
	// or ensure only our writer is used.
	// The provided NewLogger adds os.Stdout if no sinks are provided.
	// If we provide our sink, it should be the only one unless we add others.
	// For this test, we only care about the storeWriter.
	// To avoid polluting test output, we can try to make storeWriter the only sink.
	// However, NewLogger might have its own logic. Let's assume we can control it.

	// Create a logger that ONLY uses our storeWriter.
	// If NewLogger insists on adding os.Stdout, we might need a more complex setup
	// or accept console output during tests.
	// For simplicity, let's assume NewLogger uses only provided sinks if any.
	// If not, we might need to pass io.Discard as another sink to quiet stdout.
	logger, err := logging.NewLogger(
		logging.WithSink(writer), // This should make `writer` the primary sink
		logging.WithLevel("debug"),
		// To ensure no console output if NewLogger adds one by default:
		// logging.WithSink(io.Discard), // This would make MultiWriter send to both
		// The current NewLogger logic: if sinks are empty, adds os.Stdout.
		// If sinks are provided, it uses them. So WithSink(writer) is enough.
	)
	require.NoError(t, err)

	numGoroutines := 50
	logsPerGoroutine := 100
	totalLogs := numGoroutines * logsPerGoroutine

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < logsPerGoroutine; j++ {
				logger.Info().
					Int("routine", routineID).
					Int("log_num", j).
					Str("test_data", fmt.Sprintf("message %d from routine %d", j, routineID)).
					Msg("Concurrent log message")

				// Brief pause to allow goroutines to interleave more naturally,
				// though not strictly necessary as scheduling will handle it.
				// time.Sleep(time.Microsecond)
			}
		}(i)
	}

	wg.Wait()

	// Check that all logs made it to the store
	assert.Equal(t, totalLogs, store.GetPutCalls(), "Mismatch in expected number of Put calls")
	assert.NoError(t, store.GetLastError(), "Mock store encountered an error during Put")

	// Optional: Further inspect a few metrics if needed
	// metrics := store.GetMetrics()
	// require.Len(t, metrics, totalLogs)
	// For example, check one metric's content
	// t.Logf("Sample metric labels: %+v", metrics[0].Labels)
}

// TestUnit_Logging_StoreWriter_WithRealLogger demonstrates integration with NewLogger
func TestUnit_Logging_StoreWriter_WithRealLogger(t *testing.T) {
	store := newMockStore()
	ctx := context.Background()
	clusterName := "real-logger-cluster"
	cloudAccountID := "real-logger-account"

	storeSink := logging.StoreWriter(ctx, store, clusterName, cloudAccountID)

	// Create logger with our storeSink and potentially others (like a filtered stdout)
	// The NewLogger by default adds a filtered stdout if no sinks are provided.
	// If we provide a sink, it uses that.
	logger, err := logging.NewLogger(
		logging.WithSink(storeSink), // Our custom sink
		// logging.WithSink(os.Stdout), // If you also want console output for debugging this test
		logging.WithLevel("info"),
		logging.WithAttrs(func(c zerolog.Context) zerolog.Context {
			return c.Str("global_attr", "global_value")
		}),
	)
	require.NoError(t, err)

	logger.Info().Str("event", "user_login").Msg("User logged in successfully")
	logger.Error().Err(fmt.Errorf("database connection failed")).Msg("Failed to connect to DB")

	assert.Equal(t, 2, store.GetPutCalls(), "Should have two log entries in the store")

	metrics := store.GetMetrics()
	require.Len(t, metrics, 2)

	// Check first metric (info log)
	metric1 := metrics[0]
	assert.Equal(t, "log", metric1.MetricName)
	assert.Equal(t, "User logged in successfully", metric1.Value)
	assert.Equal(t, clusterName, metric1.ClusterName)
	assert.Equal(t, "user_login", metric1.Labels["event"])
	assert.Equal(t, "global_value", metric1.Labels["global_attr"]) // From WithAttrs
	assert.NotEmpty(t, metric1.Labels[zerolog.TimestampFieldName])
	assert.NotEmpty(t, metric1.Labels["version"]) // From NewLogger defaults

	// Check second metric (error log)
	metric2 := metrics[1]
	assert.Equal(t, "log", metric2.MetricName)
	assert.Equal(t, "Failed to connect to DB", metric2.Value)
	assert.Equal(t, clusterName, metric2.ClusterName)
	assert.Equal(t, "database connection failed", metric2.Labels[zerolog.ErrorFieldName])
	assert.Equal(t, "global_value", metric2.Labels["global_attr"])
}

// TestUnit_Logging_StoreWriter_HandlesMalformedJSON ensures that malformed JSON
// is consumed but doesn't call Put on the store.
func TestUnit_Logging_StoreWriter_HandlesMalformedJSON(t *testing.T) {
	store := newMockStore()
	ctx := context.Background()
	writer := logging.StoreWriter(ctx, store, "malformed-cluster", "malformed-account")

	malformedJSON := []byte("{not json")
	n, err := writer.Write(malformedJSON)

	assert.NoError(t, err, "Write should not return an error for malformed JSON as per current implementation")
	assert.Equal(t, len(malformedJSON), n, "Should consume all bytes of malformed JSON")
	assert.Equal(t, 0, store.GetPutCalls(), "Put should not be called for malformed JSON")
}

// TestUnit_Logging_StoreWriter_HandlesMissingFields gracefully handles logs with missing standard fields.
func TestUnit_Logging_StoreWriter_HandlesMissingFields(t *testing.T) {
	store := newMockStore()
	ctx := context.Background()
	writer := logging.StoreWriter(ctx, store, "missing-fields-cluster", "missing-fields-account")

	// Log missing message and timestamp
	logData := map[string]interface{}{
		zerolog.LevelFieldName: "warn",
		"custom_data":          "only_this",
	}
	jsonData, err := json.Marshal(logData)
	require.NoError(t, err)

	n, err := writer.Write(jsonData)
	require.NoError(t, err)
	assert.Equal(t, len(jsonData), n)

	assert.Equal(t, 1, store.GetPutCalls())
	metrics := store.GetMetrics()
	require.Len(t, metrics, 1)

	metric := metrics[0]
	assert.Equal(t, "", metric.Value, "Message should be empty string if not present") // Default for msg
	assert.True(t, metric.TimeStamp.IsZero(), "Timestamp should be zero value if not parsable/present")
	assert.Equal(t, "warn", metric.Labels[zerolog.LevelFieldName])
	assert.Equal(t, "only_this", metric.Labels["custom_data"])
}
