// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
// nolint
package handler

import (
	"encoding/json"

	"github.com/rs/zerolog/log"
	v1 "k8s.io/api/apps/v1"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/hook"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/types"
)

type StatefulSetHandler struct {
	hook.Handler
	settings *config.Settings
}

func NewStatefulsetHandler(writer types.DatabaseWriter, settings *config.Settings, errChan chan<- error) hook.Handler {
	h := &StatefulSetHandler{settings: settings}
	h.Handler.Create = h.Create()
	h.Handler.Update = h.Update()
	h.Handler.Writer = writer
	h.Handler.ErrorChan = errChan
	return h.Handler
}

func (h *StatefulSetHandler) Create() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		// only process if enabled, always return allowed to not block an admission
		if h.settings.Filters.Labels.Resources.StatefulSets || h.settings.Filters.Annotations.Resources.StatefulSets {
			if o, err := h.parseV1(r.Object.Raw); err == nil {
				h.writeDataToStorage(o, true)
			}
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (h *StatefulSetHandler) Update() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		// only process if enabled, always return allowed to not block an admission
		if h.settings.Filters.Labels.Resources.StatefulSets || h.settings.Filters.Annotations.Resources.StatefulSets {
			if o, err := h.parseV1(r.Object.Raw); err == nil {
				h.writeDataToStorage(o, false)
			}
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (h *StatefulSetHandler) parseV1(data []byte) (*v1.StatefulSet, error) {
	var o v1.StatefulSet
	if err := json.Unmarshal(data, &o); err != nil {
		return nil, err
	}
	return &o, nil
}

func (h *StatefulSetHandler) writeDataToStorage(o *v1.StatefulSet, isCreate bool) {
	record := FormatStatefulsetData(o, h.settings)
	if err := h.Writer.WriteData(record, isCreate); err != nil {
		log.Error().Err(err).Msgf("failed to write data to storage: %v", err)
	}
}

func FormatStatefulsetData(o *v1.StatefulSet, settings *config.Settings) types.ResourceTags {
	var (
		labels      = config.MetricLabelTags{}
		annotations = config.MetricLabelTags{}
		namespace   = o.GetNamespace()
		workload    = o.GetName()
	)
	if settings.Filters.Labels.Resources.StatefulSets {
		labels = config.Filter(o.GetLabels(), settings.LabelMatches, (settings.Filters.Labels.Enabled && settings.Filters.Labels.Resources.StatefulSets), settings)
	}
	if settings.Filters.Annotations.Resources.StatefulSets {
		annotations = config.Filter(o.GetAnnotations(), settings.AnnotationMatches, (settings.Filters.Annotations.Enabled && settings.Filters.Annotations.Resources.StatefulSets), settings)
	}
	metricLabels := config.MetricLabels{
		"workload":      workload, // standard metric labels to attach to metric
		"namespace":     namespace,
		"resource_type": config.ResourceTypeToMetricName[config.StatefulSet],
	}
	return types.ResourceTags{
		Name:         workload,
		Type:         config.StatefulSet,
		Namespace:    &namespace,
		MetricLabels: &metricLabels,
		Labels:       &labels,
		Annotations:  &annotations,
	}
}
