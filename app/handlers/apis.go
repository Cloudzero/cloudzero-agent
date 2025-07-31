// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-obvious/server"
	"github.com/go-obvious/server/api"
	"github.com/go-obvious/server/request"
	"github.com/rs/zerolog/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// APIsHandler provides the /apis endpoint for API discovery
type APIsHandler struct {
	api.Service
}

// NewAPIsHandler creates a new APIs handler that mounts to the specified path
func NewAPIsHandler(path string) *APIsHandler {
	h := &APIsHandler{
		Service: api.Service{
			APIName: "apis",
			Mounts:  map[string]*chi.Mux{},
		},
	}
	h.Service.Mounts[path] = h.Routes()
	return h
}

func (h *APIsHandler) Register(app server.Server) error {
	if err := h.Service.Register(app); err != nil {
		return err
	}
	return nil
}

func (h *APIsHandler) Routes() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/", h.listAPIGroups)
	return r
}

// listAPIGroups returns the API groups available for discovery
func (h *APIsHandler) listAPIGroups(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log.Ctx(ctx).Info().Str("path", r.URL.Path).Msg("APIsHandler: listAPIGroups called")

	// Return the API group list that includes custom.metrics.k8s.io
	// According to Kubernetes API docs, this should list all API groups supported by the cluster
	apiGroupList := metav1.APIGroupList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "APIGroupList",
			APIVersion: "v1",
		},
		Groups: []metav1.APIGroup{
			{
				Name: "custom.metrics.k8s.io",
				Versions: []metav1.GroupVersionForDiscovery{
					{
						GroupVersion: "custom.metrics.k8s.io/v1beta1",
						Version:      "v1beta1",
					},
				},
				PreferredVersion: metav1.GroupVersionForDiscovery{
					GroupVersion: "custom.metrics.k8s.io/v1beta1",
					Version:      "v1beta1",
				},
				// Include server address as required by some clients
				ServerAddressByClientCIDRs: []metav1.ServerAddressByClientCIDR{
					{
						ClientCIDR:    "0.0.0.0/0",
						ServerAddress: "",
					},
				},
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(apiGroupList); err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to encode API group list")
		request.Reply(r, w, "failed to encode API group list", http.StatusInternalServerError)
		return
	}
}
