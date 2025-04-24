// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package webhook provides kubernetes webhook resource business logic.
package webhook

import (
	"context"
	"strings"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	config "github.com/cloudzero/cloudzero-agent/app/config/insights-controller"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/handler"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/helper"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/hook"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	AppsGroup  = "apps"
	CoreGroup  = ""
	BatchGroup = "batch"
	Version1   = "v1"
)

// metricWebhookEventTotal is a Prometheus counter vector that tracks the total number of webhook events.
// It can be used to record metrics for webhook events, categorized by the following labels:
// - "kind_group": The API group of the resource.
// - "kind_version": The API version of the resource.
// - "kind_resource": The kind of the resource.
// - "operation": The operation performed (e.g., "create", "update", "delete").
//
// Usage:
// To increment the counter for a specific combination of labels, use the WithLabelValues method
// to specify the label values, followed by the Inc method to increment the counter.
//
// Example:
// metricWebhookEventTotal.WithLabelValues("apps", "v1", "deployments", "create").Inc()
//
// Ensure that the metric is registered with the Prometheus registry before use:
// prometheus.MustRegister(metricWebhookEventTotal)
var (
	webhookStatsOnce        sync.Once
	metricWebhookEventTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cloudzero_webhook_event_total",
			Help: "Total number of webhook events filterable by kind_group, kind_version, kind_resource, and operation",
		},
		[]string{"kind_group", "kind_version", "kind_resource", "operation"},
	)
)

// WebhookController defines the interface for handling AdmissionReview requests.
// It provides a method to process incoming requests and route them to the appropriate handler
// based on the resource's Group, Version, and Resource (G/V/R). If no specific handler is registered,
// the default handler is used to allow the request by default.
type WebhookController interface {
	// Review processes an AdmissionReview request and determines whether the request
	// should be allowed or denied. It dispatches the request to the appropriate handler
	// based on the resource's G/V/R or falls back to the default handler if no specific
	// handler is registered.
	//
	// Parameters:
	//   - ctx: The context for the request, used for cancellation and deadlines.
	//   - ar: The AdmissionReview containing details about the resource and operation.
	//
	// Returns:
	//   - *types.AdmissionResponse: The result of the admission process, indicating
	//     whether the request is allowed or denied.
	//   - error: Any error encountered during processing.
	Review(ctx context.Context, ar *types.AdmissionReview) (*types.AdmissionResponse, error)
}

type webhookController struct {
	defaultHandler *hook.Handler
	dispatch       map[string]map[string]map[string]*hook.Handler
	enabled        bool
	settings       *config.Settings
	clock          types.TimeProvider
}

// NewWebhookFactory creates and initializes a new WebhookController instance.
// It sets up default handlers, registers resource-specific handlers, and initializes metrics.
//
// Parameters:
//   - store: A ResourceStore instance used for storing resources.
//   - settings: A Settings instance containing configuration for the webhook.
//   - clock: A TimeProvider instance for time-related operations.
//
// Returns:
//   - *WebhookController: The initialized WebhookController instance.
//   - error: An error if the initialization fails.
func NewWebhookFactory(store types.ResourceStore, settings *config.Settings, clock types.TimeProvider) (WebhookController, error) {
	wc := &webhookController{
		dispatch: make(map[string]map[string]map[string]*hook.Handler),
		defaultHandler: &hook.Handler{
			ObjectCreator: helper.NewDynamicObjectCreator(),
			Create:        allowAlways,
			Update:        allowAlways,
			Delete:        allowAlways,
			Connect:       allowAlways,
			Store:         store,
		},
		enabled:  true,
		settings: settings,
		clock:    clock,
	}

	// expose metrics for resource kinds
	webhookStatsOnce.Do(func() {
		prometheus.MustRegister(
			metricWebhookEventTotal,
		)
	})

	// register each resource you care about:
	wc.register(AppsGroup, Version1, "deployments", handler.NewDeploymentHandler(store, settings, clock))   // ✓ check
	wc.register(AppsGroup, Version1, "statefulsets", handler.NewStatefulsetHandler(store, settings, clock)) // ✓ check
	wc.register(AppsGroup, Version1, "daemonsets", handler.NewDaemonSetHandler(store, settings, clock))     // ✓ check
	wc.register(CoreGroup, Version1, "pods", handler.NewPodHandler(store, settings, clock))                 // ✓ check
	wc.register(CoreGroup, Version1, "namespaces", handler.NewNamespaceHandler(store, settings, clock))     // ✓ check
	wc.register(CoreGroup, Version1, "nodes", handler.NewNodeHandler(store, settings, clock))               // ✓ check
	wc.register(BatchGroup, Version1, "jobs", handler.NewJobHandler(store, settings, clock))                // ✓ check
	wc.register(BatchGroup, Version1, "cronjobs", handler.NewCronJobHandler(store, settings, clock))        // ✓ check

	// Note: handlers beyond this point will not capture labels/annotations and will later be used to correlate resources
	// to cloud resources (providerID) - to the pod using them.
	wc.register(AppsGroup, Version1, "replicasets", handler.NewReplicaSetHandler(store, settings, clock))                                          // ✓ new
	wc.register(CoreGroup, Version1, "services", handler.NewServiceHandler(store, settings, clock))                                                // ✓ new
	wc.register(CoreGroup, Version1, "persistentvolumeclaims", handler.NewPersistentVolumeClaimHandler(store, settings, clock))                    // ✓ new
	wc.register("networking.k8s.io", Version1, "ingresses", handler.NewIngressHandler(store, settings, clock))                                     // ✓ new
	wc.register("apiextensions.k8s.io", Version1, "customresourcedefinitions", handler.NewCustomResourceDefinitionHandler(store, settings, clock)) // ✓ new
	wc.register("gateway.networking.k8s.io", Version1, "gateways", handler.NewGatewayHandler(store, settings, clock))                              // ✓ new

	return wc, nil
}

// Execute processes an incoming AdmissionRequest and determines the appropriate
// handler to execute based on the resource's Group, Version, and Resource (G/V/R).
// If no specific handler is registered for the resource, the default handler is used,
// which allows the request by default.
//
// Parameters:
//   - ctx: The context for the request, used for cancellation and deadlines.
//   - req: The AdmissionRequest containing details about the resource and operation.
//
// Returns:
//   - *hook.Result: The result of the admission process, indicating whether the request is allowed or denied.
//   - error: Any error encountered during processing.
func (wc *webhookController) Review(ctx context.Context, ar *types.AdmissionReview) (*types.AdmissionResponse, error) {
	grp := ar.RequestGVR.Group
	ver := ar.RequestGVR.Version
	res := strings.ToLower(ar.RequestGVR.Resource) // e.g. "pods"
	op := string(ar.Operation)

	//
	metricWebhookEventTotal.WithLabelValues(grp, ver, res, op).Inc()

	// no specific handler -> allow by default
	if !wc.registered(grp, ver, res) {
		return wc.defaultHandler.Execute(ctx, ar)
	}

	processor := wc.dispatch[grp][ver][res]
	return processor.Execute(ctx, ar)
}

// allowAlways is a trivial AdmitFunc that always allows the admission request.
func allowAlways(_ context.Context, _ *types.AdmissionReview, _ metav1.Object) (*types.AdmissionResponse, error) {
	return &types.AdmissionResponse{Allowed: true}, nil
}

// register associates a resource-specific Handler with a given group, version,
// and resource in the WebhookController's dispatch map.
//
// Parameters:
//   - group: The API group of the resource (e.g., "apps").
//   - version: The API version of the resource (e.g., "v1").
//   - resource: The resource kind (e.g., "pods").
//   - h: The Handler instance to associate with the resource.
func (wc *webhookController) register(group, version, resource string, h *hook.Handler) {
	if wc.dispatch[group] == nil {
		wc.dispatch[group] = make(map[string]map[string]*hook.Handler)
	}
	if wc.dispatch[group][version] == nil {
		wc.dispatch[group][version] = make(map[string]*hook.Handler)
	}
	wc.dispatch[group][version][resource] = h
}

// registered checks if a handler is registered for a specific group, version,
// and resource in the WebhookController's dispatch map.
//
// Parameters:
//   - grp: The API group of the resource.
//   - ver: The API version of the resource.
//   - res: The resource kind.
//
// Returns:
//   - bool: True if a handler is registered, false otherwise.
func (wc *webhookController) registered(grp, ver, res string) bool {
	if versMap, ok := wc.dispatch[grp]; ok {
		if resMap, ok := versMap[ver]; ok {
			_, ok := resMap[res]
			return ok
		}
	}
	return false
}
