// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package handlers

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-obvious/server"
	"github.com/go-obvious/server/api"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PromMetricsAPI is a thin wrapper around the Prometheus HTTP handler to
// integrate with the go-obvious server API.
type PromMetricsAPI struct {
	api.Service
}

func NewPromMetricsAPI(base string) *PromMetricsAPI {
	a := &PromMetricsAPI{
		Service: api.Service{
			APIName: "metrics",
			Mounts:  map[string]*chi.Mux{},
		},
	}
	a.Service.Mounts[base] = a.Routes()
	return a
}

func (a *PromMetricsAPI) Register(app server.Server) error {
	if err := a.Service.Register(app); err != nil {
		return err
	}
	return nil
}

func (a *PromMetricsAPI) Routes() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/", promhttp.Handler().ServeHTTP)

	return r
}
