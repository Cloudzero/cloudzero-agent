// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"context"
	"encoding/json"

	v1 "k8s.io/api/apps/v1"

	config "github.com/cloudzero/cloudzero-agent/app/config/insights-controller"
	"github.com/cloudzero/cloudzero-agent/app/http/hook"
	"github.com/cloudzero/cloudzero-agent/app/types"
)

type DaemonSetHandler struct {
	hook.Handler
	settings *config.Settings
	clock    types.TimeProvider
}

func NewDaemonSetHandler(store types.ResourceStore, settings *config.Settings, clock types.TimeProvider, errChan chan<- error,
) hook.Handler {
	// Need little trick to protect internal data
	h := &DaemonSetHandler{settings: settings}
	h.Handler.Create = h.Create()
	h.Handler.Update = h.Update()
	h.Handler.Store = store
	h.Handler.ErrorChan = errChan
	h.clock = clock
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
	genericWriteDataToStorage(ctx, h.Store, h.clock, FormatDaemonSetData(o, h.settings))
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
