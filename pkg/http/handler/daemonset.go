// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

//nolint:dupl,gofmt
package handler

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/rs/zerolog/log"
	v1 "k8s.io/api/apps/v1"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/http/hook"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/types"
)

type DaemonSetHandler struct {
	hook.Handler
	settings *config.Settings
} // &v1.DaemonSet{}

func NewDaemonSetHandler(store types.ResourceStore, settings *config.Settings, errChan chan<- error,
) hook.Handler {
	// Need little trick to protect internal data
	h := &DaemonSetHandler{settings: settings}
	h.Handler.Create = h.Create()
	h.Handler.Update = h.Update()
	h.Handler.Store = store
	h.Handler.ErrorChan = errChan
	return h.Handler
}

func (h *DaemonSetHandler) Create() hook.AdmitFunc {
	return func(ctx context.Context, r *hook.Request) (*hook.Result, error) {
		// only process if enabled, always return allowed to not block an admission
		if h.settings.Filters.Labels.Resources.DaemonSets || h.settings.Filters.Annotations.Resources.DaemonSets {
			if o, err := h.parseV1(r.Object.Raw); err == nil {
				h.writeDataToStorage(ctx, o)
			}
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (h *DaemonSetHandler) Update() hook.AdmitFunc {
	return func(ctx context.Context, r *hook.Request) (*hook.Result, error) {
		// only process if enabled, always return allowed to not block an admission
		if h.settings.Filters.Labels.Resources.DaemonSets || h.settings.Filters.Annotations.Resources.DaemonSets {
			if o, err := h.parseV1(r.Object.Raw); err == nil {
				h.writeDataToStorage(ctx, o)
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

func (h *DaemonSetHandler) writeDataToStorage(ctx context.Context, o *v1.DaemonSet) {
	record := FormatDaemonSetData(o, h.settings)
	conditions := []interface{}{}
	if record.Namespace != nil {
		conditions = append(conditions, "type = ? AND name = ? AND namespace = ?", record.Type, record.Name, *record.Namespace)
	} else {
		conditions = append(conditions, "type = ? AND name = ?", record.Type, record.Name)
	}

	if found, err := h.Store.FindFirstBy(ctx, conditions...); (err != nil && errors.Is(err, types.ErrNotFound)) || found == nil {
		if err := h.Store.Tx(ctx, func(txCtx context.Context) error {
			return h.Store.Create(txCtx, &record)
		}); err != nil {
			log.Error().Err(err).Msgf("failed to write data to storage: %v", err)
		}
	} else if found != nil {
		if err := h.Store.Tx(ctx, func(txCtx context.Context) error {
			record.ID = found.ID
			record.RecordCreated = found.RecordCreated
			record.SentAt = nil // reset send
			return h.Store.Update(txCtx, &record)
		}); err != nil {
			log.Error().Err(err).Msgf("failed to write data to storage: %v", err)
		}
	} else {
		log.Error().Err(err).Msgf("failed to write data to storage: %v", err)
	}
}

func FormatDaemonSetData(o *v1.DaemonSet, settings *config.Settings) types.ResourceTags {
	namespace := o.GetNamespace()
	labels := config.Filter(o.GetLabels(), settings.LabelMatches, (settings.Filters.Labels.Enabled && settings.Filters.Labels.Resources.DaemonSets), settings)
	annotations := config.Filter(o.GetAnnotations(), settings.AnnotationMatches, (settings.Filters.Annotations.Enabled && settings.Filters.Annotations.Resources.DaemonSets), settings)
	metricLabels := config.MetricLabels{
		"workload":      o.GetName(), // standard metric labels to attach to metric
		"namespace":     namespace,
		"resource_type": config.ResourceTypeToMetricName[config.DaemonSet],
	}
	return types.ResourceTags{
		Type:         config.DaemonSet,
		Name:         o.GetName(),
		Namespace:    &namespace,
		MetricLabels: &metricLabels,
		Labels:       &labels,
		Annotations:  &annotations,
	}
}
