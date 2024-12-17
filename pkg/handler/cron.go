// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
// nolint
package handler

import (
	"encoding/json"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/hook"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/storage"
	"github.com/rs/zerolog/log"
	batchv1 "k8s.io/api/batch/v1"
)

type CronJobHandler struct {
	hook.Handler
	settings *config.Settings
}

func NewCronJobHandler(writer storage.DatabaseWriter, settings *config.Settings, errChan chan<- error) hook.Handler {
	h := &CronJobHandler{settings: settings}
	h.Handler.Create = h.Create()
	h.Handler.Update = h.Update()
	h.Handler.Writer = writer
	h.Handler.ErrorChan = errChan
	return h.Handler
}

func (h *CronJobHandler) Create() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		// only process if enabled, always return allowed to not block an admission
		if h.settings.Filters.Labels.Resources.CronJobs || h.settings.Filters.Annotations.Resources.CronJobs {
			if o, err := h.parseV1(r.Object.Raw); err == nil {
				h.writeDataToStorage(o, true)
			}
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (h *CronJobHandler) Update() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		// only process if enabled, always return allowed to not block an admission
		if h.settings.Filters.Labels.Resources.CronJobs || h.settings.Filters.Annotations.Resources.CronJobs {
			if o, err := h.parseV1(r.Object.Raw); err == nil {
				h.writeDataToStorage(o, false)
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

func (h *CronJobHandler) writeDataToStorage(o *batchv1.CronJob, isCreate bool) {
	record := FormatCronJobData(o, h.settings)
	if err := h.Writer.WriteData(record, isCreate); err != nil {
		log.Error().Err(err).Msgf("failed to write data to storage: %v", err)
	}
}

func FormatCronJobData(o *batchv1.CronJob, settings *config.Settings) storage.ResourceTags {
	var (
		labels      config.MetricLabelTags = config.MetricLabelTags{}
		annotations config.MetricLabelTags = config.MetricLabelTags{}
		namespace                          = o.GetNamespace()
		workload                           = o.GetName()
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
	return storage.ResourceTags{
		Type:         config.CronJob,
		Name:         workload,
		Namespace:    &namespace,
		MetricLabels: &metricLabels,
		Labels:       &labels,
		Annotations:  &annotations,
	}
}
