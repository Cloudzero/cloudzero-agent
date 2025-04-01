// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package handlers provides HTTP handlers.
package handlers

import (
	"net/http"
	"net/http/pprof"
	rtprof "runtime/pprof"

	"github.com/go-chi/chi"
	"github.com/go-obvious/server"
	"github.com/go-obvious/server/api"
)

type ProfilingAPI struct {
	api.Service
}

// NewProfilingAPI creates a new handler for pprof endpoints
//
// The standard is for this API to live at /debug/pprof
func NewProfilingAPI(base string) *ProfilingAPI {
	a := &ProfilingAPI{
		Service: api.Service{
			APIName: "profiling",
			Mounts:  map[string]*chi.Mux{},
		},
	}
	a.Service.Mounts[base] = a.Routes()
	return a
}

func (a *ProfilingAPI) Register(app server.Server) error {
	if err := a.Service.Register(app); err != nil {
		return err
	}
	return nil
}

func (a *ProfilingAPI) Routes() *chi.Mux {
	r := chi.NewRouter()

	r.Get("/", pprof.Index)
	r.Get("/cmdline", pprof.Cmdline)
	r.Get("/profile", pprof.Profile)
	r.Get("/symbol", pprof.Symbol)
	r.Get("/trace", pprof.Trace)

	for _, profile := range rtprof.Profiles() {
		r.Get("/"+profile.Name(), func(w http.ResponseWriter, r *http.Request) {
			pprof.Handler(profile.Name()).ServeHTTP(w, r)
		})
	}

	return r
}
