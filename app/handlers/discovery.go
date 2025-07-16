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

const (
	// descriptionKey is the key for description in OpenAPI spec
	descriptionKey = "description"
	// applicationJSON is the content type for JSON responses
	applicationJSON = "application/json"
)

// DiscoveryAPI provides API discovery endpoints required by HPA controller
type DiscoveryAPI struct {
	api.Service
}

// NewDiscoveryAPI creates a new API discovery handler
func NewDiscoveryAPI() *DiscoveryAPI {
	d := &DiscoveryAPI{
		Service: api.Service{
			APIName: "discovery",
			Mounts:  map[string]*chi.Mux{},
		},
	}
	d.Service.Mounts["/"] = d.Routes()
	return d
}

func (d *DiscoveryAPI) Register(app server.Server) error {
	if err := d.Service.Register(app); err != nil {
		return err
	}
	return nil
}

func (d *DiscoveryAPI) Routes() *chi.Mux {
	r := chi.NewRouter()

	// API discovery endpoints (required by HPA controller) - ONLY these endpoints
	r.Get("/apis", d.listAPIGroups)
	r.Get("/openapi/v2", d.getOpenAPISpec)

	return r
}

// listAPIGroups returns the API groups available for discovery
func (d *DiscoveryAPI) listAPIGroups(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log.Ctx(ctx).Info().Str("path", r.URL.Path).Msg("DiscoveryAPI: listAPIGroups called")

	// Return the API group list that includes custom.metrics.k8s.io
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
			},
		},
	}

	w.Header().Set("Content-Type", applicationJSON)
	if err := json.NewEncoder(w).Encode(apiGroupList); err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to encode API group list")
		request.Reply(r, w, "failed to encode API group list", http.StatusInternalServerError)
		return
	}
}

// getOpenAPISpec returns a minimal OpenAPI v2 specification
func (d *DiscoveryAPI) getOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log.Ctx(ctx).Info().Str("path", r.URL.Path).Msg("DiscoveryAPI: getOpenAPISpec called")

	// Return a minimal OpenAPI v2 spec for our custom metrics API
	openAPISpec := map[string]interface{}{
		"swagger": "2.0",
		"info": map[string]interface{}{
			"title":   "CloudZero Custom Metrics API",
			"version": "v1beta1",
		},
		"paths": map[string]interface{}{
			"/apis/custom.metrics.k8s.io/v1beta1/": map[string]interface{}{
				"get": map[string]interface{}{
					descriptionKey: "List available custom metrics",
					"produces":     []string{"application/json"},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							descriptionKey: "List of available metrics",
						},
					},
				},
			},
			"/apis/custom.metrics.k8s.io/v1beta1/namespaces/{namespace}/pods/{metric}": map[string]interface{}{
				"get": map[string]interface{}{
					descriptionKey: "Get custom metric for pods in namespace",
					"produces":     []string{"application/json"},
					"parameters": []map[string]interface{}{
						{
							"name":     "namespace",
							"in":       "path",
							"required": true,
							"type":     "string",
						},
						{
							"name":     "metric",
							"in":       "path",
							"required": true,
							"type":     "string",
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							descriptionKey: "Metric value list",
						},
					},
				},
			},
		},
	}

	w.Header().Set("Content-Type", applicationJSON)
	if err := json.NewEncoder(w).Encode(openAPISpec); err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to encode OpenAPI spec")
		request.Reply(r, w, "failed to encode OpenAPI spec", http.StatusInternalServerError)
		return
	}
}
