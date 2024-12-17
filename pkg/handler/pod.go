// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
// nolint
package handler

import (
	"encoding/json"

	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/hook"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/storage"
)

type PodHandler struct {
	hook.Handler
	settings *config.Settings
} // &corev1.Pod{}

func NewPodHandler(writer storage.DatabaseWriter, settings *config.Settings, errChan chan<- error) hook.Handler {
	h := &PodHandler{settings: settings}
	h.Handler.Create = h.Create()
	h.Handler.Update = h.Update()
	h.Handler.Writer = writer
	h.Handler.ErrorChan = errChan
	return h.Handler
}

func (h *PodHandler) Create() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		// only process if enabled, always return allowed to not block an admission
		if h.settings.Filters.Labels.Resources.Pods || h.settings.Filters.Annotations.Resources.Pods {
			if o, err := h.parseV1(r.Object.Raw); err == nil {
				h.writeDataToStorage(o, true)
			}
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (h *PodHandler) Update() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		// only process if enabled, always return allowed to not block an admission
		if h.settings.Filters.Labels.Resources.Pods || h.settings.Filters.Annotations.Resources.Pods {
			if o, err := h.parseV1(r.Object.Raw); err == nil {
				h.writeDataToStorage(o, false)
			}
		}

		return &hook.Result{Allowed: true}, nil
	}
}

func (h *PodHandler) parseV1(data []byte) (*corev1.Pod, error) {
	var o corev1.Pod
	if err := json.Unmarshal(data, &o); err != nil {
		return nil, err
	}
	return &o, nil
}

func (h *PodHandler) writeDataToStorage(o *corev1.Pod, isCreate bool) {
	record := FormatPodData(o, h.settings)
	if err := h.Writer.WriteData(record, isCreate); err != nil {
		log.Error().Err(err).Msgf("failed to write data to storage: %v", err)
	}
}

func FormatPodData(o *corev1.Pod, settings *config.Settings) storage.ResourceTags {
	var (
		labels      config.MetricLabelTags = config.MetricLabelTags{}
		annotations config.MetricLabelTags = config.MetricLabelTags{}
		namespace                          = o.GetNamespace()
		podName                            = o.GetName()
	)
	if settings.Filters.Labels.Resources.Pods {
		labels = config.Filter(o.GetLabels(), settings.LabelMatches, (settings.Filters.Labels.Enabled && settings.Filters.Labels.Resources.Pods), settings)
	}
	if settings.Filters.Annotations.Resources.Pods {
		annotations = config.Filter(o.GetAnnotations(), settings.AnnotationMatches, (settings.Filters.Annotations.Enabled && settings.Filters.Annotations.Resources.Pods), settings)
	}
	metricLabels := config.MetricLabels{
		"pod":           podName, // standard metric labels to attach to metric
		"namespace":     namespace,
		"resource_type": config.ResourceTypeToMetricName[config.Pod],
	}
	return storage.ResourceTags{
		Type:         config.Pod,
		Name:         podName,
		Namespace:    &namespace,
		MetricLabels: &metricLabels,
		Labels:       &labels,
		Annotations:  &annotations,
	}
}
