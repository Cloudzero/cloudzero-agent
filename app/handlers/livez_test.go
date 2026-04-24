// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package handlers_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/go-obvious/server/test"
	"github.com/stretchr/testify/assert"

	"github.com/cloudzero/cloudzero-agent/app/handlers"
	"github.com/cloudzero/cloudzero-agent/app/http/middleware"
)

const (
	livezTestRateThreshold = 0.20
	livezTestMinFailures   = 3
	livezTestWindow        = 60 * time.Second
	livezTestCooldown      = 5 * time.Minute
)

// TestLivez_HealthyWhenTrackerEmpty: a fresh tracker with no recorded responses
// should produce an OK response.
func TestLivez_HealthyWhenTrackerEmpty(t *testing.T) {
	tracker := middleware.NewErrorRateTracker(livezTestWindow)

	api := handlers.NewLivezAPI("/", tracker, livezTestRateThreshold, livezTestMinFailures, livezTestCooldown)

	req, _ := http.NewRequest("GET", "/", nil)
	resp, err := test.InvokeService(api.Service, "/", *req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode,
		"empty tracker should produce a healthy livez response")
}

// TestLivez_UnavailableWhenInstantaneouslyUnhealthy: after the tracker has
// recorded enough 5xx to trip the rate threshold, /livez should return 503.
func TestLivez_UnavailableWhenInstantaneouslyUnhealthy(t *testing.T) {
	tracker := middleware.NewErrorRateTracker(livezTestWindow)
	for i := 0; i < livezTestMinFailures; i++ {
		tracker.Record(http.StatusInternalServerError)
	}

	api := handlers.NewLivezAPI("/", tracker, livezTestRateThreshold, livezTestMinFailures, livezTestCooldown)

	req, _ := http.NewRequest("GET", "/", nil)
	resp, err := test.InvokeService(api.Service, "/", *req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode,
		"after threshold-crossing failures, livez should be 503")
}

// TestLivez_LatchKeepsUnhealthyAfterEventsAgeOut: the key property that
// distinguishes /livez from /healthz — once the tracker has observed an
// unhealthy evaluation, subsequent livez calls stay 503 through the cooldown
// window even if the sliding window's events have all aged out.
func TestLivez_LatchKeepsUnhealthyAfterEventsAgeOut(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	tracker := middleware.NewErrorRateTrackerWithClock(livezTestWindow, clock)

	for i := 0; i < livezTestMinFailures; i++ {
		tracker.Record(http.StatusInternalServerError)
	}

	api := handlers.NewLivezAPI("/", tracker, livezTestRateThreshold, livezTestMinFailures, livezTestCooldown)

	// Trip the latch.
	{
		req, _ := http.NewRequest("GET", "/", nil)
		resp, err := test.InvokeService(api.Service, "/", *req)
		assert.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	}

	// Age out the sliding window's events but stay within the cooldown.
	now = now.Add(livezTestWindow + time.Second)

	req, _ := http.NewRequest("GET", "/", nil)
	resp, err := test.InvokeService(api.Service, "/", *req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode,
		"events aged out but cooldown still active — livez must remain 503 so liveness can fire")
}

// TestLivez_HealthyAfterCooldownElapses: once the cooldown has elapsed with no
// further unhealthy evaluations, the latch releases and /livez goes back to
// 200. This matches the "restart and rejoin" case — a fresh pod should not
// inherit the previous pod's latch state because the tracker is per-process.
func TestLivez_HealthyAfterCooldownElapses(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	tracker := middleware.NewErrorRateTrackerWithClock(livezTestWindow, clock)

	for i := 0; i < livezTestMinFailures; i++ {
		tracker.Record(http.StatusInternalServerError)
	}

	api := handlers.NewLivezAPI("/", tracker, livezTestRateThreshold, livezTestMinFailures, livezTestCooldown)

	// Trip the latch via a probe call.
	{
		req, _ := http.NewRequest("GET", "/", nil)
		resp, err := test.InvokeService(api.Service, "/", *req)
		assert.NoError(t, err)
		resp.Body.Close()
		assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode)
	}

	// Advance past the cooldown — with no new failures, the latch releases.
	now = now.Add(livezTestCooldown + time.Second)

	req, _ := http.NewRequest("GET", "/", nil)
	resp, err := test.InvokeService(api.Service, "/", *req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode,
		"after cooldown elapses with no new unhealthy evaluations, livez should release")
}
