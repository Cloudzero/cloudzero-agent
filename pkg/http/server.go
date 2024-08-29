// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package http

import (
	"fmt"
	"net/http"
	"time"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/healthz"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/hook"
)

type RouteSegment struct {
	Route string
	Hook  hook.Handler
}

// NewServer creates and return a http.Server
func NewServer(cfg *config.Settings, routes ...RouteSegment) *http.Server {
	ah := handler()
	mux := http.NewServeMux()
	for _, route := range routes {
		mux.Handle(route.Route, ah.Serve(route.Hook))
	}
	// Internal routes
	mux.Handle("/healthz", healthz.NewHealthz().EndpointHandler())

	return &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout),
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout),
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeout),
	}
}
