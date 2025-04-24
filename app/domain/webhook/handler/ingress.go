// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

//nolint:dupl // There is currently substantial duplication in the handlers :(

package handler

// NOTES:
// There are a few types we might want to support.
// &networkingv1.Ingress{} : https://github.com/kubernetes/api/blob/master/networking/v1/types.go#L241
// extensionsv1beta1.Ingress{} : https://github.com/kubernetes/api/blob/master/networking/v1beta1/types.go#L35

import (
	"context"

	// extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	config "github.com/cloudzero/cloudzero-agent/app/config/insights-controller"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/helper"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/hook"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/rs/zerolog/log"
)

type IngressHandler struct {
	hook.Handler
	settings *config.Settings
	clock    types.TimeProvider
}

func NewIngressHandler(store types.ResourceStore, settings *config.Settings, clock types.TimeProvider) *hook.Handler {
	h := &IngressHandler{settings: settings}
	h.ObjectCreator = helper.NewStaticObjectCreator(&networkingv1.Ingress{})
	h.Handler.Create = h.Create()
	h.Handler.Update = h.Update()
	h.Handler.Delete = h.Delete()
	h.Handler.Store = store
	h.clock = clock
	return &h.Handler
}

func (h *IngressHandler) Create() hook.AdmitFunc {
	return func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
		o, ok := obj.(*networkingv1.Ingress)
		if !ok {
			log.Warn().Msg("unable to cast to pod object instance")
			return &types.AdmissionResponse{Allowed: true}, nil
		}
		debugPrintObject(o, "ingress created")
		// not saving labels/annotations
		return &types.AdmissionResponse{Allowed: true}, nil
	}
}

func (h *IngressHandler) Update() hook.AdmitFunc {
	return func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
		o, ok := obj.(*networkingv1.Ingress)
		if !ok {
			log.Warn().Msg("unable to cast to pod object instance")
			return &types.AdmissionResponse{Allowed: true}, nil
		}
		debugPrintObject(o, "ingress updated")
		// not saving labels/annotations
		return &types.AdmissionResponse{Allowed: true}, nil
	}
}

func (h *IngressHandler) Delete() hook.AdmitFunc {
	return func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
		o, ok := obj.(*networkingv1.Ingress)
		if !ok {
			log.Warn().Msg("unable to cast to pod object instance")
			return &types.AdmissionResponse{Allowed: true}, nil
		}
		debugPrintObject(o, "ingress deleted")
		return &types.AdmissionResponse{Allowed: true}, nil
	}
}

func FormatIngressData(o *networkingv1.Ingress, settings *config.Settings) types.ResourceTags {
	var (
		labels      = config.MetricLabelTags{}
		annotations = config.MetricLabelTags{}
		namespace   = o.GetNamespace()
		workload    = o.GetName()
	)
	labels = config.Filter(o.GetLabels(), settings.LabelMatches, settings.Filters.Labels.Enabled, settings)
	annotations = config.Filter(o.GetAnnotations(), settings.AnnotationMatches, settings.Filters.Annotations.Enabled, settings)
	metricLabels := config.MetricLabels{
		"workload":      workload, // standard metric labels to attach to metric
		"namespace":     namespace,
		"resource_type": config.ResourceTypeToMetricName[config.Pod],
	}
	return types.ResourceTags{
		Type:         config.Ingress,
		Name:         workload,
		Namespace:    &namespace,
		MetricLabels: &metricLabels,
		Labels:       &labels,
		Annotations:  &annotations,
	}
}
