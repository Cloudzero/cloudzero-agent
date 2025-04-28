// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package webhook provides kubernetes webhook resource business logic.
package webhook

import (
	"context"
	"strings"
	"sync"

	appsv1 "k8s.io/api/apps/v1"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	appsv1beta2 "k8s.io/api/apps/v1beta2"
	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	storagev1 "k8s.io/api/storage/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1beta1 "k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/handler"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/helper"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/hook"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/prometheus/client_golang/prometheus"
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
			Name: types.ObservabilityMetric("webhook_types_total"),
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
	// GetSupported returns a nested map structure that indicates the supported
	// Group-Version-Kind (GVK) combinations for the webhook controller.
	//
	// Returns:
	//   - map[string]map[string]map[string]metav1.Object: A nested map where the first
	//     key represents the API group, the second key represents the API version,
	//     and the third key represents the kind. The value is a metav1.Object
	//     representing the supported resource.
	GetSupported() map[string]map[string]map[string]metav1.Object

	// IsSupported checks whether a specific Group-Version-Kind (GVK)
	// combination is supported for GET operations by the webhook controller.
	//
	// Parameters:
	//   - g: The API group of the resource.
	//   - v: The API version of the resource.
	//   - k: The kind of the resource.
	//
	// Returns:
	//   - bool: True if the specified GVK combination is supported for GET operations,
	//     otherwise false.
	IsSupported(g, v, k string) bool

	// GetConfigurationAccessor retrieves the configuration accessor for a specific
	// Group-Version-Kind (GVK) combination, if it is registered.
	//
	// Parameters:
	//   - g: The API group of the resource.
	//   - v: The API version of the resource.
	//   - k: The kind of the resource.
	//
	// Returns:
	//   - config.ConfigAccessor: The configuration accessor for the specified GVK,
	//     or nil if the GVK is not registered.
	GetConfigurationAccessor(g, v, k string) config.ConfigAccessor

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

	// Apps API
	wc.register(types.GroupApps, types.V1, types.KindDeployment, handler.NewDeploymentHandler(store, settings, clock, &appsv1.Deployment{}))
	wc.register(types.GroupApps, types.V1Beta2, types.KindDeployment, handler.NewDeploymentHandler(store, settings, clock, &appsv1beta2.Deployment{}))
	wc.register(types.GroupApps, types.V1Beta1, types.KindDeployment, handler.NewDeploymentHandler(store, settings, clock, &appsv1beta1.Deployment{}))
	wc.register(types.GroupApps, types.V1, types.KindStatefulSet, handler.NewStatefulSetHandler(store, settings, clock, &appsv1.StatefulSet{}))
	wc.register(types.GroupApps, types.V1Beta2, types.KindStatefulSet, handler.NewStatefulSetHandler(store, settings, clock, &appsv1beta2.StatefulSet{}))
	wc.register(types.GroupApps, types.V1Beta1, types.KindStatefulSet, handler.NewStatefulSetHandler(store, settings, clock, &appsv1beta1.StatefulSet{}))
	wc.register(types.GroupApps, types.V1, types.KindDaemonSet, handler.NewDaemonSetHandler(store, settings, clock, &appsv1.DaemonSet{}))
	wc.register(types.GroupApps, types.V1Beta2, types.KindDaemonSet, handler.NewDaemonSetHandler(store, settings, clock, &appsv1beta2.DaemonSet{}))
	wc.register(types.GroupApps, types.V1, types.KindReplicaSet, handler.NewReplicaSetHandler(store, settings, clock, &appsv1.ReplicaSet{}))
	wc.register(types.GroupApps, types.V1Beta2, types.KindReplicaSet, handler.NewReplicaSetHandler(store, settings, clock, &appsv1beta2.ReplicaSet{}))
	// Core API
	wc.register(types.GroupCore, types.V1, types.KindPod, handler.NewPodHandler(store, settings, clock, &corev1.Pod{}))
	wc.register(types.GroupCore, types.V1, types.KindNamespace, handler.NewNamespaceHandler(store, settings, clock, &corev1.Namespace{}))
	wc.register(types.GroupCore, types.V1, types.KindNode, handler.NewNodeHandler(store, settings, clock, &corev1.Node{}))
	wc.register(types.GroupCore, types.V1, types.KindService, handler.NewServiceHandler(store, settings, clock, &corev1.Service{}))
	wc.register(types.GroupCore, types.V1, types.KindPersistentVolume, handler.NewPersistentVolumeClaimHandler(store, settings, clock, &corev1.PersistentVolume{}))
	wc.register(types.GroupCore, types.V1, types.KindPersistentVolumeClaim, handler.NewPersistentVolumeClaimHandler(store, settings, clock, &corev1.PersistentVolumeClaim{}))
	// Batch API
	wc.register(types.GroupBatch, types.V1, types.KindJob, handler.NewJobHandler(store, settings, clock, &batchv1.Job{}))
	wc.register(types.GroupBatch, types.V1, types.KindCronJob, handler.NewCronJobHandler(store, settings, clock, &batchv1.CronJob{}))
	wc.register(types.GroupBatch, types.V1Beta1, types.KindCronJob, handler.NewCronJobHandler(store, settings, clock, &batchv1beta1.CronJob{}))
	// CRD API Objects
	wc.register(types.GroupExt, types.V1, types.KindCRD, handler.NewCustomResourceDefinitionHandler(store, settings, clock, &metav1.PartialObjectMetadata{}))
	wc.register(types.GroupExt, types.V1Beta1, types.KindCRD, handler.NewCustomResourceDefinitionHandler(store, settings, clock, &metav1beta1.PartialObjectMetadata{}))
	// Network API Objects
	wc.register(types.GroupNet, types.V1, types.KindIngress, handler.NewIngressHandler(store, settings, clock, &networkingv1.Ingress{}))
	wc.register(types.GroupNet, types.V1Beta1, types.KindIngress, handler.NewIngressHandler(store, settings, clock, &networkingv1beta1.Ingress{}))
	// Gateway API Objects
	wc.register(types.GroupGateway, types.V1, types.KindGateway, handler.NewGatewayHandler(store, settings, clock, &gatewayv1.Gateway{}))
	wc.register(types.GroupGateway, types.V1Beta1, types.KindGateway, handler.NewGatewayHandler(store, settings, clock, &gatewayv1beta1.Gateway{}))
	// StorageClass API Objects
	wc.register(types.GroupStorage, types.V1, types.KindStorageClass, handler.NewStorageClassHandler(store, settings, clock, &storagev1.StorageClass{}))
	wc.register(types.GroupStorage, types.V1Beta1, types.KindStorageClass, handler.NewStorageClassHandler(store, settings, clock, &storagev1beta1.StorageClass{}))
	// NetworkClass API Objects
	wc.register(types.GroupNet, types.V1, types.KindIngressClass, handler.NewIngressClassHandler(store, settings, clock, &networkingv1.IngressClass{}))
	wc.register(types.GroupNet, types.V1Beta1, types.KindIngressClass, handler.NewIngressClassHandler(store, settings, clock, &networkingv1beta1.IngressClass{}))
	// GatewayClass API Objects
	wc.register(types.GroupNet, types.V1, types.KindGatewayClass, handler.NewGatewayClassHandler(store, settings, clock, &gatewayv1.GatewayClass{}))
	wc.register(types.GroupNet, types.V1Beta1, types.KindGatewayClass, handler.NewGatewayClassHandler(store, settings, clock, &gatewayv1beta1.GatewayClass{}))

	return wc, nil
}

func (wc *webhookController) GetSupported() map[string]map[string]map[string]metav1.Object {
	result := map[string]map[string]map[string]metav1.Object{}
	for g, versions := range wc.dispatch {
		if result[g] == nil {
			result[g] = make(map[string]map[string]metav1.Object)
		}
		for v, kinds := range versions {
			if result[g][v] == nil {
				result[g][v] = make(map[string]metav1.Object)
			}
			for k, h := range kinds {
				result[g][v][k] = h.ObjectType
			}
		}
	}
	return result
}

func (wc *webhookController) IsSupported(g, v, k string) bool {
	return wc.registered(g, v, k)
}

func (wc *webhookController) GetConfigurationAccessor(g, v, k string) config.ConfigAccessor {
	if !wc.registered(g, v, k) {
		return nil
	}
	return wc.dispatch[g][v][k].Accessor
}

func (wc *webhookController) Review(ctx context.Context, ar *types.AdmissionReview) (*types.AdmissionResponse, error) {
	g := ar.RequestGVK.Group
	v := ar.RequestGVK.Version
	k := strings.ToLower(ar.RequestGVK.Kind) // e.g. "pod"
	o := string(ar.Operation)

	//
	metricWebhookEventTotal.WithLabelValues(g, v, k, o).Inc()

	// no specific handler -> allow by default
	if !wc.registered(g, v, k) {
		return wc.defaultHandler.Execute(ctx, ar)
	}

	processor := wc.dispatch[g][v][k]
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
func (wc *webhookController) registered(g, v, k string) bool {
	if versMap, ok := wc.dispatch[g]; ok {
		if resMap, ok := versMap[v]; ok {
			_, ok := resMap[k]
			return ok
		}
	}
	return false
}
