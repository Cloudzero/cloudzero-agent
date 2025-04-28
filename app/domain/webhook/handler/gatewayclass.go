// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

//nolint:dupl // Duplication is acceptable we expect to extend the definitions later
package handler

import (
	"context"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/helper"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/hook"
	"github.com/cloudzero/cloudzero-agent/app/types"
)

var (
	gatewayOnce       sync.Once
	GatewayClassTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: types.ObservabilityMetric("gateway_classes_total"),
			Help: "Tracks the total number of gateway class events, categorized by name and controller.",
		},
		[]string{"name", "controller"},
	)
)

type GatewayClassConfigAccessor struct {
	settings *config.Settings
}

func NewGatewayClassConfigAccessor(settings *config.Settings) config.ConfigAccessor {
	return &GatewayClassConfigAccessor{settings: settings}
}

func (g *GatewayClassConfigAccessor) LabelsEnabled() bool {
	return false
}

func (g *GatewayClassConfigAccessor) AnnotationsEnabled() bool {
	return false
}

func (g *GatewayClassConfigAccessor) LabelsEnabledForType() bool {
	return true
}

func (g *GatewayClassConfigAccessor) AnnotationsEnabledForType() bool {
	return true
}

func (g *GatewayClassConfigAccessor) ResourceType() config.ResourceType {
	return config.GatewayClass
}

func (g *GatewayClassConfigAccessor) Settings() *config.Settings {
	return g.settings
}

type GatewayClassHandler struct {
	hook.Handler
	settings   *config.Settings
	clock      types.TimeProvider
	formatData DataFormatter
}

func NewGatewayClassHandler(
	store types.ResourceStore,
	settings *config.Settings,
	clock types.TimeProvider,
	objectType metav1.Object,
) *hook.Handler {
	gatewayOnce.Do(func() {
		prometheus.MustRegister(GatewayClassTotal)
	})

	h := &GatewayClassHandler{
		settings:   settings,
		clock:      clock,
		formatData: WorkloadDataFormatter,
	}
	h.Accessor = NewGatewayClassConfigAccessor(settings)
	h.ObjectType = objectType
	h.ObjectCreator = helper.NewStaticObjectCreator(objectType)
	h.Handler.Store = store
	h.Handler.Create = h.Create()
	return &h.Handler
}

func (h *GatewayClassHandler) Create() hook.AdmitFunc {
	return func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
		// Use a type assertion to handle both v1 and v1beta1 GatewayClass
		switch o := obj.(type) {
		case *gatewayv1.GatewayClass:
			GatewayClassTotal.WithLabelValues(o.GetName(), string(o.Spec.ControllerName)).Inc()
			debugPrintObject(o, "create "+config.ResourceTypeToMetricName[h.Accessor.ResourceType()])
		case *gatewayv1beta1.GatewayClass:
			GatewayClassTotal.WithLabelValues(o.GetName(), string(o.Spec.ControllerName)).Inc()
			debugPrintObject(o, "create "+config.ResourceTypeToMetricName[h.Accessor.ResourceType()])
		default:
			log.Warn().Msgf("unable to cast to object instance of type %T", obj)
			return &types.AdmissionResponse{Allowed: true}, nil
		}
		return &types.AdmissionResponse{Allowed: true}, nil
	}
}
