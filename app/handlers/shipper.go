// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package handlers

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-obvious/server"
	"github.com/go-obvious/server/api"

	"github.com/cloudzero/cloudzero-agent/app/domain/shipper"
)

type ShipperAPI struct {
	api.Service
	shipper *shipper.MetricShipper
}

func NewShipperAPI(base string, d *shipper.MetricShipper) server.API {
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
	r.Get("/metrics", a.shipper.GetMetricHandler().ServeHTTP)
	return r
}
