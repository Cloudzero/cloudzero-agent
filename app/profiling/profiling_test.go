// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package profiling_test

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudzero/cloudzero-agent/app/profiling"
)

// waitForPort polls addr until it either accepts a connection or the deadline
// is exceeded. Returns true if the port is reachable within the timeout.
func waitForPort(addr string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://" + addr + "/debug/pprof/")
		if err == nil {
			resp.Body.Close()
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// portOpen returns true if the given address is currently serving HTTP.
func portOpen(addr string) bool {
	resp, err := http.Get("http://" + addr + "/debug/pprof/")
	if err != nil {
		return false
	}
	resp.Body.Close()
	return true
}

// TestStart_Disabled_DoesNothing verifies that calling Start with Enabled:false
// starts no HTTP listener and returns a callable (no-op) stop function.
func TestStart_Disabled_DoesNothing(t *testing.T) {
	stop := profiling.Start(profiling.Options{}) // Enabled defaults to false
	require.NotNil(t, stop)

	// Default port must NOT be listening.
	assert.False(t, portOpen("localhost:6060"), "port 6060 should not be listening when disabled")

	// stop must not panic.
	stop()
}

// TestStart_Enabled_StartsPprofServer verifies that enabling profiling causes
// the pprof HTTP server to start and serve the index page.
func TestStart_Enabled_StartsPprofServer(t *testing.T) {
	const port = 16061
	addr := fmt.Sprintf("localhost:%d", port)

	stop := profiling.Start(profiling.Options{
		Enabled: true,
		Port:    port,
	})
	require.NotNil(t, stop)
	t.Cleanup(stop)

	require.True(t, waitForPort(addr, 2*time.Second), "pprof server did not start on port %d", port)

	resp, err := http.Get("http://" + addr + "/debug/pprof/")
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

// TestStart_WritesHeapDumps verifies that heap profile files are written to
// opts.Dir at approximately HeapInterval.
func TestStart_WritesHeapDumps(t *testing.T) {
	const port = 16062
	dir := t.TempDir()

	stop := profiling.Start(profiling.Options{
		Enabled:      true,
		Port:         port,
		Dir:          dir,
		HeapInterval: 200 * time.Millisecond,
	})
	require.NotNil(t, stop)
	t.Cleanup(stop)

	// Poll for at least 2 heap dumps. HeapInterval is 200ms, so under a
	// healthy runner this completes in ~400ms; the generous deadline absorbs
	// scheduling jitter on loaded CI without making the happy path slow.
	var matches []string
	require.Eventually(t, func() bool {
		m, err := filepath.Glob(filepath.Join(dir, "heap-*.pb.gz"))
		if err != nil {
			return false
		}
		matches = m
		return len(matches) >= 2
	}, 5*time.Second, 50*time.Millisecond, "expected at least 2 heap dump files within deadline")

	for _, m := range matches {
		info, err := os.Stat(m)
		require.NoError(t, err)
		assert.Greater(t, info.Size(), int64(0), "heap file %s should not be empty", m)
	}
}

// TestStart_WritesCPUProfiles verifies that CPU profile files are written to
// opts.Dir after the first CPUInterval elapses.
func TestStart_WritesCPUProfiles(t *testing.T) {
	const port = 16063
	dir := t.TempDir()

	stop := profiling.Start(profiling.Options{
		Enabled:     true,
		Port:        port,
		Dir:         dir,
		CPUInterval: 300 * time.Millisecond,
		CPUDuration: 100 * time.Millisecond,
	})
	require.NotNil(t, stop)
	t.Cleanup(stop)

	// Poll for at least one CPU capture. First capture finishes at
	// CPUInterval+CPUDuration = 400ms; the generous deadline absorbs runner
	// scheduling jitter.
	var matches []string
	require.Eventually(t, func() bool {
		m, err := filepath.Glob(filepath.Join(dir, "cpu-*.pb.gz"))
		if err != nil {
			return false
		}
		matches = m
		return len(matches) >= 1
	}, 5*time.Second, 50*time.Millisecond, "expected at least 1 CPU profile file within deadline")

	for _, m := range matches {
		info, err := os.Stat(m)
		require.NoError(t, err)
		assert.Greater(t, info.Size(), int64(0), "cpu file %s should not be empty", m)
	}
}

// TestStart_StopShutsDownServer verifies that calling the stop function causes
// the pprof HTTP server to stop accepting connections.
func TestStart_StopShutsDownServer(t *testing.T) {
	const port = 16064
	addr := fmt.Sprintf("localhost:%d", port)

	stop := profiling.Start(profiling.Options{
		Enabled: true,
		Port:    port,
	})
	require.NotNil(t, stop)

	require.True(t, waitForPort(addr, 2*time.Second), "pprof server did not start on port %d", port)

	stop()

	// Give the server a moment to finish shutting down.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !portOpen(addr) {
			return // success
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Errorf("pprof server on port %d is still reachable after stop()", port)
}
