// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

//nolint:dupl // There is currently substantial duplication in the handlers :(
package handler

import (
	"context"
	"encoding/json"

	batchv1 "k8s.io/api/batch/v1"

	config "github.com/cloudzero/cloudzero-agent-validator/app/config/insights-controller"
	"github.com/cloudzero/cloudzero-agent-validator/app/http/hook"
	"github.com/cloudzero/cloudzero-agent-validator/app/types"
)

type CronJobHandler struct {
	hook.Handler
	settings *config.Settings
	clock    types.TimeProvider
}

func NewCronJobHandler(store types.ResourceStore, settings *config.Settings, clock types.TimeProvider, errChan chan<- error) hook.Handler {
	h := &CronJobHandler{settings: settings}
	h.Handler.Create = h.Create()
	h.Handler.Update = h.Update()
	h.Handler.Store = store
	h.Handler.ErrorChan = errChan
	h.clock = clock
	return h.Handler
}

func (h *CronJobHandler) Create() hook.AdmitFunc {
	return func(ctx context.Context, r *hook.Request) (*hook.Result, error) {
		// only process if enabled, always return allowed to not block an admission
		if h.settings.Filters.Labels.Resources.CronJobs || h.settings.Filters.Annotations.Resources.CronJobs {
			if o, err := h.parseV1(r.Object.Raw); err == nil {
				h.writeDataToStorage(ctx, o)
			}
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (h *CronJobHandler) Update() hook.AdmitFunc {
	return func(ctx context.Context, r *hook.Request) (*hook.Result, error) {
		// only process if enabled, always return allowed to not block an admission
		if h.settings.Filters.Labels.Resources.CronJobs || h.settings.Filters.Annotations.Resources.CronJobs {
			if o, err := h.parseV1(r.Object.Raw); err == nil {
				h.writeDataToStorage(ctx, o)
			}
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (h *CronJobHandler) parseV1(data []byte) (*batchv1.CronJob, error) {
	var o batchv1.CronJob
	if err := json.Unmarshal(data, &o); err != nil {
		return nil, err
	}
	return &o, nil
}

func (h *CronJobHandler) writeDataToStorage(ctx context.Context, o *batchv1.CronJob) {
	genericWriteDataToStorage(ctx, h.Store, h.clock, FormatCronJobData(o, h.settings))
}

func FormatCronJobData(o *batchv1.CronJob, settings *config.Settings) types.ResourceTags {
	var (
		labels      = config.MetricLabelTags{}
		annotations = config.MetricLabelTags{}
		namespace   = o.GetNamespace()
		workload    = o.GetName()
	)
	if settings.Filters.Labels.Resources.CronJobs {
		labels = config.Filter(o.GetLabels(), settings.LabelMatches, (settings.Filters.Labels.Enabled && settings.Filters.Labels.Resources.CronJobs), settings)
	}
	if settings.Filters.Annotations.Resources.CronJobs {
		annotations = config.Filter(o.GetAnnotations(), settings.AnnotationMatches, (settings.Filters.Annotations.Enabled && settings.Filters.Annotations.Resources.CronJobs), settings)
	}
	metricLabels := config.MetricLabels{
		"workload":      workload, // standard metric labels to attach to metric
		"namespace":     namespace,
		"resource_type": config.ResourceTypeToMetricName[config.CronJob],
	}
	return types.ResourceTags{
		Type:         config.CronJob,
		Name:         workload,
		Namespace:    &namespace,
		MetricLabels: &metricLabels,
		Labels:       &labels,
		Annotations:  &annotations,
	}
}
