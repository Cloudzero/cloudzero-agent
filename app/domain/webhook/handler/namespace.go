// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

//nolint:dupl // There is currently substantial duplication in the handlers :(
package handler

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	config "github.com/cloudzero/cloudzero-agent/app/config/insights-controller"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/helper"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/hook"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/rs/zerolog/log"
)

type NamespaceHandler struct {
	hook.Handler
	settings *config.Settings
	clock    types.TimeProvider
}

func NewNamespaceHandler(store types.ResourceStore, settings *config.Settings, clock types.TimeProvider) *hook.Handler {
	h := &NamespaceHandler{settings: settings}
	h.ObjectCreator = helper.NewStaticObjectCreator(&corev1.Namespace{})
	h.Handler.Create = h.Create()
	h.Handler.Update = h.Update()
	h.Handler.Delete = h.Delete()
	h.Handler.Store = store
	h.clock = clock
	return &h.Handler
}

func (h *NamespaceHandler) Create() hook.AdmitFunc {
	return func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
		o, ok := obj.(*corev1.Namespace)
		if !ok {
			log.Warn().Msg("unable to cast to namespace object instance")
			return &types.AdmissionResponse{Allowed: true}, nil
		}
		debugPrintObject(o, "namespace created")
		if h.settings.Filters.Labels.Resources.Namespaces || h.settings.Filters.Annotations.Resources.Namespaces {
			genericWriteDataToStorage(ctx, h.Store, h.clock, FormatNamespaceData(o, h.settings))
		}
		return &types.AdmissionResponse{Allowed: true}, nil
	}
}

func (h *NamespaceHandler) Update() hook.AdmitFunc {
	return func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
		o, ok := obj.(*corev1.Namespace)
		if !ok {
			log.Warn().Msg("unable to cast to namespace object instance")
			return &types.AdmissionResponse{Allowed: true}, nil
		}
		debugPrintObject(o, "namespace updated")
		if h.settings.Filters.Labels.Resources.Namespaces || h.settings.Filters.Annotations.Resources.Namespaces {
			genericWriteDataToStorage(ctx, h.Store, h.clock, FormatNamespaceData(o, h.settings))
		}
		return &types.AdmissionResponse{Allowed: true}, nil
	}
}

func (h *NamespaceHandler) Delete() hook.AdmitFunc {
	return func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
		o, ok := obj.(*corev1.Namespace)
		if !ok {
			log.Warn().Msg("unable to cast to namespace object instance")
			return &types.AdmissionResponse{Allowed: true}, nil
		}
		debugPrintObject(o, "namespace deleted")
		return &types.AdmissionResponse{Allowed: true}, nil
	}
}

func FormatNamespaceData(o *corev1.Namespace, settings *config.Settings) types.ResourceTags {
	var (
		labels      = config.MetricLabelTags{}
		annotations = config.MetricLabelTags{}
		namespace   = o.GetName()
	)
	if settings.Filters.Labels.Enabled {
		labels = config.Filter(o.GetLabels(), settings.LabelMatches, (settings.Filters.Labels.Enabled && settings.Filters.Labels.Resources.Namespaces), settings)
	}
	if settings.Filters.Annotations.Enabled {
		annotations = config.Filter(o.GetAnnotations(), settings.AnnotationMatches, (settings.Filters.Annotations.Enabled && settings.Filters.Annotations.Resources.Namespaces), settings)
	}
	metricLabels := config.MetricLabels{
		"namespace":     namespace, // standard metric labels to attach to metric
		"resource_type": config.ResourceTypeToMetricName[config.Namespace],
	}
	return types.ResourceTags{
		Name:         namespace,
		Namespace:    nil,
		Type:         config.Namespace,
		MetricLabels: &metricLabels,
		Labels:       &labels,
		Annotations:  &annotations,
	}
}
