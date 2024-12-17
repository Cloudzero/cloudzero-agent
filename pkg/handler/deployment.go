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

type DeploymentHandler struct {
	hook.Handler
	settings *config.Settings
} // &v1.Deployment{}

// NewValidationHook creates a new instance of deployment validation hook
func NewDeploymentHandler(writer types.DatabaseWriter, settings *config.Settings, errChan chan<- error) hook.Handler {
	// Need little trick to protect internal data
	d := &DeploymentHandler{settings: settings}
	d.Handler.Create = d.Create()
	d.Handler.Update = d.Update()
	d.Handler.Writer = writer
	d.Handler.ErrorChan = errChan
	return d.Handler
}

func (h *DeploymentHandler) Create() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		// only process if enabled, always return allowed to not block an admission
		if h.settings.Filters.Labels.Resources.Deployments || h.settings.Filters.Annotations.Resources.Deployments {
			if o, err := h.parseV1(r.Object.Raw); err == nil {
				h.writeDataToStorage(o, true)
			}
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (h *DeploymentHandler) Update() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		// only process if enabled, always return allowed to not block an admission
		if h.settings.Filters.Labels.Resources.Deployments || h.settings.Filters.Annotations.Resources.Deployments {
			if o, err := h.parseV1(r.Object.Raw); err == nil {
				h.writeDataToStorage(o, false)
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

func (h *DeploymentHandler) writeDataToStorage(o *v1.Deployment, isCreate bool) {
	record := FormatDeploymentData(o, h.settings)
	if err := h.Writer.WriteData(record, isCreate); err != nil {
		log.Error().Err(err).Msgf("failed to write data to storage: %v", err)
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
