// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

//nolint:dupl // There is currently substantial duplication in the handlers :(
package handler

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/helper"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/hook"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/rs/zerolog/log"
)

type PodHandler struct {
	hook.Handler
	settings *config.Settings
	clock    types.TimeProvider
}

func NewPodHandler(store types.ResourceStore, settings *config.Settings, clock types.TimeProvider) *hook.Handler {
	h := &PodHandler{settings: settings}
	h.ObjectCreator = helper.NewStaticObjectCreator(&corev1.Pod{})
	h.Handler.Create = h.Create()
	h.Handler.Update = h.Update()
	h.Handler.Delete = h.Delete()
	h.Handler.Store = store
	h.clock = clock
	return &h.Handler
}

func (h *PodHandler) Create() hook.AdmitFunc {
	return func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
		o, ok := obj.(*corev1.Pod)
		if !ok {
			log.Warn().Msg("unable to cast to pod object instance")
			return &types.AdmissionResponse{Allowed: true}, nil
		}
		debugPrintObject(o, "pod add")
		if h.settings.Filters.Labels.Resources.Pods || h.settings.Filters.Annotations.Resources.Pods {
			genericWriteDataToStorage(ctx, h.Store, h.clock, FormatPodData(o, h.settings))
		}
		return &types.AdmissionResponse{Allowed: true}, nil
	}
}

func (h *PodHandler) Update() hook.AdmitFunc {
	return func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
		o, ok := obj.(*corev1.Pod)
		if !ok {
			log.Warn().Msg("unable to cast to pod object instance")
			return &types.AdmissionResponse{Allowed: true}, nil
		}
		debugPrintObject(o, "pod updated")
		if h.settings.Filters.Labels.Resources.Pods || h.settings.Filters.Annotations.Resources.Pods {
			genericWriteDataToStorage(ctx, h.Store, h.clock, FormatPodData(o, h.settings))
		}
		return &types.AdmissionResponse{Allowed: true}, nil
	}
}

func (h *PodHandler) Delete() hook.AdmitFunc {
	return func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
		o, ok := obj.(*corev1.Pod)
		if !ok {
			log.Warn().Msg("unable to cast to pod object instance")
			return &types.AdmissionResponse{Allowed: true}, nil
		}
		debugPrintObject(o, "pod deleted")
		return &types.AdmissionResponse{Allowed: true}, nil
	}
}

func FormatPodData(o *corev1.Pod, settings *config.Settings) types.ResourceTags {
	var (
		labels      = config.MetricLabelTags{}
		annotations = config.MetricLabelTags{}
		namespace   = o.GetNamespace()
		podName     = o.GetName()
	)
	if settings.Filters.Labels.Enabled {
		labels = config.Filter(o.GetLabels(), settings.LabelMatches, (settings.Filters.Labels.Enabled && settings.Filters.Labels.Resources.Pods), settings)
	}
	if settings.Filters.Annotations.Enabled {
		annotations = config.Filter(o.GetAnnotations(), settings.AnnotationMatches, (settings.Filters.Annotations.Enabled && settings.Filters.Annotations.Resources.Pods), settings)
	}
	metricLabels := config.MetricLabels{
		"pod":           podName, // standard metric labels to attach to metric
		"namespace":     namespace,
		"resource_type": config.ResourceTypeToMetricName[config.Pod],
	}
	return types.ResourceTags{
		Type:         config.Pod,
		Name:         podName,
		Namespace:    &namespace,
		MetricLabels: &metricLabels,
		Labels:       &labels,
		Annotations:  &annotations,
	}
}
