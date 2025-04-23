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

type NodeHandler struct {
	hook.Handler
	settings *config.Settings
	clock    types.TimeProvider
}

func NewNodeHandler(store types.ResourceStore, settings *config.Settings, clock types.TimeProvider) *hook.Handler {
	h := &NodeHandler{settings: settings}
	h.ObjectCreator = helper.NewStaticObjectCreator(&corev1.Node{})
	h.Handler.Create = h.Create()
	h.Handler.Update = h.Update()
	h.Handler.Delete = h.Delete()
	h.Handler.Store = store
	h.clock = clock
	return &h.Handler
}

func (h *NodeHandler) Create() hook.AdmitFunc {
	return func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
		o, ok := obj.(*corev1.Node)
		if !ok {
			log.Warn().Msg("unable to cast to node object instance")
			return &types.AdmissionResponse{Allowed: true}, nil
		}
		debugPrintObject(o, "node created")
		genericWriteDataToStorage(ctx, h.Store, h.clock, FormatNodeData(o, h.settings))
		return &types.AdmissionResponse{Allowed: true}, nil
	}
}

func (h *NodeHandler) Update() hook.AdmitFunc {
	return func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
		o, ok := obj.(*corev1.Node)
		if !ok {
			log.Warn().Msg("unable to cast to node object instance")
			return &types.AdmissionResponse{Allowed: true}, nil
		}
		debugPrintObject(o, "node updated")
		genericWriteDataToStorage(ctx, h.Store, h.clock, FormatNodeData(o, h.settings))
		return &types.AdmissionResponse{Allowed: true}, nil
	}
}

func (h *NodeHandler) Delete() hook.AdmitFunc {
	return func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
		o, ok := obj.(*corev1.Node)
		if !ok {
			log.Warn().Msg("unable to cast to node object instance")
			return &types.AdmissionResponse{Allowed: true}, nil
		}
		debugPrintObject(o, "node deleted")
		return &types.AdmissionResponse{Allowed: true}, nil
	}
}

func FormatNodeData(o *corev1.Node, settings *config.Settings) types.ResourceTags {
	var (
		labels      = config.MetricLabelTags{}
		annotations = config.MetricLabelTags{}
		workload    = o.GetName()
	)

	labels = config.Filter(o.GetLabels(), settings.LabelMatches, settings.Filters.Labels.Enabled, settings)
	annotations = config.Filter(o.GetAnnotations(), settings.AnnotationMatches, settings.Filters.Annotations.Enabled, settings)
	metricLabels := config.MetricLabels{
		"node":          workload, // standard metric labels to attach to metric
		"resource_type": config.ResourceTypeToMetricName[config.Node],
	}
	return types.ResourceTags{
		Name:         workload,
		Namespace:    nil,
		Type:         config.Node,
		MetricLabels: &metricLabels,
		Labels:       &labels,
		Annotations:  &annotations,
	}
}
