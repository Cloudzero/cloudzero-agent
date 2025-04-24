// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

//nolint:dupl // There is currently substantial duplication in the handlers :(
package handler

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	config "github.com/cloudzero/cloudzero-agent/app/config/insights-controller"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/helper"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/hook"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/rs/zerolog/log"
)

// ---------------------------------[EX 1]---------------------------------------
// apiVersion: gateway.networking.k8s.io/v1beta1
// kind: Gateway
// metadata:
//
//	name: my-hotel
//
// spec:
//
//	gatewayClassName: amazon-vpc-lattice
//	listeners:
//	  - name: http
//	    protocol: HTTP
//	    port: 80
// ---------------------------------[EX 2]---------------------------------------
// # 1. Define a GatewayClass that your controller will watch
// ----------
// apiVersion: gateway.networking.k8s.io/v1
// kind: GatewayClass
// metadata:
//   name: example-gatewayclass
// spec:
//   controllerName: example.com/gateway-controller
// ---
// # 2. Create a Gateway bound to that class
// ----------
// apiVersion: gateway.networking.k8s.io/v1
// kind: Gateway
// metadata:
//   name: example-gateway
//   namespace: default
// spec:
//   gatewayClassName: example-gatewayclass
//   listeners:
//     - name: http
//       protocol: HTTP
//       port: 80
//       # allow Routes from any namespace
//       allowedRoutes:
//         namespaces:
//           from: All
// ---
// # 3. Attach an HTTPRoute to send traffic to a Service
// ----------
// apiVersion: gateway.networking.k8s.io/v1
// kind: HTTPRoute
// metadata:
//   name: example-httproute
//   namespace: default
// spec:
//   parentRefs:
//     - name: example-gateway
//   hostnames:
//     - "example.com"
//   rules:
//     - matches:
//         - path:
//             type: PathPrefix
//             value: "/"
//       forwardTo:
//         - serviceName: my-service
//           port: 80

type GatewayHandler struct {
	hook.Handler
	settings *config.Settings
	clock    types.TimeProvider
}

func NewGatewayHandler(store types.ResourceStore, settings *config.Settings, clock types.TimeProvider) *hook.Handler {
	h := &GatewayHandler{settings: settings}
	h.ObjectCreator = helper.NewStaticObjectCreator(&gatewayv1.Gateway{})
	h.Handler.Create = h.Create()
	h.Handler.Update = h.Update()
	h.Handler.Delete = h.Delete()
	h.Handler.Store = store
	h.clock = clock
	return &h.Handler
}

func (h *GatewayHandler) Create() hook.AdmitFunc {
	return func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
		o, ok := obj.(*gatewayv1.Gateway)
		if !ok {
			log.Warn().Msg("unable to cast to gateway object instance")
			return &types.AdmissionResponse{Allowed: true}, nil
		}
		debugPrintObject(o, "gateway created")
		return &types.AdmissionResponse{Allowed: true}, nil
	}
}

func (h *GatewayHandler) Update() hook.AdmitFunc {
	return func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
		o, ok := obj.(*gatewayv1.Gateway)
		if !ok {
			log.Warn().Msg("unable to cast to gateway object instance")
			return &types.AdmissionResponse{Allowed: true}, nil
		}
		debugPrintObject(o, "gateway updated")
		return &types.AdmissionResponse{Allowed: true}, nil
	}
}

func (h *GatewayHandler) Delete() hook.AdmitFunc {
	return func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
		o, ok := obj.(*gatewayv1.Gateway)
		if !ok {
			log.Warn().Msg("unable to cast to gateway object instance")
			return &types.AdmissionResponse{Allowed: true}, nil
		}
		debugPrintObject(o, "gateway deleted")
		return &types.AdmissionResponse{Allowed: true}, nil
	}
}

func FormatGatewayData(o *gatewayv1.Gateway, settings *config.Settings) types.ResourceTags {
	var (
		labels      = config.MetricLabelTags{}
		annotations = config.MetricLabelTags{}
		namespace   = o.GetNamespace()
		workload    = o.GetName()
	)
	labels = config.Filter(o.GetLabels(), settings.LabelMatches, settings.Filters.Labels.Enabled, settings)
	annotations = config.Filter(o.GetAnnotations(), settings.AnnotationMatches, settings.Filters.Annotations.Enabled, settings)
	metricLabels := config.MetricLabels{
		"workload":      workload,
		"namespace":     namespace,
		"resource_type": config.ResourceTypeToMetricName[config.Gateway],
	}
	return types.ResourceTags{
		Type:         config.Gateway,
		Name:         workload,
		Namespace:    &namespace,
		MetricLabels: &metricLabels,
		Labels:       &labels,
		Annotations:  &annotations,
	}
}
