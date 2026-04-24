// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package handlers

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-obvious/server"
	"github.com/go-obvious/server/api"

	"github.com/cloudzero/cloudzero-agent/app/http/middleware"
)

// LivezAPI exposes an HTTP endpoint suitable for use as a Kubernetes liveness
// probe. It differs from the general /healthz endpoint in one important way:
// it has sticky-latch semantics.
//
// /healthz reflects the instantaneous state of the ErrorRateTracker: if
// traffic stops arriving or recent failures age out of the sliding window,
// it immediately goes back to reporting healthy. That's the right behavior
// for readiness — a pod that has recovered should rejoin the Service
// quickly.
//
// /livez, in contrast, reports unhealthy for `cooldown` after the last
// time the tracker's evaluation returned unhealthy, even if subsequent
// evaluations (with events aged out) would pass. This prevents a
// readiness-eviction → no-traffic → window-ages-out → readmit flap loop:
// once the pod has been demonstrated to fail, liveness will fire and
// restart it instead.
type LivezAPI struct {
	// api.Service provides the foundational HTTP server infrastructure from go-obvious/server.
	api.Service

	tracker       *middleware.ErrorRateTracker
	rateThreshold float64
	minFailures   int
	cooldown      time.Duration
}

// NewLivezAPI constructs a LivezAPI bound to the given tracker and threshold
// parameters. The same tracker should be passed to the collector's request
// middleware so the two see the same stream of response statuses.
func NewLivezAPI(base string, tracker *middleware.ErrorRateTracker, rateThreshold float64, minFailures int, cooldown time.Duration) *LivezAPI {
	a := &LivezAPI{
		tracker:       tracker,
		rateThreshold: rateThreshold,
		minFailures:   minFailures,
		cooldown:      cooldown,
		Service: api.Service{
			APIName: "livez",
			Mounts:  map[string]*chi.Mux{},
		},
	}
	a.Mounts[base] = a.Routes()
	return a
}

// Register integrates the LivezAPI with the CloudZero Agent HTTP server.
func (a *LivezAPI) Register(app server.Server) error {
	return a.Service.Register(app)
}

// Routes configures HTTP request routing for the liveness endpoint.
func (a *LivezAPI) Routes() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/", a.GetLivez)
	return r
}

// GetLivez returns 200 OK if the tracker reports healthy with respect to the
// configured cooldown, and 503 Service Unavailable otherwise.
func (a *LivezAPI) GetLivez(w http.ResponseWriter, _ *http.Request) {
	if a.tracker.HealthyWithinCooldown(a.rateThreshold, a.minFailures, a.cooldown) {
		w.WriteHeader(http.StatusOK)
		return
	}
	http.Error(w, "error rate exceeded within cooldown", http.StatusServiceUnavailable)
}
