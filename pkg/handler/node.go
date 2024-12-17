// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-Licenoe-Identifier: Apache-2.0
// nolint
package handler

import (
	"encoding/json"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/hook"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/storage"
	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"
)

type NodeHandler struct {
	hook.Handler
	settings *config.Settings
} // &corev1.Node{}

func NewNodeHandler(writer storage.DatabaseWriter, settings *config.Settings, errChan chan<- error) hook.Handler {
	h := &NodeHandler{settings: settings}
	h.Handler.Create = h.Create()
	h.Handler.Update = h.Update()
	h.Handler.Writer = writer
	h.Handler.ErrorChan = errChan
	return h.Handler
}

func (h *NodeHandler) Create() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		// only process if enabled, always return allowed to not block an admission
		if h.settings.Filters.Labels.Resources.Nodes || h.settings.Filters.Annotations.Resources.Nodes {
			if o, err := h.parseV1(r.Object.Raw); err == nil {
				h.writeDataToStorage(o, true)
			}
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (h *NodeHandler) Update() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		// only process if enabled, always return allowed to not block an admission
		if h.settings.Filters.Labels.Resources.Nodes || h.settings.Filters.Annotations.Resources.Nodes {
			if o, err := h.parseV1(r.Object.Raw); err == nil {
				h.writeDataToStorage(o, false)
			}
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (h *NodeHandler) parseV1(data []byte) (*corev1.Node, error) {
	var o corev1.Node
	if err := json.Unmarshal(data, &o); err != nil {
		return nil, err
	}
	return &o, nil
}

func (h *NodeHandler) writeDataToStorage(o *corev1.Node, isCreate bool) {
	record := FormatNodeData(o, h.settings)
	if err := h.Writer.WriteData(record, isCreate); err != nil {
		log.Error().Err(err).Msgf("failed to write data to storage: %v", err)
	}
}

func FormatNodeData(o *corev1.Node, settings *config.Settings) storage.ResourceTags {
	var (
		labels      config.MetricLabelTags = config.MetricLabelTags{}
		annotations config.MetricLabelTags = config.MetricLabelTags{}
		workload                           = o.GetName()
	)

	if settings.Filters.Labels.Resources.Nodes {
		labels = config.Filter(o.GetLabels(), settings.LabelMatches, (settings.Filters.Labels.Enabled && settings.Filters.Labels.Resources.Nodes), settings)
	}
	if settings.Filters.Annotations.Resources.Nodes {
		annotations = config.Filter(o.GetAnnotations(), settings.AnnotationMatches, (settings.Filters.Annotations.Enabled && settings.Filters.Annotations.Resources.Nodes), settings)
	}

	metricLabels := config.MetricLabels{
		"node":          workload, // standard metric labels to attach to metric
		"resource_type": config.ResourceTypeToMetricName[config.Node],
	}
	return storage.ResourceTags{
		Name:         workload,
		Namespace:    nil,
		Type:         config.Node,
		MetricLabels: &metricLabels,
		Labels:       &labels,
		Annotations:  &annotations,
	}
}
