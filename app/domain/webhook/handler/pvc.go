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

type PersistentVolumeClaimHandler struct {
	hook.Handler
	settings *config.Settings
	clock    types.TimeProvider
}

func NewPersistentVolumeClaimHandler(store types.ResourceStore, settings *config.Settings, clock types.TimeProvider) *hook.Handler {
	h := &PersistentVolumeClaimHandler{settings: settings}
	h.ObjectCreator = helper.NewStaticObjectCreator(&corev1.PersistentVolumeClaim{})
	h.Handler.Create = h.Create()
	h.Handler.Update = h.Update()
	h.Handler.Delete = h.Delete()
	h.Handler.Store = store
	h.clock = clock
	return &h.Handler
}

func (h *PersistentVolumeClaimHandler) Create() hook.AdmitFunc {
	return func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
		o, ok := obj.(*corev1.PersistentVolumeClaim)
		if !ok {
			log.Warn().Msg("unable to cast to PersistentVolumeClaim object instance")
			return &types.AdmissionResponse{Allowed: true}, nil
		}
		debugPrintObject(o, "persistent volume claim created")
		// not storing labels and annotations
		return &types.AdmissionResponse{Allowed: true}, nil
	}
}

func (h *PersistentVolumeClaimHandler) Update() hook.AdmitFunc {
	return func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
		o, ok := obj.(*corev1.PersistentVolumeClaim)
		if !ok {
			log.Warn().Msg("unable to cast to PersistentVolumeClaim object instance")
			return &types.AdmissionResponse{Allowed: true}, nil
		}
		debugPrintObject(o, "persistent volume claim updated")
		// not storing labels and annotations
		return &types.AdmissionResponse{Allowed: true}, nil
	}
}

func (h *PersistentVolumeClaimHandler) Delete() hook.AdmitFunc {
	return func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
		o, ok := obj.(*corev1.PersistentVolumeClaim)
		if !ok {
			log.Warn().Msg("unable to cast to PersistentVolumeClaim object instance")
			return &types.AdmissionResponse{Allowed: true}, nil
		}
		debugPrintObject(o, "persistent volume claim deleted")
		return &types.AdmissionResponse{Allowed: true}, nil
	}
}

func FormatPersistentVolumeClaimData(o *corev1.PersistentVolumeClaim, settings *config.Settings) types.ResourceTags {
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
		"resource_type": config.ResourceTypeToMetricName[config.PersistentVolumeClaim],
	}
	return types.ResourceTags{
		Type:         config.PersistentVolumeClaim,
		Name:         workload,
		Namespace:    &namespace,
		MetricLabels: &metricLabels,
		Labels:       &labels,
		Annotations:  &annotations,
	}
}
