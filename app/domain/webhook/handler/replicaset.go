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

type ReplicaSetHandler struct {
	hook.Handler
	settings *config.Settings
	clock    types.TimeProvider
}

func NewReplicaSetHandler(store types.ResourceStore, settings *config.Settings, clock types.TimeProvider) *hook.Handler {
	h := &ReplicaSetHandler{settings: settings}
	h.ObjectCreator = helper.NewStaticObjectCreator(&appsv1.ReplicaSet{})
	h.Handler.Create = h.Create()
	h.Handler.Update = h.Update()
	h.Handler.Delete = h.Delete()
	h.Handler.Store = store
	h.clock = clock
	return &h.Handler
}

func (h *ReplicaSetHandler) Create() hook.AdmitFunc {
	return func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
		o, ok := obj.(*appsv1.ReplicaSet)
		if !ok {
			log.Warn().Msg("unable to cast to replicaset object instance")
			return &types.AdmissionResponse{Allowed: true}, nil
		}
		debugPrintObject(o, "replica add")
		// Don't save labels/annotations
		return &types.AdmissionResponse{Allowed: true}, nil
	}
}

func (h *ReplicaSetHandler) Update() hook.AdmitFunc {
	return func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
		o, ok := obj.(*appsv1.ReplicaSet)
		if !ok {
			log.Warn().Msg("unable to cast to replicaset object instance")
			return &types.AdmissionResponse{Allowed: true}, nil
		}
		debugPrintObject(o, "replica update")
		// Don't save labels/annotations
		return &types.AdmissionResponse{Allowed: true}, nil
	}
}

func (h *ReplicaSetHandler) Delete() hook.AdmitFunc {
	return func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
		o, ok := obj.(*appsv1.ReplicaSet)
		if !ok {
			log.Warn().Msg("unable to cast to node object instance")
			return &types.AdmissionResponse{Allowed: true}, nil
		}
		debugPrintObject(o, "replica deleted")
		return &types.AdmissionResponse{Allowed: true}, nil
	}
}

func FormatReplicaSetData(o *appsv1.ReplicaSet, settings *config.Settings) types.ResourceTags {
	var (
		labels      = config.MetricLabelTags{}
		annotations = config.MetricLabelTags{}
		namespace   = o.GetNamespace()
		rsName      = o.GetName()
	)
	labels = config.Filter(o.GetLabels(), settings.LabelMatches, settings.Filters.Labels.Enabled, settings)
	annotations = config.Filter(o.GetAnnotations(), settings.AnnotationMatches, settings.Filters.Annotations.Enabled, settings)
	metricLabels := config.MetricLabels{
		"replicaset":    rsName, // standard metric labels to attach to metric
		"namespace":     namespace,
		"resource_type": config.ResourceTypeToMetricName[config.ReplicaSet],
	}
	return types.ResourceTags{
		Type:         config.ReplicaSet,
		Name:         rsName,
		Namespace:    &namespace,
		MetricLabels: &metricLabels,
		Labels:       &labels,
		Annotations:  &annotations,
	}
}
