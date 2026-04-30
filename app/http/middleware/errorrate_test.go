// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package middleware_test

import (
	"errors"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/go-obvious/server/healthz"
	"github.com/stretchr/testify/assert"

	"github.com/cloudzero/cloudzero-agent/app/http/middleware"
)

const (
	testWindow        = 60 * time.Second
	testRateThreshold = 0.20
	testMinFailures   = 3
)

func TestErrorRateTracker_Empty_IsHealthy(t *testing.T) {
	tracker := middleware.NewErrorRateTracker(testWindow)

	assert.True(t, tracker.Healthy(testRateThreshold, testMinFailures),
		"a tracker with no recorded responses should be healthy")
}

func TestErrorRateTracker_SuccessesOnly_IsHealthy(t *testing.T) {
	tracker := middleware.NewErrorRateTracker(testWindow)

	for i := 0; i < 20; i++ {
		tracker.Record(http.StatusNoContent)
	}

	assert.True(t, tracker.Healthy(testRateThreshold, testMinFailures),
		"a tracker containing only 2xx responses should be healthy")
}

func TestErrorRateTracker_SingleFailure_IsHealthyBelowMinFailures(t *testing.T) {
	tracker := middleware.NewErrorRateTracker(testWindow)

	tracker.Record(http.StatusInternalServerError)

	assert.True(t, tracker.Healthy(testRateThreshold, testMinFailures),
		"one 5xx response is below the minFailures floor and should not trip unhealthy")
}

func TestErrorRateTracker_ThreeFailuresOutOfThree_IsUnhealthy(t *testing.T) {
	tracker := middleware.NewErrorRateTracker(testWindow)

	for i := 0; i < 3; i++ {
		tracker.Record(http.StatusInternalServerError)
	}

	assert.False(t, tracker.Healthy(testRateThreshold, testMinFailures),
		"3 of 3 responses being 5xx is 100% failure rate, well above 20% threshold")
}

func TestErrorRateTracker_MixedBelowRateThreshold_IsHealthy(t *testing.T) {
	tracker := middleware.NewErrorRateTracker(testWindow)

	// 3 failures out of 20 = 15%, below the 20% threshold
	for i := 0; i < 3; i++ {
		tracker.Record(http.StatusInternalServerError)
	}
	for i := 0; i < 17; i++ {
		tracker.Record(http.StatusNoContent)
	}

	assert.True(t, tracker.Healthy(testRateThreshold, testMinFailures),
		"3 failures in 20 requests is 15%, below the 20% threshold")
}

func TestErrorRateTracker_MixedAboveRateThreshold_IsUnhealthy(t *testing.T) {
	tracker := middleware.NewErrorRateTracker(testWindow)

	// 5 failures out of 20 = 25%, above the 20% threshold, at least 3 absolute
	for i := 0; i < 5; i++ {
		tracker.Record(http.StatusInternalServerError)
	}
	for i := 0; i < 15; i++ {
		tracker.Record(http.StatusNoContent)
	}

	assert.False(t, tracker.Healthy(testRateThreshold, testMinFailures),
		"5 failures in 20 requests is 25%, above the 20% threshold")
}

func TestErrorRateTracker_StatusCodeClassification(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantFail   bool
	}{
		{"2xx success", http.StatusNoContent, false},
		{"3xx redirect", http.StatusMovedPermanently, false},
		{"4xx client error", http.StatusBadRequest, false},
		{"5xx server error", http.StatusInternalServerError, true},
		{"503 unavailable", http.StatusServiceUnavailable, true},
		{"599", 599, true},
		{"600 non-standard", 600, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := middleware.NewErrorRateTracker(testWindow)

			// Record three of the code under test so we clear the minFailures floor
			// if it is classified as a failure.
			for i := 0; i < 3; i++ {
				tracker.Record(tt.statusCode)
			}

			// With 3 of 3 being the code under test:
			//   - failure code → 100% failure rate → unhealthy
			//   - non-failure code → 0% failure rate → healthy
			assert.Equal(t, !tt.wantFail, tracker.Healthy(testRateThreshold, testMinFailures),
				"status code %d classification mismatch", tt.statusCode)
		})
	}
}

func TestErrorRateTracker_OldEventsAgeOut(t *testing.T) {
	// Custom clock so we can advance time deterministically.
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }

	tracker := middleware.NewErrorRateTrackerWithClock(testWindow, clock)

	// Record a burst of failures at t=0
	for i := 0; i < 10; i++ {
		tracker.Record(http.StatusInternalServerError)
	}
	assert.False(t, tracker.Healthy(testRateThreshold, testMinFailures),
		"immediately after a burst of 5xx, tracker should be unhealthy")

	// Advance past the window — all old events should age out.
	now = now.Add(testWindow + time.Second)

	assert.True(t, tracker.Healthy(testRateThreshold, testMinFailures),
		"after the window elapses with no new events, tracker should be healthy again")
}

const testCooldown = 5 * time.Minute

// TestErrorRateTracker_StickyLatch_StaysUnhealthyWithinCooldown verifies that
// HealthyWithinCooldown keeps reporting unhealthy for `cooldown` after the
// last evaluation that tripped the threshold, even after the sliding window's
// own events have aged out. This is the liveness-probe view: once the pod has
// been demonstrated to fail, don't let it look healthy again until the probe
// has had time to restart it.
func TestErrorRateTracker_StickyLatch_StaysUnhealthyWithinCooldown(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }

	tracker := middleware.NewErrorRateTrackerWithClock(testWindow, clock)

	// Record 3 × 500 — tripping the threshold. Must actually call
	// HealthyWithinCooldown (or Healthy) once so the latch gets set; the
	// tracker doesn't speculate about unhealthy state until asked.
	for i := 0; i < 3; i++ {
		tracker.Record(http.StatusInternalServerError)
	}
	assert.False(t, tracker.HealthyWithinCooldown(testRateThreshold, testMinFailures, testCooldown),
		"immediately after 3 × 500, HealthyWithinCooldown should be false")

	// Advance past the sliding window. The raw events age out, so
	// the instantaneous Healthy() view goes back to healthy.
	now = now.Add(testWindow + time.Second)
	assert.True(t, tracker.Healthy(testRateThreshold, testMinFailures),
		"after window elapses, instantaneous Healthy() should be true (events aged out)")

	// But the cooldown hasn't elapsed yet — HealthyWithinCooldown should
	// still latch unhealthy.
	assert.False(t, tracker.HealthyWithinCooldown(testRateThreshold, testMinFailures, testCooldown),
		"within cooldown after last unhealthy evaluation, HealthyWithinCooldown must remain false")
}

// TestErrorRateTracker_StickyLatch_ClearsAfterCooldown verifies the latch
// releases once the cooldown elapses with no further unhealthy evaluations.
func TestErrorRateTracker_StickyLatch_ClearsAfterCooldown(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }

	tracker := middleware.NewErrorRateTrackerWithClock(testWindow, clock)

	for i := 0; i < 3; i++ {
		tracker.Record(http.StatusInternalServerError)
	}
	// Trip the latch.
	assert.False(t, tracker.HealthyWithinCooldown(testRateThreshold, testMinFailures, testCooldown))

	// Advance past both the window AND the cooldown. No new failures.
	now = now.Add(testCooldown + time.Second)

	assert.True(t, tracker.HealthyWithinCooldown(testRateThreshold, testMinFailures, testCooldown),
		"after cooldown elapses with no new unhealthy evaluations, HealthyWithinCooldown should be true")
}

// TestErrorRateTracker_StickyLatch_RefreshesOnContinuedFailures verifies that
// each new unhealthy evaluation pushes the latch forward, so a pod that keeps
// failing never slides back to healthy via the cooldown expiring.
func TestErrorRateTracker_StickyLatch_RefreshesOnContinuedFailures(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }

	tracker := middleware.NewErrorRateTrackerWithClock(testWindow, clock)

	// Record failures and trip the latch.
	for i := 0; i < 3; i++ {
		tracker.Record(http.StatusInternalServerError)
	}
	assert.False(t, tracker.HealthyWithinCooldown(testRateThreshold, testMinFailures, testCooldown))

	// Advance past window, then record fresh failures. The latch's
	// lastUnhealthyAt should move to "now," so the cooldown is measured from
	// this newer timestamp.
	now = now.Add(testWindow + time.Second)
	for i := 0; i < 3; i++ {
		tracker.Record(http.StatusInternalServerError)
	}
	assert.False(t, tracker.HealthyWithinCooldown(testRateThreshold, testMinFailures, testCooldown),
		"newly recorded failures refresh the latch")

	// Advance a little less than the cooldown — still within latch.
	now = now.Add(testCooldown - time.Second)
	assert.False(t, tracker.HealthyWithinCooldown(testRateThreshold, testMinFailures, testCooldown),
		"still within cooldown measured from the most recent unhealthy evaluation")

	// Advance past cooldown — latch releases.
	now = now.Add(2 * time.Second)
	assert.True(t, tracker.HealthyWithinCooldown(testRateThreshold, testMinFailures, testCooldown))
}

func TestErrorRateTracker_ConcurrentAccess_DoesNotRace(t *testing.T) {
	// Meaningful only under `go test -race`, but runs cleanly under normal test too.
	tracker := middleware.NewErrorRateTracker(testWindow)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				tracker.Record(http.StatusNoContent)
				_ = tracker.Healthy(testRateThreshold, testMinFailures)
			}
		}()
	}
	wg.Wait()
}

func TestErrorRateMiddleware_RecordsStatusCodesIntoTracker(t *testing.T) {
	tracker := middleware.NewErrorRateTracker(testWindow)

	// Handler that always returns 500 so we can drive the tracker into an
	// unhealthy state purely through the middleware.
	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})

	wrapped := middleware.ErrorRateMiddleware(tracker)(handler)

	// One failure is below the minFailures floor — tracker should still be healthy.
	for i := 0; i < 1; i++ {
		rr := httptestRecorder()
		req, _ := http.NewRequest("POST", "/", nil)
		wrapped.ServeHTTP(rr, req)
	}
	assert.True(t, tracker.Healthy(testRateThreshold, testMinFailures),
		"one failure recorded through the middleware should be below minFailures")

	// Two more failures — now we have 3 of 3 all failing, clearly over the 20% threshold.
	for i := 0; i < 2; i++ {
		rr := httptestRecorder()
		req, _ := http.NewRequest("POST", "/", nil)
		wrapped.ServeHTTP(rr, req)
	}
	assert.False(t, tracker.Healthy(testRateThreshold, testMinFailures),
		"after 3 failing requests via the middleware, the tracker should be unhealthy")
}

// TestErrorRateMiddleware_PanicRecordsFailure verifies that when the wrapped
// handler panics, the panic is still recorded as a 500 in the tracker before
// it propagates to the outer panic middleware. Otherwise a panicking endpoint
// would never register as failing on the health check.
func TestErrorRateMiddleware_PanicRecordsFailure(t *testing.T) {
	tracker := middleware.NewErrorRateTracker(testWindow)

	handler := http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {
		panic("boom")
	})
	wrapped := middleware.ErrorRateMiddleware(tracker)(handler)

	// Fire testMinFailures panicking requests. Each must propagate its panic
	// out past the middleware (we catch it here in the test via recover()),
	// but the tracker must have logged each as a failure before the panic
	// escapes — otherwise we'd still see Healthy=true after all three.
	for i := 0; i < testMinFailures; i++ {
		func() {
			defer func() { _ = recover() }()
			rr := httptestRecorder()
			req, _ := http.NewRequest("POST", "/", nil)
			wrapped.ServeHTTP(rr, req)
		}()
	}

	assert.False(t, tracker.Healthy(testRateThreshold, testMinFailures),
		"panicking handlers must be recorded as failures so the health check can see them")
}

func TestErrorRateMiddleware_RecordsDefaultStatusWhenHandlerDoesNotCallWriteHeader(t *testing.T) {
	// Go's http package treats a handler that writes a body without calling
	// WriteHeader as an implicit 200. Make sure the tracker agrees — otherwise
	// "silent" success responses would go uncounted.
	tracker := middleware.NewErrorRateTracker(testWindow)

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// No WriteHeader, no body; effectively an empty 200 OK response.
		_ = w
	})
	wrapped := middleware.ErrorRateMiddleware(tracker)(handler)

	for i := 0; i < 5; i++ {
		rr := httptestRecorder()
		req, _ := http.NewRequest("POST", "/", nil)
		wrapped.ServeHTTP(rr, req)
	}
	assert.True(t, tracker.Healthy(testRateThreshold, testMinFailures),
		"successful responses (no explicit WriteHeader) should be recorded as non-failures")
}

// httptestRecorder is a thin wrapper so tests don't need to import httptest
// at every call site.
func httptestRecorder() *httptestRR {
	return &httptestRR{headers: http.Header{}}
}

type httptestRR struct {
	status  int
	headers http.Header
}

func (r *httptestRR) Header() http.Header         { return r.headers }
func (r *httptestRR) Write(b []byte) (int, error) { return len(b), nil }
func (r *httptestRR) WriteHeader(code int)        { r.status = code }

// TestErrorRateTracker_HealthzIntegration verifies the pattern used in
// collector/main.go: register a health-check function that reads the tracker,
// and have go-obvious's healthz aggregator report failure when the tracker
// reports unhealthy. This is the chain that actually makes /healthz return
// 503 on the running collector.
//
// Note: go-obvious's healthz package is a process-wide singleton, so this
// test uses a unique check name per run to avoid cross-contamination.
func TestErrorRateTracker_HealthzIntegration(t *testing.T) {
	tracker := middleware.NewErrorRateTracker(testWindow)

	checkName := "test-errorrate-tracker-integration-" + t.Name()
	healthz.Register(checkName, func() error {
		if !tracker.Healthy(testRateThreshold, testMinFailures) {
			return errors.New("collector 5xx rate exceeds threshold")
		}
		return nil
	})

	// With an empty tracker, the check passes.
	assert.NoError(t, runNamedCheck(checkName),
		"with no recorded failures, the registered health check should pass")

	// 3 × 500 → tracker reports unhealthy → check returns error.
	for i := 0; i < 3; i++ {
		tracker.Record(http.StatusInternalServerError)
	}
	assert.Error(t, runNamedCheck(checkName),
		"after 3 × 500, the registered health check should fail, which is what makes /healthz return 503")
}

// runNamedCheck invokes just one of the healthz singleton's registered checks
// by name, so the integration test doesn't depend on whatever other tests in
// the process may have registered.
func runNamedCheck(name string) error {
	c, ok := healthz.NewHealthz().(interface {
		Checks() map[string]healthz.HealthCheck
	})
	if !ok {
		return errors.New("healthz singleton did not expose Checks()")
	}
	fn, present := c.Checks()[name]
	if !present {
		return errors.New("check " + name + " not registered")
	}
	return fn()
}
