// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

//nolint:dupl // There is currently substantial duplication in the handlers :(
package handler

import (
	"context"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	config "github.com/cloudzero/cloudzero-agent/app/config/insights-controller"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/helper"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/hook"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/rs/zerolog/log"
)

type JobHandler struct {
	hook.Handler
	settings *config.Settings
	clock    types.TimeProvider
}

func NewJobHandler(store types.ResourceStore, settings *config.Settings, clock types.TimeProvider) *hook.Handler {
	h := &JobHandler{settings: settings}
	h.ObjectCreator = helper.NewStaticObjectCreator(&batchv1.Job{})
	h.Handler.Create = h.Create()
	h.Handler.Update = h.Update()
	h.Handler.Delete = h.Delete()
	h.Handler.Store = store
	h.clock = clock
	return &h.Handler
}

func (h *JobHandler) Create() hook.AdmitFunc {
	return func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
		o, ok := obj.(*batchv1.Job)
		if !ok {
			log.Warn().Msg("unable to cast to job object instance")
			return &types.AdmissionResponse{Allowed: true}, nil
		}
		debugPrintObject(o, "job created")
		if h.settings.Filters.Labels.Resources.Jobs || h.settings.Filters.Annotations.Resources.Jobs {
			genericWriteDataToStorage(ctx, h.Store, h.clock, FormatJobData(o, h.settings))
		}
		return &types.AdmissionResponse{Allowed: true}, nil
	}
}

func (h *JobHandler) Update() hook.AdmitFunc {
	return func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
		o, ok := obj.(*batchv1.Job)
		if !ok {
			log.Warn().Msg("unable to cast to job object instance")
			return &types.AdmissionResponse{Allowed: true}, nil
		}
		debugPrintObject(o, "job updated")
		if h.settings.Filters.Labels.Resources.Jobs || h.settings.Filters.Annotations.Resources.Jobs {
			genericWriteDataToStorage(ctx, h.Store, h.clock, FormatJobData(o, h.settings))
		}
		return &types.AdmissionResponse{Allowed: true}, nil
	}
}

func (h *JobHandler) Delete() hook.AdmitFunc {
	return func(ctx context.Context, r *types.AdmissionReview, obj metav1.Object) (*types.AdmissionResponse, error) {
		o, ok := obj.(*batchv1.Job)
		if !ok {
			log.Warn().Msg("unable to cast to job object instance")
			return &types.AdmissionResponse{Allowed: true}, nil
		}
		debugPrintObject(o, "job deleted")
		return &types.AdmissionResponse{Allowed: true}, nil
	}
}

func FormatJobData(o *batchv1.Job, settings *config.Settings) types.ResourceTags {
	var (
		labels      = config.MetricLabelTags{}
		annotations = config.MetricLabelTags{}
		namespace   = o.GetNamespace()
		workload    = o.GetName()
	)
	if settings.Filters.Labels.Enabled {
		labels = config.Filter(o.GetLabels(), settings.LabelMatches, (settings.Filters.Labels.Enabled && settings.Filters.Labels.Resources.Jobs), settings)
	}
	if settings.Filters.Annotations.Enabled {
		annotations = config.Filter(o.GetAnnotations(), settings.AnnotationMatches, (settings.Filters.Annotations.Enabled && settings.Filters.Annotations.Resources.Jobs), settings)
	}
	metricLabels := config.MetricLabels{
		"workload":      workload, // standard metric labels to attach to metric
		"namespace":     namespace,
		"resource_type": config.ResourceTypeToMetricName[config.Job],
	}
	return types.ResourceTags{
		Type:         config.Job,
		Name:         workload,
		Namespace:    &namespace,
		MetricLabels: &metricLabels,
		Labels:       &labels,
		Annotations:  &annotations,
	}
}
