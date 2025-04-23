// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

//nolint:dupl // There is currently substantial duplication in the handlers :(
package handler

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	config "github.com/cloudzero/cloudzero-agent/app/config/insights-controller"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/helper"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/hook"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/rs/zerolog/log"
)

type DeploymentHandler struct {
	hook.Handler
	settings *config.Settings
	clock    types.TimeProvider
}

// NewDeploymentHandler creates a new instance of deployment validation hook
func NewDeploymentHandler(store types.ResourceStore, settings *config.Settings, clock types.TimeProvider) *hook.Handler {
	// Need little trick to protect internal data
	h := &DeploymentHandler{settings: settings}
	h.ObjectCreator = helper.NewStaticObjectCreator(&appsv1.Deployment{})
	h.Handler.Create = h.Create()
	h.Handler.Update = h.Update()
	h.Handler.Delete = h.Delete()
	h.Handler.Store = store
	h.clock = clock
	return &h.Handler
}

func (h *DeploymentHandler) Create() hook.AdmitFunc {
	return func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
		o, ok := obj.(*appsv1.Deployment)
		if !ok {
			log.Warn().Msg("unable to cast to deployment object instance")
			return &types.AdmissionResponse{Allowed: true}, nil
		}
		debugPrintObject(o, "deployment created")
		genericWriteDataToStorage(ctx, h.Store, h.clock, FormatDeploymentData(o, h.settings))
		return &types.AdmissionResponse{Allowed: true}, nil
	}
}

func (h *DeploymentHandler) Update() hook.AdmitFunc {
	return func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
		o, ok := obj.(*appsv1.Deployment)
		if !ok {
			log.Warn().Msg("unable to cast to deployment object instance")
			return &types.AdmissionResponse{Allowed: true}, nil
		}
		debugPrintObject(o, "deployment updated")
		genericWriteDataToStorage(ctx, h.Store, h.clock, FormatDeploymentData(o, h.settings))
		return &types.AdmissionResponse{Allowed: true}, nil
	}
}

func (h *DeploymentHandler) Delete() hook.AdmitFunc {
	return func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
		o, ok := obj.(*appsv1.Deployment)
		if !ok {
			log.Warn().Msg("unable to cast to deployment object instance")
			return &types.AdmissionResponse{Allowed: true}, nil
		}
		debugPrintObject(o, "deployment deleted")
		return &types.AdmissionResponse{Allowed: true}, nil
	}
}

func FormatDeploymentData(o *appsv1.Deployment, settings *config.Settings) types.ResourceTags {
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
		"resource_type": config.ResourceTypeToMetricName[config.Deployment],
	}
	return types.ResourceTags{
		Type:         config.Deployment,
		Name:         workload,
		Namespace:    &namespace,
		MetricLabels: &metricLabels,
		Labels:       &labels,
		Annotations:  &annotations,
	}
}
