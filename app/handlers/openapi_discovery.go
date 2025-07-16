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
)

const (
	// openAPIApplicationJSON is the content type for JSON responses
	openAPIApplicationJSON = "application/json"
	// openAPIDescriptionKey is the key for description in OpenAPI spec
	openAPIDescriptionKey = "description"
	// openAPITypeKey is the key for type in OpenAPI spec
	openAPITypeKey = "type"
	// openAPIStringValue is the string value for type in OpenAPI spec
	openAPIStringValue = "string"
	// openAPIArrayValue is the array value for type in OpenAPI spec
	openAPIArrayValue = "array"
	// openAPIObjectValue is the object value for type in OpenAPI spec
	openAPIObjectValue = "object"
	// openAPIRefKey is the key for references in OpenAPI spec
	openAPIRefKey = "$ref"
)

// OpenAPIv2DiscoveryAPI provides OpenAPI v2 specification endpoint for API discovery
type OpenAPIv2DiscoveryAPI struct {
	api.Service
}

// NewOpenAPIv2DiscoveryAPI creates a new OpenAPI v2 discovery handler that mounts to the specified path
func NewOpenAPIv2DiscoveryAPI(path string) *OpenAPIv2DiscoveryAPI {
	h := &OpenAPIv2DiscoveryAPI{
		Service: api.Service{
			APIName: "openapi-v2-discovery",
			Mounts:  map[string]*chi.Mux{},
		},
	}
	h.Service.Mounts[path] = h.Routes()
	return h
}

func (h *OpenAPIv2DiscoveryAPI) Register(app server.Server) error {
	if err := h.Service.Register(app); err != nil {
		return err
	}
	return nil
}

func (h *OpenAPIv2DiscoveryAPI) Routes() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/", h.getOpenAPIv2Spec)
	return r
}

// getOpenAPIv2Spec returns an OpenAPI v2 specification
func (h *OpenAPIv2DiscoveryAPI) getOpenAPIv2Spec(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log.Ctx(ctx).Info().Str("path", r.URL.Path).Msg("OpenAPIv2DiscoveryAPI: getOpenAPIv2Spec called")

	// Return an aggregated OpenAPI v2 spec for our custom metrics API
	openAPISpec := map[string]interface{}{
		"swagger": "2.0",
		"info": map[string]interface{}{
			"title":   "CloudZero Custom Metrics API",
			"version": "v1beta1",
		},
		"host":     "",
		"basePath": "/",
		"schemes":  []string{"https"},
		"consumes": []string{openAPIApplicationJSON},
		"produces": []string{openAPIApplicationJSON},
		"paths": map[string]interface{}{
			"/apis/custom.metrics.k8s.io/v1beta1": map[string]interface{}{
				"get": map[string]interface{}{
					openAPIDescriptionKey: "List available custom metrics",
					"operationId":         "listCustomMetrics",
					"produces":            []string{openAPIApplicationJSON},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							openAPIDescriptionKey: "List of available metrics",
							"schema": map[string]interface{}{
								openAPITypeKey: openAPIArrayValue,
								"items": map[string]interface{}{
									openAPITypeKey: openAPIStringValue,
								},
							},
						},
					},
					"tags": []string{"custom.metrics.k8s.io_v1beta1"},
				},
			},
			"/apis/custom.metrics.k8s.io/v1beta1/namespaces/{namespace}/pods/{metric}": map[string]interface{}{
				"get": map[string]interface{}{
					openAPIDescriptionKey: "Get custom metric for pods in namespace",
					"operationId":         "getCustomMetricForPods",
					"produces":            []string{openAPIApplicationJSON},
					"parameters": []map[string]interface{}{
						{
							"name":                "namespace",
							"in":                  "path",
							openAPIDescriptionKey: "object name and auth scope, such as for teams and projects",
							"required":            true,
							openAPITypeKey:        openAPIStringValue,
						},
						{
							"name":                "metric",
							"in":                  "path",
							openAPIDescriptionKey: "the name of the metric",
							"required":            true,
							openAPITypeKey:        openAPIStringValue,
						},
					},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							openAPIDescriptionKey: "Metric value list",
							"schema": map[string]interface{}{
								openAPIRefKey: "#/definitions/io.k8s.metrics.pkg.apis.custom_metrics.v1beta1.MetricValueList",
							},
						},
					},
					"tags": []string{"custom.metrics.k8s.io_v1beta1"},
				},
			},
		},
		"definitions": map[string]interface{}{
			"io.k8s.metrics.pkg.apis.custom_metrics.v1beta1.MetricValue": map[string]interface{}{
				openAPITypeKey: openAPIObjectValue,
				"properties": map[string]interface{}{
					"describedObject": map[string]interface{}{
						openAPIRefKey: "#/definitions/io.k8s.api.core.v1.ObjectReference",
					},
					"metricName": map[string]interface{}{
						openAPITypeKey: openAPIStringValue,
					},
					"timestamp": map[string]interface{}{
						openAPIRefKey: "#/definitions/io.k8s.apimachinery.pkg.apis.meta.v1.Time",
					},
					"value": map[string]interface{}{
						openAPIRefKey: "#/definitions/io.k8s.apimachinery.pkg.api.resource.Quantity",
					},
				},
			},
			"io.k8s.metrics.pkg.apis.custom_metrics.v1beta1.MetricValueList": map[string]interface{}{
				openAPITypeKey: openAPIObjectValue,
				"properties": map[string]interface{}{
					"items": map[string]interface{}{
						openAPITypeKey: openAPIArrayValue,
						"items": map[string]interface{}{
							openAPIRefKey: "#/definitions/io.k8s.metrics.pkg.apis.custom_metrics.v1beta1.MetricValue",
						},
					},
					"metadata": map[string]interface{}{
						openAPIRefKey: "#/definitions/io.k8s.apimachinery.pkg.apis.meta.v1.ListMeta",
					},
				},
			},
		},
		"tags": []map[string]interface{}{
			{
				"name":                "custom.metrics.k8s.io_v1beta1",
				openAPIDescriptionKey: "Custom metrics API for autoscaling",
			},
		},
	}

	// Support both JSON and protobuf as per Kubernetes API docs
	acceptHeader := r.Header.Get("Accept")
	if acceptHeader == "application/com.github.proto-openapi.spec.v2@v1.0+protobuf" {
		w.Header().Set("Content-Type", "application/com.github.proto-openapi.spec.v2@v1.0+protobuf")
		// For now, return JSON even for protobuf requests since we don't have protobuf encoding
	} else {
		w.Header().Set("Content-Type", openAPIApplicationJSON)
	}

	if err := json.NewEncoder(w).Encode(openAPISpec); err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to encode OpenAPI v2 spec")
		request.Reply(r, w, "failed to encode OpenAPI v2 spec", http.StatusInternalServerError)
		return
	}
}

// OpenAPIv3DiscoveryAPI provides OpenAPI v3 specification endpoint
type OpenAPIv3DiscoveryAPI struct {
	api.Service
}

// NewOpenAPIv3DiscoveryAPI creates a new OpenAPI v3 discovery handler that mounts to the specified path
func NewOpenAPIv3DiscoveryAPI(path string) *OpenAPIv3DiscoveryAPI {
	h := &OpenAPIv3DiscoveryAPI{
		Service: api.Service{
			APIName: "openapi-v3-discovery",
			Mounts:  map[string]*chi.Mux{},
		},
	}
	h.Service.Mounts[path] = h.Routes()
	return h
}

func (h *OpenAPIv3DiscoveryAPI) Register(app server.Server) error {
	if err := h.Service.Register(app); err != nil {
		return err
	}
	return nil
}

func (h *OpenAPIv3DiscoveryAPI) Routes() *chi.Mux {
	r := chi.NewRouter()
	// This serves OpenAPI v3 discovery at /openapi/v3
	r.Get("/", h.getOpenAPIv3Discovery)
	return r
}

// getOpenAPIv3Discovery returns OpenAPI v3 discovery information
func (h *OpenAPIv3DiscoveryAPI) getOpenAPIv3Discovery(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log.Ctx(ctx).Info().Str("path", r.URL.Path).Msg("OpenAPIv3DiscoveryAPI: getOpenAPIv3Discovery called")

	// Return OpenAPI v3 discovery with available API groups
	discovery := map[string]interface{}{
		"paths": map[string]interface{}{
			"apis/custom.metrics.k8s.io/v1beta1": map[string]interface{}{
				"serverRelativeURL": "/openapi/v3/apis/custom.metrics.k8s.io/v1beta1",
			},
		},
	}

	w.Header().Set("Content-Type", openAPIApplicationJSON)
	if err := json.NewEncoder(w).Encode(discovery); err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to encode OpenAPI v3 discovery")
		request.Reply(r, w, "failed to encode OpenAPI v3 discovery", http.StatusInternalServerError)
		return
	}
}
