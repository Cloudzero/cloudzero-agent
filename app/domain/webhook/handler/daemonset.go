// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

//nolint:dupl // There is currently substantial duplication in the handlers :(
package handler

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/helper"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/hook"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/rs/zerolog/log"
)

type DaemonSetHandler struct {
	hook.Handler
	settings *config.Settings
	clock    types.TimeProvider
}

func NewDaemonSetHandler(store types.ResourceStore, settings *config.Settings, clock types.TimeProvider) *hook.Handler {
	// Need little trick to protect internal data
	h := &DaemonSetHandler{settings: settings}
	h.ObjectCreator = helper.NewStaticObjectCreator(&appsv1.DaemonSet{})
	h.Handler.Create = h.Create()
	h.Handler.Update = h.Update()
	h.Handler.Delete = h.Delete()
	h.Handler.Store = store
	h.clock = clock
	return &h.Handler
}

func (h *DaemonSetHandler) Create() hook.AdmitFunc {
	return func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
		o, ok := obj.(*appsv1.DaemonSet)
		if !ok {
			log.Warn().Msg("unable to cast to daemonset object instance")
			return &types.AdmissionResponse{Allowed: true}, nil
		}
		debugPrintObject(o, "daemonset created")
		// only process if enabled, always return allowed to not block an admission
		if h.settings.Filters.Labels.Resources.DaemonSets || h.settings.Filters.Annotations.Resources.DaemonSets {
			genericWriteDataToStorage(ctx, h.Store, h.clock, FormatDaemonSetData(o, h.settings))
		}
		return &types.AdmissionResponse{Allowed: true}, nil
	}
}

func (h *DaemonSetHandler) Update() hook.AdmitFunc {
	return func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
		o, ok := obj.(*appsv1.DaemonSet)
		if !ok {
			log.Warn().Msg("unable to cast to daemonset object instance")
			return &types.AdmissionResponse{Allowed: true}, nil
		}
		debugPrintObject(o, "daemonset updated")
		// only process if enabled, always return allowed to not block an admission
		if h.settings.Filters.Labels.Resources.DaemonSets || h.settings.Filters.Annotations.Resources.DaemonSets {
			genericWriteDataToStorage(ctx, h.Store, h.clock, FormatDaemonSetData(o, h.settings))
		}
		return &types.AdmissionResponse{Allowed: true}, nil
	}
}

func (h *DaemonSetHandler) Delete() hook.AdmitFunc {
	return func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
		o, ok := obj.(*appsv1.DaemonSet)
		if !ok {
			log.Warn().Msg("unable to cast to daemonset object instance")
			return &types.AdmissionResponse{Allowed: true}, nil
		}
		debugPrintObject(o, "daemonset deleted")
		return &types.AdmissionResponse{Allowed: true}, nil
	}
}

func FormatDaemonSetData(o *appsv1.DaemonSet, settings *config.Settings) types.ResourceTags {
	var (
		labels      = config.MetricLabelTags{}
		annotations = config.MetricLabelTags{}
		namespace   = o.GetNamespace()
		workload    = o.GetName()
	)
	if settings.Filters.Labels.Enabled {
		labels = config.Filter(o.GetLabels(), settings.LabelMatches, (settings.Filters.Labels.Enabled && settings.Filters.Labels.Resources.DaemonSets), settings)
	}
	if settings.Filters.Annotations.Enabled {
		annotations = config.Filter(o.GetAnnotations(), settings.AnnotationMatches, (settings.Filters.Annotations.Enabled && settings.Filters.Annotations.Resources.DaemonSets), settings)
	}
	metricLabels := config.MetricLabels{
		"workload":      workload, // standard metric labels to attach to metric
		"namespace":     namespace,
		"resource_type": config.ResourceTypeToMetricName[config.DaemonSet],
	}
	return types.ResourceTags{
		Type:         config.DaemonSet,
		Name:         workload,
		Namespace:    &namespace,
		MetricLabels: &metricLabels,
		Labels:       &labels,
		Annotations:  &annotations,
	}
}
