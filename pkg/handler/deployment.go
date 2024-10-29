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

type DeploymentHandler struct {
	hook.Handler
	settings *config.Settings
} // &v1.Deployment{}

// NewValidationHook creates a new instance of deployment validation hook
func NewDeploymentHandler(writer storage.DatabaseWriter, settings *config.Settings, errChan chan<- error) hook.Handler {
	// Need little trick to protect internal data
	d := &DeploymentHandler{settings: settings}
	d.Handler.Create = d.Create()
	d.Handler.Update = d.Update()
	d.Handler.Writer = writer
	d.Handler.ErrorChan = errChan
	return d.Handler
}

func (d *DeploymentHandler) Create() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		dp, err := d.parseV1(r.Object.Raw)
		d.writeDataToStorage(dp, true)
		if err != nil {
			return &hook.Result{Msg: err.Error()}, nil
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (d *DeploymentHandler) Update() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		dp, err := d.parseV1(r.Object.Raw)
		d.writeDataToStorage(dp, false)
		if err != nil {
			return &hook.Result{Msg: err.Error()}, nil
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (d *DeploymentHandler) parseV1(object []byte) (*v1.Deployment, error) {
	var dp v1.Deployment
	if err := json.Unmarshal(object, &dp); err != nil {
		return nil, err
	}
	return &dp, nil
}

func (d *DeploymentHandler) writeDataToStorage(dp *v1.Deployment, isCreate bool) {
	namespace := dp.GetNamespace()
	labels := config.Filter(dp.GetLabels(), d.settings.LabelMatches, d.settings.Filters.Labels.Enabled, *d.settings)
	annotations := config.Filter(dp.GetAnnotations(), d.settings.AnnotationMatches, d.settings.Filters.Annotations.Enabled, *d.settings)
	metricLabels := config.MetricLabels{
		"workload": dp.GetName(), // standard metric labels to attach to metric
	}
	row := storage.ResourceTags{
		Type:         config.Deployment,
		Name:         dp.GetName(),
		Namespace:    &namespace,
		MetricLabels: &metricLabels,
		Labels:       &labels,
		Annotations:  &annotations,
	}
	if err := d.Writer.WriteData(row, isCreate); err != nil {
		log.Error().Err(err).Msgf("failed to write data to storage: %v", err)
	}
}
