// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

//nolint:dupl // Duplication is acceptable we expect to extend the definitions later
package handler

import (
	"context"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/helper"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/hook"
	"github.com/cloudzero/cloudzero-agent/app/types"
)

var (
	ingressClassOnce  sync.Once
	IngressTypesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: types.ObservabilityMetric("ingress_types_total"),
			Help: "Tracks the total number of ingress class events, categorized by name and controller.",
		},
		[]string{"name", "controller"},
	)
)

type IngressClassConfigAccessor struct {
	settings *config.Settings
}

func NewIngressClassConfigAccessor(settings *config.Settings) config.ConfigAccessor {
	return &IngressClassConfigAccessor{settings: settings}
}

func (i *IngressClassConfigAccessor) LabelsEnabled() bool {
	return i.settings.Filters.Labels.Enabled
}

func (i *IngressClassConfigAccessor) AnnotationsEnabled() bool {
	return i.settings.Filters.Annotations.Enabled
}

func (i *IngressClassConfigAccessor) LabelsEnabledForType() bool {
	return true
}

func (i *IngressClassConfigAccessor) AnnotationsEnabledForType() bool {
	return true
}

func (i *IngressClassConfigAccessor) ResourceType() config.ResourceType {
	return config.IngressClass
}

func (i *IngressClassConfigAccessor) Settings() *config.Settings {
	return i.settings
}

type IngressClassHandler struct {
	hook.Handler
	settings   *config.Settings
	clock      types.TimeProvider
	formatData DataFormatter
}

func NewIngressClassHandler(
	store types.ResourceStore,
	settings *config.Settings,
	clock types.TimeProvider,
	objectType metav1.Object,
) *hook.Handler {
	ingressClassOnce.Do(func() {
		prometheus.MustRegister(IngressTypesTotal)
	})

	h := &IngressClassHandler{
		settings:   settings,
		clock:      clock,
		formatData: WorkloadDataFormatter,
	}
	h.Accessor = NewIngressClassConfigAccessor(settings)
	h.ObjectType = objectType
	h.ObjectCreator = helper.NewStaticObjectCreator(objectType)
	h.Handler.Store = store
	h.Handler.Create = h.createHandler()
	return &h.Handler
}

func (h *IngressClassHandler) createHandler() hook.AdmitFunc {
	return func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
		// Use a type assertion to handle both v1 and v1beta1 IngressClass
		switch o := obj.(type) {
		case *networkingv1.IngressClass:
			IngressTypesTotal.WithLabelValues(o.GetName(), o.Spec.Controller).Inc()
			debugPrintObject(o, "create "+config.ResourceTypeToMetricName[h.Accessor.ResourceType()])
		case *networkingv1beta1.IngressClass:
			IngressTypesTotal.WithLabelValues(o.GetName(), o.Spec.Controller).Inc()
			debugPrintObject(o, "create "+config.ResourceTypeToMetricName[h.Accessor.ResourceType()])
		default:
			log.Warn().Msgf("unable to cast to object instance of type %T", obj)
			return &types.AdmissionResponse{Allowed: true}, nil
		}
		return &types.AdmissionResponse{Allowed: true}, nil
	}
}
