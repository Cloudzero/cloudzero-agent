// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-obvious/server"
	"github.com/go-obvious/server/api"
	"github.com/go-obvious/server/request"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/metrics/pkg/apis/custom_metrics/v1beta1"

	"github.com/cloudzero/cloudzero-agent/app/domain"
)

const (
	// metricName is the name of the custom metric for HPA scaling
	metricName = "czo_cost_metrics_shipping_progress"
	// errorMsgFailedToGetMetricValue is the error message when failing to get metric value
	errorMsgFailedToGetMetricValue = "failed to get metric value"
	// apiVersion is the API version for custom metrics
	apiVersion = "custom.metrics.k8s.io/v1beta1"
	// contentTypeJSON is the content type for JSON responses
	contentTypeJSON = "application/json"
	// contentTypeHeader is the HTTP header for content type
	contentTypeHeader = "Content-Type"
	// kubernetesAPIVersion is the Kubernetes API version
	kubernetesAPIVersion = "v1"
	// rootPath is the root path for API endpoints
	rootPath = "/"
)

// CustomMetricsAPI implements the Kubernetes custom metrics API
type CustomMetricsAPI struct {
	api.Service
	collector *domain.MetricCollector
	k8sClient kubernetes.Interface
}

// NewCustomMetricsAPI creates a new custom metrics API handler
func NewCustomMetricsAPI(base string, collector *domain.MetricCollector, k8sClient kubernetes.Interface) *CustomMetricsAPI {
	a := &CustomMetricsAPI{
		collector: collector,
		k8sClient: k8sClient,
		Service: api.Service{
			APIName: "custom-metrics",
			Mounts:  map[string]*chi.Mux{},
		},
	}

	// Mount custom metrics API at its base path
	a.Service.Mounts[base] = a.Routes()

	return a
}

func (a *CustomMetricsAPI) Register(app server.Server) error {
	if err := a.Service.Register(app); err != nil {
		return err
	}
	return nil
}

func (a *CustomMetricsAPI) Routes() *chi.Mux {
	r := chi.NewRouter()

	// Custom metrics API routes
	r.Get("/", a.listCustomMetrics)
	r.Get("/namespaces/{namespace}/pods", a.listPodsMetrics)
	r.Get("/namespaces/{namespace}/pods/{pod}/{metric}", a.getCustomMetricForPod)
	r.Get("/namespaces/{namespace}/pods/{metric}", a.getCustomMetricForPods)

	return r
}

// CustomMetricsWithDiscoveryAPI combines custom metrics API with discovery endpoints
type CustomMetricsWithDiscoveryAPI struct {
	api.Service
	customMetricsAPI *CustomMetricsAPI
}

// NewCustomMetricsWithDiscoveryAPI creates a new API that includes both custom metrics and discovery endpoints
func NewCustomMetricsWithDiscoveryAPI(base string, collector *domain.MetricCollector, k8sClient kubernetes.Interface) *CustomMetricsWithDiscoveryAPI {
	customMetricsAPI := &CustomMetricsAPI{
		collector: collector,
		k8sClient: k8sClient,
	}

	a := &CustomMetricsWithDiscoveryAPI{
		customMetricsAPI: customMetricsAPI,
		Service: api.Service{
			APIName: "custom-metrics-with-discovery",
			Mounts:  map[string]*chi.Mux{},
		},
	}

	// Mount custom metrics API at its base path
	a.Service.Mounts[base] = customMetricsAPI.Routes()

	// Mount discovery endpoints at root level
	a.Service.Mounts["/apis"] = a.createAPIGroupsRoute()
	a.Service.Mounts["/openapi/v2"] = a.createOpenAPIv2Route()

	return a
}

func (a *CustomMetricsWithDiscoveryAPI) Register(app server.Server) error {
	if err := a.Service.Register(app); err != nil {
		return err
	}
	return nil
}

// createAPIGroupsRoute creates a router for the /apis endpoint
func (a *CustomMetricsWithDiscoveryAPI) createAPIGroupsRoute() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/", a.customMetricsAPI.getAPIGroups)
	return r
}

// createOpenAPIv2Route creates a router for the /openapi/v2 endpoint
func (a *CustomMetricsWithDiscoveryAPI) createOpenAPIv2Route() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/", a.customMetricsAPI.getOpenAPIv2Spec)
	return r
}

// listCustomMetrics returns the list of available custom metrics
func (a *CustomMetricsAPI) listCustomMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Return APIResourceList according to Kubernetes custom metrics API v1beta1 spec
	apiResourceList := metav1.APIResourceList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "APIResourceList",
			APIVersion: "v1",
		},
		GroupVersion: apiVersion,
		APIResources: []metav1.APIResource{
			{
				Name:       "pods/" + metricName,
				Namespaced: true,
				Kind:       "MetricValueList",
				Verbs:      []string{"get"},
			},
		},
	}

	w.Header().Set(contentTypeHeader, contentTypeJSON)
	if err := json.NewEncoder(w).Encode(apiResourceList); err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to encode metrics list")
		request.Reply(r, w, "failed to encode metrics list", http.StatusInternalServerError)
		return
	}
}

// listPodsMetrics returns the list of available metrics for pods
func (a *CustomMetricsAPI) listPodsMetrics(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	metrics := []string{metricName}

	w.Header().Set(contentTypeHeader, contentTypeJSON)
	if err := json.NewEncoder(w).Encode(metrics); err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to encode pods metrics list")
		request.Reply(r, w, "failed to encode pods metrics list", http.StatusInternalServerError)
		return
	}
}

// getCustomMetricForPod returns the custom metric value for a specific pod
func (a *CustomMetricsAPI) getCustomMetricForPod(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	namespace := chi.URLParam(r, "namespace")
	pod := chi.URLParam(r, "pod")
	metric := chi.URLParam(r, "metric")

	if metric != metricName {
		request.Reply(r, w, fmt.Sprintf("metric %s not found", metric), http.StatusNotFound)
		return
	}

	// Special case: if pod is "*", return metrics for all pods (HPA compatibility)
	if pod == "*" {
		a.getCustomMetricForPods(w, r)
		return
	}

	// Get the current metric value from Prometheus
	value, err := a.getCurrentMetricValue()
	if err != nil {
		log.Ctx(ctx).Err(err).Msg(errorMsgFailedToGetMetricValue)
		request.Reply(r, w, errorMsgFailedToGetMetricValue, http.StatusInternalServerError)
		return
	}

	metricValue := &v1beta1.MetricValue{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MetricValue",
			APIVersion: apiVersion,
		},
		DescribedObject: corev1.ObjectReference{
			Kind:       "Pod",
			Namespace:  namespace,
			Name:       pod,
			APIVersion: "v1",
		},
		MetricName: metric,
		Timestamp:  metav1.NewTime(time.Now()),
		Value:      *value,
		Selector:   nil, // No label selector for this metric
	}

	w.Header().Set(contentTypeHeader, contentTypeJSON)
	if err := json.NewEncoder(w).Encode(metricValue); err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to encode metric value")
		request.Reply(r, w, "failed to encode metric value", http.StatusInternalServerError)
		return
	}
}

// getCustomMetricForPods returns the custom metric values for multiple pods
func (a *CustomMetricsAPI) getCustomMetricForPods(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	namespace := chi.URLParam(r, "namespace")
	metric := chi.URLParam(r, "metric")

	if metric != metricName {
		request.Reply(r, w, fmt.Sprintf("metric %s not found", metric), http.StatusNotFound)
		return
	}

	// Get the current metric value from Prometheus
	value, err := a.getCurrentMetricValue()
	if err != nil {
		log.Ctx(ctx).Err(err).Msg(errorMsgFailedToGetMetricValue)
		request.Reply(r, w, errorMsgFailedToGetMetricValue, http.StatusInternalServerError)
		return
	}

	// Create metric values for pods
	var items []v1beta1.MetricValue

	if a.k8sClient != nil {
		// Query actual pods from the namespace that match our deployment
		pods, err := a.k8sClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: "app.kubernetes.io/component=aggregator",
		})
		if err != nil {
			log.Ctx(ctx).Err(err).Msg("failed to list pods")
			request.Reply(r, w, "failed to list pods", http.StatusInternalServerError)
			return
		}

		// Create metric values for all running aggregator pods
		items = make([]v1beta1.MetricValue, 0, len(pods.Items))
		for _, pod := range pods.Items {
			// Only include running pods
			if pod.Status.Phase == corev1.PodRunning {
				items = append(items, v1beta1.MetricValue{
					DescribedObject: corev1.ObjectReference{
						Kind:       "Pod",
						Namespace:  namespace,
						Name:       pod.Name,
						APIVersion: "v1",
					},
					MetricName: metric,
					Timestamp:  metav1.NewTime(time.Now()),
					Value:      *value,
					Selector:   nil, // No label selector for this metric
				})
			}
		}
	} else {
		// Fallback: return a single metric value with wildcard name when k8s client is not available
		items = []v1beta1.MetricValue{
			{
				DescribedObject: corev1.ObjectReference{
					Kind:       "Pod",
					Namespace:  namespace,
					Name:       "*",
					APIVersion: "v1",
				},
				MetricName: metric,
				Timestamp:  metav1.NewTime(time.Now()),
				Value:      *value,
				Selector:   nil, // No label selector for this metric
			},
		}
	}

	metricValueList := &v1beta1.MetricValueList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "MetricValueList",
			APIVersion: apiVersion,
		},
		ListMeta: metav1.ListMeta{
			ResourceVersion: "1",
		},
		Items: items,
	}

	w.Header().Set(contentTypeHeader, contentTypeJSON)
	if err := json.NewEncoder(w).Encode(metricValueList); err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to encode metric value list")
		request.Reply(r, w, "failed to encode metric value list", http.StatusInternalServerError)
		return
	}
}

// getCurrentMetricValue retrieves the current value of the shipping progress metric from Prometheus
func (a *CustomMetricsAPI) getCurrentMetricValue() (*resource.Quantity, error) {
	// Get metrics from the default Prometheus registry
	metricFamilies, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		return nil, fmt.Errorf("failed to gather metrics: %w", err)
	}

	// Find the czo_cost_metrics_shipping_progress metric
	for _, mf := range metricFamilies {
		if mf.GetName() == metricName {
			metrics := mf.GetMetric()
			if len(metrics) > 0 {
				// Get the gauge value
				gauge := metrics[0].GetGauge()
				if gauge != nil {
					value := gauge.GetValue()
					// Convert to Kubernetes resource.Quantity
					// Parse the value as a decimal string to preserve precision
					quantityStr := fmt.Sprintf("%.0fm", value*1000)
					quantity, err := resource.ParseQuantity(quantityStr)
					if err != nil {
						return nil, fmt.Errorf("failed to parse quantity %s: %w", quantityStr, err)
					}
					return &quantity, nil
				}
			}
		}
	}

	// If metric not found, return 0
	quantity := resource.NewMilliQuantity(0, resource.DecimalSI)
	return quantity, nil
}

// getAPIGroups returns the API groups available for discovery
func (a *CustomMetricsAPI) getAPIGroups(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log.Ctx(ctx).Info().Str("path", r.URL.Path).Msg("CustomMetricsAPI: getAPIGroups called")

	// Return the API group list that includes custom.metrics.k8s.io
	apiGroupList := metav1.APIGroupList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "APIGroupList",
			APIVersion: kubernetesAPIVersion,
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
				ServerAddressByClientCIDRs: []metav1.ServerAddressByClientCIDR{
					{
						ClientCIDR:    "0.0.0.0/0",
						ServerAddress: "",
					},
				},
			},
		},
	}

	w.Header().Set(contentTypeHeader, contentTypeJSON)
	if err := json.NewEncoder(w).Encode(apiGroupList); err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to encode API group list")
		request.Reply(r, w, "failed to encode API group list", http.StatusInternalServerError)
		return
	}
}

// getOpenAPIv2Spec returns an OpenAPI v2 specification
func (a *CustomMetricsAPI) getOpenAPIv2Spec(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log.Ctx(ctx).Info().Str("path", r.URL.Path).Msg("CustomMetricsAPI: getOpenAPIv2Spec called")

	// Return minimal OpenAPI v2 spec for our custom metrics API
	openAPISpec := map[string]interface{}{
		"swagger": "2.0",
		"info": map[string]interface{}{
			"title":   "CloudZero Custom Metrics API",
			"version": "v1beta1",
		},
		"host":     "",
		"basePath": rootPath,
		"schemes":  []string{"https"},
		"consumes": []string{contentTypeJSON},
		"produces": []string{contentTypeJSON},
		"paths": map[string]interface{}{
			"/apis/custom.metrics.k8s.io/v1beta1": map[string]interface{}{
				"get": map[string]interface{}{
					"description": "List available custom metrics",
					"operationId": "listCustomMetrics",
					"produces":    []string{contentTypeJSON},
					"responses": map[string]interface{}{
						"200": map[string]interface{}{
							"description": "List of available metrics",
						},
					},
					"tags": []string{"custom.metrics.k8s.io_v1beta1"},
				},
			},
		},
		"tags": []map[string]interface{}{
			{
				"name":        "custom.metrics.k8s.io_v1beta1",
				"description": "Custom metrics API for autoscaling",
			},
		},
	}

	w.Header().Set(contentTypeHeader, contentTypeJSON)
	if err := json.NewEncoder(w).Encode(openAPISpec); err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to encode OpenAPI v2 spec")
		request.Reply(r, w, "failed to encode OpenAPI v2 spec", http.StatusInternalServerError)
		return
	}
}
