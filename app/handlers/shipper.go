// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package handlers

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-obvious/server"
	"github.com/go-obvious/server/api"
	"github.com/go-obvious/server/request"

	"github.com/cloudzero/cloudzero-insights-controller/app/domain"
)

type ShipperAPI struct {
	api.Service
	shipper *domain.MetricShipper
}

func NewShipperAPI(base string, d *domain.MetricShipper) *ShipperAPI {
	a := &ShipperAPI{
		shipper: d,
		Service: api.Service{
			APIName: "shipper",
			Mounts:  map[string]*chi.Mux{},
		},
	}
	a.Service.Mounts[base] = a.Routes()
	return a
}

func (a *ShipperAPI) Register(app server.Server) error {
	if err := a.Service.Register(app); err != nil {
		return err
	}
	return nil
}

func (a *ShipperAPI) Routes() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/", a.GetMetrics)
	return r
}

func (a *ShipperAPI) GetMetrics(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	stats, err := a.shipper.GetStatus()
	if err != nil {
		request.Reply(r, w, err, http.StatusInternalServerError)
		return
	}

	request.Reply(r, w, stats, http.StatusNoContent)
}
