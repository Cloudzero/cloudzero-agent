// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

//nolint:dupl // There is currently substantial duplication in the handlers :(
package handler

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/rs/zerolog/log"
	v1 "k8s.io/api/apps/v1"

	config "github.com/cloudzero/cloudzero-insights-controller/app/config/insights-controller"
	"github.com/cloudzero/cloudzero-insights-controller/app/types"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/http/hook"
)

type DeploymentHandler struct {
	hook.Handler
	settings *config.Settings
	clock    types.TimeProvider
}

// NewDeploymentHandler creates a new instance of deployment validation hook
func NewDeploymentHandler(store types.ResourceStore, settings *config.Settings, clock types.TimeProvider, errChan chan<- error) hook.Handler {
	// Need little trick to protect internal data
	d := &DeploymentHandler{settings: settings}
	d.Handler.Create = d.Create()
	d.Handler.Update = d.Update()
	d.Handler.Store = store
	d.Handler.ErrorChan = errChan
	d.clock = clock
	return d.Handler
}

func (h *DeploymentHandler) Create() hook.AdmitFunc {
	return func(ctx context.Context, r *hook.Request) (*hook.Result, error) {
		// only process if enabled, always return allowed to not block an admission
		if h.settings.Filters.Labels.Resources.Deployments || h.settings.Filters.Annotations.Resources.Deployments {
			if o, err := h.parseV1(r.Object.Raw); err == nil {
				h.writeDataToStorage(ctx, o)
			}
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (h *DeploymentHandler) Update() hook.AdmitFunc {
	return func(ctx context.Context, r *hook.Request) (*hook.Result, error) {
		// only process if enabled, always return allowed to not block an admission
		if h.settings.Filters.Labels.Resources.Deployments || h.settings.Filters.Annotations.Resources.Deployments {
			if o, err := h.parseV1(r.Object.Raw); err == nil {
				h.writeDataToStorage(ctx, o)
			}
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (h *DeploymentHandler) parseV1(data []byte) (*v1.Deployment, error) {
	var o v1.Deployment
	if err := json.Unmarshal(data, &o); err != nil {
		return nil, err
	}
	return &o, nil
}

func (h *DeploymentHandler) writeDataToStorage(ctx context.Context, o *v1.Deployment) {
	record := FormatDeploymentData(o, h.settings)
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

func FormatDeploymentData(o *v1.Deployment, settings *config.Settings) types.ResourceTags {
	var (
		labels      = config.MetricLabelTags{}
		annotations = config.MetricLabelTags{}
		namespace   = o.GetNamespace()
		workload    = o.GetName()
	)
	if settings.Filters.Labels.Resources.Deployments {
		labels = config.Filter(o.GetLabels(), settings.LabelMatches, (settings.Filters.Labels.Enabled && settings.Filters.Labels.Resources.Deployments), settings)
	}
	if settings.Filters.Annotations.Resources.Deployments {
		annotations = config.Filter(o.GetAnnotations(), settings.AnnotationMatches, (settings.Filters.Annotations.Enabled && settings.Filters.Annotations.Resources.Deployments), settings)
	}
	metricLabels := config.MetricLabels{
		"workload":      workload, // standard metric labels to attach to metric
		"namespace":     namespace,
		"resource_type": config.ResourceTypeToMetricName[config.Deployment],
	}
	return types.ResourceTags{
		Type:         config.Deployment,
		Name:         workload,
		Namespace:    &namespace,
		MetricLabels: &metricLabels,
		Labels:       &labels,
		Annotations:  &annotations,
	}
}
