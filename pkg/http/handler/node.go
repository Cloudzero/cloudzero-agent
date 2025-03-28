// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

//nolint:dupl // There is currently substantial duplication in the handlers :(
package handler

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"

	"github.com/cloudzero/cloudzero-insights-controller/app/types"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/http/hook"
)

type NodeHandler struct {
	hook.Handler
	settings *config.Settings
	clock    types.TimeProvider
}

func NewNodeHandler(store types.ResourceStore, settings *config.Settings, clock types.TimeProvider, errChan chan<- error) hook.Handler {
	h := &NodeHandler{settings: settings}
	h.Handler.Create = h.Create()
	h.Handler.Update = h.Update()
	h.Handler.Store = store
	h.Handler.ErrorChan = errChan
	h.clock = clock
	return h.Handler
}

func (h *NodeHandler) Create() hook.AdmitFunc {
	return func(ctx context.Context, r *hook.Request) (*hook.Result, error) {
		// only process if enabled, always return allowed to not block an admission
		if h.settings.Filters.Labels.Resources.Nodes || h.settings.Filters.Annotations.Resources.Nodes {
			if o, err := h.parseV1(r.Object.Raw); err == nil {
				h.writeDataToStorage(ctx, o)
			}
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (h *NodeHandler) Update() hook.AdmitFunc {
	return func(ctx context.Context, r *hook.Request) (*hook.Result, error) {
		// only process if enabled, always return allowed to not block an admission
		if h.settings.Filters.Labels.Resources.Nodes || h.settings.Filters.Annotations.Resources.Nodes {
			if o, err := h.parseV1(r.Object.Raw); err == nil {
				h.writeDataToStorage(ctx, o)
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

func (h *NodeHandler) writeDataToStorage(ctx context.Context, o *corev1.Node) {
	record := FormatNodeData(o, h.settings)
	conditions := []interface{}{}
	if record.Namespace != nil {
		conditions = append(conditions, "type = ? AND name = ? AND namespace = ?", record.Type, record.Name, *record.Namespace)
	} else {
		conditions = append(conditions, "type = ? AND name = ?", record.Type, record.Name)
	}

	if found, err := h.Store.FindFirstBy(ctx, conditions...); (err != nil && errors.Is(err, types.ErrNotFound)) || found == nil {
		if err = h.Store.Tx(ctx, func(txCtx context.Context) error {
			return h.Store.Create(txCtx, &record)
		}); err != nil {
			log.Err(err).Msg("failed to write data to storage")
		}
	} else if found != nil {
		if err = h.Store.Tx(ctx, func(txCtx context.Context) error {
			record.ID = found.ID
			record.RecordCreated = found.RecordCreated
			record.RecordUpdated = h.clock.GetCurrentTime()
			record.SentAt = nil // reset send
			return h.Store.Update(txCtx, &record)
		}); err != nil {
			log.Err(err).Msg("failed to write data to storage")
		}
	} else {
		log.Err(err).Msg("failed to write data to storage")
	}
}

func FormatNodeData(o *corev1.Node, settings *config.Settings) types.ResourceTags {
	var (
		labels      = config.MetricLabelTags{}
		annotations = config.MetricLabelTags{}
		workload    = o.GetName()
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
	return types.ResourceTags{
		Name:         workload,
		Namespace:    nil,
		Type:         config.Node,
		MetricLabels: &metricLabels,
		Labels:       &labels,
		Annotations:  &annotations,
	}
}
