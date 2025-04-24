// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

//nolint:dupl // There is currently substantial duplication in the handlers :(
package handler

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/helper"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/hook"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/rs/zerolog/log"
)

type CustomResourceDefinitionHandler struct {
	hook.Handler
	settings *config.Settings
	clock    types.TimeProvider
}

func NewCustomResourceDefinitionHandler(store types.ResourceStore, settings *config.Settings, clock types.TimeProvider) *hook.Handler {
	h := &CustomResourceDefinitionHandler{settings: settings}
	h.ObjectCreator = helper.NewStaticObjectCreator(&metav1.PartialObjectMetadata{})
	h.Handler.Create = h.Create()
	h.Handler.Update = h.Update()
	h.Handler.Delete = h.Delete()
	h.Handler.Store = store
	h.clock = clock
	return &h.Handler
}

func (h *CustomResourceDefinitionHandler) Create() hook.AdmitFunc {
	return func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
		o, ok := obj.(*metav1.PartialObjectMetadata)
		if !ok {
			log.Warn().Msg("unable to cast to custom resource definition object instance")
			return &types.AdmissionResponse{Allowed: true}, nil
		}
		debugPrintObject(o, "CRD created")
		// not storing labels/annotations
		return &types.AdmissionResponse{Allowed: true}, nil
	}
}

func (h *CustomResourceDefinitionHandler) Update() hook.AdmitFunc {
	return func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
		o, ok := obj.(*metav1.PartialObjectMetadata)
		if !ok {
			log.Warn().Msg("unable to cast to custom resource definition object instance")
			return &types.AdmissionResponse{Allowed: true}, nil
		}
		debugPrintObject(o, "CRD updated")
		// not storing labels/annotations
		return &types.AdmissionResponse{Allowed: true}, nil
	}
}

func (h *CustomResourceDefinitionHandler) Delete() hook.AdmitFunc {
	return func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
		o, ok := obj.(*metav1.PartialObjectMetadata)
		if !ok {
			log.Warn().Msg("unable to cast to custom resource definition object instance")
			return &types.AdmissionResponse{Allowed: true}, nil
		}
		debugPrintObject(o, "CRD deleted")
		return &types.AdmissionResponse{Allowed: true}, nil
	}
}

func FormatCustomResourceDefinitionData(o *metav1.PartialObjectMetadata, settings *config.Settings) types.ResourceTags {
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
		"resource_type": config.ResourceTypeToMetricName[config.CustomResourceDefinition],
	}
	return types.ResourceTags{
		Type:         config.CustomResourceDefinition,
		Name:         workload,
		Namespace:    &namespace,
		MetricLabels: &metricLabels,
		Labels:       &labels,
		Annotations:  &annotations,
	}
}
