// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
// nolint
package handler

import (
	"encoding/json"

	"github.com/rs/zerolog/log"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/hook"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/storage"

	v1 "k8s.io/api/apps/v1"
)

type DaemonSetHandler struct {
	hook.Handler
	settings *config.Settings
} // &v1.DaemonSet{}

func NewDaemonSetHandler(writer storage.DatabaseWriter,
	settings *config.Settings,
	errChan chan<- error,
) hook.Handler {
	// Need little trick to protect internal data
	h := &DaemonSetHandler{settings: settings}
	h.Handler.Create = h.Create()
	h.Handler.Update = h.Update()
	h.Handler.Writer = writer
	h.Handler.ErrorChan = errChan
	return h.Handler
}

func (h *DaemonSetHandler) Create() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		// only process if enabled, always return allowed to not block an admission
		if h.settings.Filters.Labels.Resources.DaemonSets || h.settings.Filters.Annotations.Resources.DaemonSets {
			if o, err := h.parseV1(r.Object.Raw); err == nil {
				h.writeDataToStorage(o, true)
			}
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (h *DaemonSetHandler) Update() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		// only process if enabled, always return allowed to not block an admission
		if h.settings.Filters.Labels.Resources.DaemonSets || h.settings.Filters.Annotations.Resources.DaemonSets {
			if o, err := h.parseV1(r.Object.Raw); err == nil {
				h.writeDataToStorage(o, false)
			}
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (h *DaemonSetHandler) parseV1(data []byte) (*v1.DaemonSet, error) {
	var o v1.DaemonSet
	if err := json.Unmarshal(data, &o); err != nil {
		return nil, err
	}
	return &o, nil
}

func (h *DaemonSetHandler) writeDataToStorage(o *v1.DaemonSet, isCreate bool) {
	record := FormatDaemonSetData(o, h.settings)
	if err := h.Writer.WriteData(record, isCreate); err != nil {
		log.Error().Err(err).Msgf("failed to write data to storage: %v", err)
	}
}

func FormatDaemonSetData(o *v1.DaemonSet, settings *config.Settings) storage.ResourceTags {
	namespace := o.GetNamespace()
	labels := config.Filter(o.GetLabels(), settings.LabelMatches, (settings.Filters.Labels.Enabled && settings.Filters.Labels.Resources.DaemonSets), settings)
	annotations := config.Filter(o.GetAnnotations(), settings.AnnotationMatches, (settings.Filters.Annotations.Enabled && settings.Filters.Annotations.Resources.DaemonSets), settings)
	metricLabels := config.MetricLabels{
		"workload":      o.GetName(), // standard metric labels to attach to metric
		"namespace":     namespace,
		"resource_type": config.ResourceTypeToMetricName[config.DaemonSet],
	}
	return storage.ResourceTags{
		Type:         config.DaemonSet,
		Name:         o.GetName(),
		Namespace:    &namespace,
		MetricLabels: &metricLabels,
		Labels:       &labels,
		Annotations:  &annotations,
	}
}
