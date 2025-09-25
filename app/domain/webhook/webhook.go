// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package webhook provides Kubernetes admission webhook business logic for CloudZero Agent cost allocation.
// This package implements the core admission control system that validates and enhances Kubernetes resources
// with cost allocation metadata during their lifecycle (CREATE, UPDATE, DELETE operations).
//
// The webhook system operates as a Primary Adapter in the hexagonal architecture, receiving
// admission requests from the Kubernetes API server and applying CloudZero's cost allocation
// policies before resources are persisted to etcd.
//
// Key responsibilities:
//   - Admission validation: Ensure resources have required cost allocation tags
//   - Metadata injection: Add missing cost center, team, and project labels
//   - Policy enforcement: Validate cost allocation rules and organizational policies
//   - Resource tracking: Store resource metadata for billing and cost optimization
//   - Multi-version support: Handle different Kubernetes API versions (v1, v1beta1, v1beta2)
//
// Architecture:
//   - WebhookController: Main orchestration and routing logic
//   - Resource Handlers: Specialized logic for each Kubernetes resource type
//   - Configuration Management: Dynamic policy updates and feature toggles
//   - Metrics Integration: Prometheus monitoring for webhook performance and errors
//
// The webhook supports 20+ Kubernetes resource types across multiple API groups:
// Apps (Deployment, StatefulSet, DaemonSet), Core (Pod, Service, PVC), Batch (Job, CronJob),
// Networking (Ingress, IngressClass), Gateway API, Storage, and Custom Resource Definitions.
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

var (
	// webhookStatsOnce ensures Prometheus metrics are registered only once during application startup.
	// This sync.Once prevents duplicate metric registration errors when the webhook controller
	// is initialized multiple times during testing or service restarts.
	//
	// Used in NewWebhookFactory to safely register webhook metrics with the global Prometheus registry
	// without risking "duplicate metric descriptor" panics during repeated initialization.
	webhookStatsOnce sync.Once

	// metricWebhookEventTotal tracks admission webhook processing volume across all supported resource types.
	// This Prometheus counter vector enables monitoring of webhook load, resource type distribution,
	// and operation patterns essential for CloudZero Agent operational monitoring and capacity planning.
	//
	// Labels provide comprehensive filtering and aggregation capabilities:
	//   - "kind_group": Kubernetes API group (apps, core, batch, networking, gateway-api)
	//   - "kind_version": API version (v1, v1beta1, v1beta2) for compatibility tracking
	//   - "kind_resource": Resource kind (deployment, pod, service) for workload analysis
	//   - "operation": Admission operation (create, update, delete) for lifecycle monitoring
	//
	// Operational use cases:
	//   - Capacity planning: Monitor webhook request volume and resource hotspots
	//   - Performance analysis: Identify high-traffic resource types requiring optimization
	//   - Cost allocation insights: Track resource creation patterns by type and namespace
	//   - Compliance monitoring: Validate webhook coverage across supported resource types
	//
	// Example queries:
	//   - Total webhook events: sum(rate(czo_webhook_types_total[5m]))
	//   - Pod creation rate: sum(rate(czo_webhook_types_total{kind_resource="pod", operation="create"}[5m]))
	//   - API version usage: sum by (kind_version) (czo_webhook_types_total)
	//
	// The metric uses ObservabilityMetric() naming convention, prefixing with "czo_" to distinguish
	// operational metrics from cost allocation metrics that use "cloudzero_" prefix.
	metricWebhookEventTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: types.ObservabilityMetric("webhook_types_total"),
			Help: "Total number of webhook events filterable by kind_group, kind_version, kind_resource, and operation",
		},
		[]string{"kind_group", "kind_version", "kind_resource", "operation"},
	)
)

// WebhookController defines the core interface for CloudZero Agent admission webhook processing.
// This interface orchestrates the entire admission control pipeline, routing incoming Kubernetes
// admission requests to appropriate resource-specific handlers based on Group/Version/Kind (GVK) mapping.
//
// The controller implements a dispatch pattern where each supported Kubernetes resource type
// (Pod, Deployment, Service, etc.) has a dedicated handler with specialized cost allocation logic.
// When no specific handler exists, a default handler allows requests to proceed without modification.
//
// Key architectural responsibilities:
//   - Request routing: Map GVK combinations to specialized resource handlers
//   - Handler registry: Maintain supported resource types and API version compatibility
//   - Configuration access: Provide resource-specific configuration for cost allocation policies
//   - Default behavior: Ensure unknown resources are handled gracefully without blocking
//
// The interface enables testing through mock implementations and supports runtime reconfiguration
// of cost allocation policies without requiring webhook service restarts.
//
// Integration points:
//   - HTTP handlers receive admission requests and delegate to this controller
//   - Resource stores persist cost allocation metadata extracted during processing
//   - Configuration services provide dynamic policy updates and feature toggles
//   - Monitoring systems track processing metrics and error rates
type WebhookController interface {
	// GetSupported returns the complete registry of supported Kubernetes resource types for cost allocation.
	// This method provides a comprehensive map of Group/Version/Kind combinations that the CloudZero Agent
	// webhook can process, enabling dynamic discovery of supported resources and API version compatibility.
	//
	// The nested map structure enables efficient lookup and iteration:
	//   - First key (string): API group (apps, core, batch, networking, gateway-api, storage, extensions)
	//   - Second key (string): API version (v1, v1beta1, v1beta2) for backward compatibility
	//   - Third key (string): Resource kind (deployment, pod, service, ingress, job, etc.)
	//   - Value (metav1.Object): Prototype object for the resource type, used for deserialization
	//
	// Used by:
	//   - HTTP handlers to validate incoming admission requests against supported resources
	//   - Configuration systems to generate webhook configuration YAML with proper GVK declarations
	//   - Monitoring dashboards to display webhook coverage and supported resource inventory
	//   - Testing frameworks to verify handler registration for all supported resource combinations
	//
	// The returned map reflects the current handler registry and updates dynamically as handlers
	// are registered during webhook controller initialization or runtime reconfiguration.
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

	// Review executes the core admission control logic for CloudZero cost allocation validation.
	// This method is the primary entry point for processing Kubernetes admission requests,
	// orchestrating the complete cost allocation pipeline from resource validation to metadata storage.
	//
	// Processing pipeline:
	//   1. Extract Group/Version/Kind from admission request for handler selection
	//   2. Record webhook event metrics for operational monitoring and analysis
	//   3. Route to resource-specific handler or default allow-all handler
	//   4. Execute cost allocation validation, policy enforcement, and metadata injection
	//   5. Store resource metadata for billing attribution and cost optimization
	//   6. Return admission decision with detailed explanations and warnings
	//
	// Resource-specific handlers implement specialized logic for different Kubernetes resource types:
	//   - Pods: Validate required cost tags, inject missing labels, track resource requests
	//   - Deployments: Enforce team assignments, validate cost center policies
	//   - Services: Apply network cost allocation rules, validate ingress configurations
	//   - PVCs: Implement storage cost attribution, validate retention policies
	//   - Jobs: Apply batch workload cost tracking, validate resource quotas
	//
	// Default handler behavior (unknown resources):
	//   - Allow admission without modification to prevent blocking legitimate workloads
	//   - Log unknown resource types for potential handler development
	//   - Record metrics for monitoring webhook coverage gaps
	//
	// Error handling and resilience:
	//   - Admission failures are logged with detailed context for troubleshooting
	//   - Network timeouts and context cancellation are handled gracefully
	//   - Configuration errors result in admission denial with clear user guidance
	//   - Handler panics are recovered to prevent webhook service disruption
	//
	// Performance considerations:
	//   - Request processing latency is tracked via Prometheus metrics
	//   - Handler selection uses efficient map lookup (O(1) complexity)
	//   - Resource deserialization is optimized for common resource types
	//   - Database operations are batched where possible for efficiency
	//
	// Returns admission response with:
	//   - Allowed: Boolean decision whether to permit the resource operation
	//   - Message: Detailed explanation for denials or informational guidance
	//   - Warnings: Non-blocking advice about cost allocation best practices
	//   - ID: Request correlation ID for audit trails and troubleshooting
	Review(ctx context.Context, ar *types.AdmissionReview) (*types.AdmissionResponse, error)

	// Settings returns the configuration settings for the webhook controller.
	//
	// Returns:
	//   - *config.Settings: The settings instance containing webhook configuration.
	Settings() *config.Settings
}

// webhookController implements the WebhookController interface for CloudZero admission control processing.
// This struct maintains the complete handler registry, routing logic, and operational state required
// for processing admission requests across all supported Kubernetes resource types.
//
// The controller uses a three-level dispatch map for efficient O(1) handler lookup by
// Group/Version/Kind, enabling fast routing of admission requests to appropriate handlers
// without linear searches or complex matching logic.
type webhookController struct {
	// defaultHandler processes admission requests for unsupported or unknown resource types.
	// This handler implements a "fail-open" policy, allowing admission by default to prevent
	// the CloudZero Agent from blocking legitimate Kubernetes operations when encountering
	// new resource types or API versions not yet supported by the agent.
	//
	// The default handler also serves as a fallback during handler registration failures
	// or configuration errors, ensuring webhook stability and operational continuity.
	defaultHandler *hook.Handler

	// dispatch provides the three-level routing map for efficient handler selection by GVK.
	// Structure: dispatch[group][version][kind] -> *hook.Handler
	//
	// Examples:
	//   - dispatch["apps"]["v1"]["deployment"] -> Deployment handler
	//   - dispatch["core"]["v1"]["pod"] -> Pod handler
	//   - dispatch["batch"]["v1"]["job"] -> Job handler
	//
	// This structure enables O(1) handler lookup performance and supports the full
	// complexity of Kubernetes API versioning across different resource types.
	dispatch map[string]map[string]map[string]*hook.Handler

	// enabled controls whether the webhook controller actively processes admission requests.
	// When disabled, all requests are passed to the default handler which allows admission
	// without cost allocation processing. This enables operational maintenance and
	// emergency bypass scenarios without requiring webhook service shutdown.
	enabled bool

	// settings provides access to dynamic configuration for cost allocation policies.
	// This includes feature toggles, validation rules, cost center mappings, and
	// resource-specific configuration that can be updated without service restarts.
	// Handlers access this configuration through the controller interface.
	settings *config.Settings

	// clock abstracts time operations for testing and consistent timestamp generation.
	// Used throughout the webhook processing pipeline for audit logging, resource
	// metadata timestamps, and operational metrics. Enables deterministic testing
	// of time-sensitive operations and consistent behavior across time zones.
	clock types.TimeProvider
}

// NewWebhookFactory constructs a fully configured WebhookController for CloudZero admission control.
// This factory function initializes the complete webhook processing pipeline, including handler registration
// for all supported Kubernetes resource types, metrics setup, and operational configuration.
//
// The factory performs comprehensive initialization:
//  1. Creates default "allow-all" handler for unknown resource types
//  2. Registers 20+ resource-specific handlers across 5 API groups
//  3. Establishes Prometheus metrics for operational monitoring
//  4. Configures multi-version API support (v1, v1beta1, v1beta2)
//  5. Sets up dependency injection for storage, configuration, and time services
//
// Supported resource types and API groups:
//   - Apps API: Deployment, StatefulSet, DaemonSet, ReplicaSet
//   - Core API: Pod, Service, PVC, PV, Namespace, Node
//   - Batch API: Job, CronJob
//   - Networking API: Ingress, IngressClass
//   - Gateway API: Gateway, GatewayClass
//   - Storage API: StorageClass
//   - Extensions API: CustomResourceDefinition
//
// Multi-version compatibility:
//
//	Each resource type is registered for multiple API versions to ensure compatibility
//	across different Kubernetes cluster versions. This prevents admission failures when
//	clusters use deprecated API versions during upgrades or legacy installations.
//
// Dependencies and integration:
//   - store: ResourceStore for persisting cost allocation metadata
//   - settings: Dynamic configuration for cost allocation policies and feature toggles
//   - clock: TimeProvider for consistent timestamps and testing determinism
//
// Error conditions:
//
//	Currently always returns success, but the error return enables future validation
//	of configuration consistency, handler registration conflicts, or dependency issues.
//
// Performance optimizations:
//   - Handler registration uses efficient map initialization
//   - Prometheus metrics are registered once using sync.Once
//   - Dispatch map structure enables O(1) handler lookup during request processing
func NewWebhookFactory(store types.ResourceStore, settings *config.Settings, clock types.TimeProvider) (WebhookController, error) {
	wc := &webhookController{
		dispatch: make(map[string]map[string]map[string]*hook.Handler),
		defaultHandler: &hook.Handler{
			ObjectCreator: helper.NewDynamicObjectCreator(),
			Create:        hook.AllowAlways,
			Update:        hook.AllowAlways,
			Delete:        hook.AllowAlways,
			Connect:       hook.AllowAlways,
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

func (wc *webhookController) Settings() *config.Settings {
	return wc.settings
}
