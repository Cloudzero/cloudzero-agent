// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package middleware

import (
	"net/http"
	"sync"
	"time"
)

// ErrorRateTracker maintains a bounded, time-windowed view of HTTP response
// statuses so a health check can ask "is the rate of 5xx responses on this
// endpoint exceeding some threshold?" — without hair-trigger flapping on a
// single transient error.
//
// The tracker classifies any status code in [500, 600) as a failure. All other
// codes count as a success.
//
// Two views are exposed:
//
//   - Healthy: instantaneous rate, ideal for readiness. As soon as failures
//     age out of the window or are drowned in successes, it returns true.
//     This lets a pod that's recovered rejoin the Service quickly.
//   - HealthyWithinCooldown: sticky-latch view, ideal for liveness. Once an
//     evaluation returns unhealthy, the tracker remembers the time, and
//     HealthyWithinCooldown keeps returning false until the cooldown has
//     elapsed since the last unhealthy evaluation. This prevents a
//     readiness-eviction → no-traffic → window-ages-out → readmit flap loop
//     and ensures a persistently-bad pod actually gets restarted.
//
// Pruning of old events happens lazily on each Record and Healthy call. No
// background goroutine.
type ErrorRateTracker struct {
	mu              sync.Mutex
	window          time.Duration
	now             func() time.Time
	events          []errorRateEvent
	lastUnhealthyAt time.Time
}

type errorRateEvent struct {
	at      time.Time
	failure bool
}

// NewErrorRateTracker creates a tracker that retains response events for the
// given window. Events older than the window are pruned on the next call.
func NewErrorRateTracker(window time.Duration) *ErrorRateTracker {
	return NewErrorRateTrackerWithClock(window, time.Now)
}

// NewErrorRateTrackerWithClock is a test seam for injecting a deterministic
// clock.
func NewErrorRateTrackerWithClock(window time.Duration, now func() time.Time) *ErrorRateTracker {
	return &ErrorRateTracker{
		window: window,
		now:    now,
	}
}

// Record adds an observation of one HTTP response to the window.
func (t *ErrorRateTracker) Record(statusCode int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := t.now()
	t.prune(now)
	t.events = append(t.events, errorRateEvent{
		at:      now,
		failure: isFailureStatus(statusCode),
	})
}

// Healthy reports whether the tracker's recent window is within acceptable
// bounds: the failure rate must be at or below rateThreshold, or the absolute
// number of failures must be below minFailures. The minFailures floor prevents
// a single transient error (or a small burst on a quiet endpoint) from
// marking the endpoint unhealthy.
//
// This is the instantaneous view; use it for readiness probes so a pod that
// has recovered rejoins the Service as soon as its recent responses are
// healthy again.
func (t *ErrorRateTracker) Healthy(rateThreshold float64, minFailures int) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.evaluateLocked(rateThreshold, minFailures)
}

// HealthyWithinCooldown is the sticky-latch view on top of Healthy. It returns
// false if either:
//   - the instantaneous evaluation is unhealthy, or
//   - some earlier evaluation tripped the threshold within the last cooldown.
//
// Use this for liveness probes so a pod that has been demonstrated unhealthy
// cannot look healthy again just because traffic stopped arriving (which
// would cause the sliding window to empty and Healthy to flip back to true).
// The latch guarantees that if a pod is persistently bad, liveness eventually
// fires and the pod is restarted instead of flapping Ready / NotReady.
func (t *ErrorRateTracker) HealthyWithinCooldown(rateThreshold float64, minFailures int, cooldown time.Duration) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.evaluateLocked(rateThreshold, minFailures) {
		return false
	}
	if t.lastUnhealthyAt.IsZero() {
		return true
	}
	return t.now().Sub(t.lastUnhealthyAt) > cooldown
}

// evaluateLocked performs the instantaneous health check and, as a side
// effect, records `now` as the most recent unhealthy evaluation when the
// verdict is false. Caller must hold t.mu.
func (t *ErrorRateTracker) evaluateLocked(rateThreshold float64, minFailures int) bool {
	now := t.now()
	t.prune(now)

	total := len(t.events)
	if total == 0 {
		return true
	}

	failures := 0
	for _, e := range t.events {
		if e.failure {
			failures++
		}
	}

	if failures < minFailures {
		return true
	}
	if float64(failures)/float64(total) <= rateThreshold {
		return true
	}
	t.lastUnhealthyAt = now
	return false
}

// prune drops events whose timestamp is older than the window boundary.
// Caller must hold t.mu.
func (t *ErrorRateTracker) prune(now time.Time) {
	cutoff := now.Add(-t.window)
	i := 0
	for ; i < len(t.events); i++ {
		if t.events[i].at.After(cutoff) {
			break
		}
	}
	t.events = t.events[i:]
}

// statusCodesEnd marks the first value past the HTTP 5xx range. Any code in
// [StatusInternalServerError, statusCodesEnd) is treated as a server-side
// failure by the tracker; anything at or above this (non-standard) is not.
const statusCodesEnd = 600

func isFailureStatus(code int) bool {
	return code >= http.StatusInternalServerError && code < statusCodesEnd
}

// ErrorRateMiddleware returns HTTP middleware that records the outgoing
// response status on each request into the given ErrorRateTracker. The
// tracker can then be read by a readiness / liveness health check to decide
// whether this instance is still serving requests successfully.
//
// If the wrapped handler does not call WriteHeader, the response defaults to
// 200 OK (matching Go's net/http behavior) and is recorded as a non-failure.
//
// If the wrapped handler panics, the panic is first recorded as a 500 in the
// tracker (so the health check sees it) and then re-raised for the outer
// panic-recovery middleware to translate into an HTTP 500 response. Without
// this, a panicking endpoint would never register as failing on the health
// check.
func ErrorRateMiddleware(tracker *ErrorRateTracker) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			defer func() {
				if rvr := recover(); rvr != nil {
					tracker.Record(http.StatusInternalServerError)
					panic(rvr)
				}
			}()
			next.ServeHTTP(rec, r)
			tracker.Record(rec.status)
		})
	}
}
