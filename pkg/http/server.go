// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/domain/healthz"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/http/hook"
)

type RouteSegment struct {
	Route string
	Hook  http.Handler
}

type AdmissionRouteSegment struct {
	Route string
	Hook  hook.Handler
}

// NewServer creates and return a http.Server
func NewServer(cfg *config.Settings, routes []RouteSegment, admissionRoutes ...AdmissionRouteSegment) *http.Server {
	ah := handler()
	mux := http.NewServeMux()
	for _, route := range admissionRoutes {
		mux.Handle(route.Route, ah.Serve(route.Hook))
	}
	// Internal routes
	mux.Handle("/healthz", healthz.NewHealthz().EndpointHandler())
	mux.Handle("/metrics", promhttp.Handler())

	for _, route := range routes {
		mux.Handle(route.Route, route.Hook)
	}

	handler := MetricsMiddlewareWrapper(mux)
	handler = LoggingMiddlewareWrapper(handler)

	return &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      handler,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout),
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout),
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeout),
	}
}
