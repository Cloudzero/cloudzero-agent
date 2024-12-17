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

type JobHandler struct {
	hook.Handler
	settings *config.Settings
}

func NewJobHandler(writer storage.DatabaseWriter, settings *config.Settings, errChan chan<- error) hook.Handler {
	h := &JobHandler{settings: settings}
	h.Handler.Create = h.Create()
	h.Handler.Update = h.Update()
	h.Handler.Writer = writer
	h.Handler.ErrorChan = errChan
	return h.Handler
}

func (h *JobHandler) Create() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		// only process if enabled, always return allowed to not block an admission
		if h.settings.Filters.Labels.Resources.Jobs || h.settings.Filters.Annotations.Resources.Jobs {
			if o, err := h.parseV1(r.Object.Raw); err == nil {
				h.writeDataToStorage(o, true)
			}
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (h *JobHandler) Update() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		// only process if enabled, always return allowed to not block an admission
		if h.settings.Filters.Labels.Resources.Jobs || h.settings.Filters.Annotations.Resources.Jobs {
			if o, err := h.parseV1(r.Object.Raw); err == nil {
				h.writeDataToStorage(o, false)
			}
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (h *JobHandler) parseV1(data []byte) (*batchv1.Job, error) {
	var o batchv1.Job
	if err := json.Unmarshal(data, &o); err != nil {
		return nil, err
	}
	return &o, nil
}

func (h *JobHandler) writeDataToStorage(o *batchv1.Job, isCreate bool) {
	record := FormatJobData(o, h.settings)
	if err := h.Writer.WriteData(record, isCreate); err != nil {
		log.Error().Err(err).Msgf("failed to write data to storage: %v", err)
	}
}

func FormatJobData(o *batchv1.Job, settings *config.Settings) storage.ResourceTags {
	var (
		labels      config.MetricLabelTags = config.MetricLabelTags{}
		annotations config.MetricLabelTags = config.MetricLabelTags{}
		namespace                          = o.GetNamespace()
		workload                           = o.GetName()
	)
	if settings.Filters.Labels.Resources.Jobs {
		labels = config.Filter(o.GetLabels(), settings.LabelMatches, (settings.Filters.Labels.Enabled && settings.Filters.Labels.Resources.Jobs), settings)
	}
	if settings.Filters.Annotations.Resources.Jobs {
		annotations = config.Filter(o.GetAnnotations(), settings.AnnotationMatches, (settings.Filters.Annotations.Enabled && settings.Filters.Annotations.Resources.Jobs), settings)
	}

	metricLabels := config.MetricLabels{
		"workload":      workload, // standard metric labels to attach to metric
		"namespace":     namespace,
		"resource_type": config.ResourceTypeToMetricName[config.Job],
	}
	return storage.ResourceTags{
		Type:         config.Job,
		Name:         workload,
		Namespace:    &namespace,
		MetricLabels: &metricLabels,
		Labels:       &labels,
		Annotations:  &annotations,
	}
}
