// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package handler

import (
	"encoding/json"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/prometheus/prometheus/prompb"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/hook"
	remoteWrite "github.com/cloudzero/cloudzero-insights-controller/pkg/http"
	v1 "k8s.io/api/apps/v1"
)

type DeploymentHandler struct {
	hook.Handler
	settings *config.Settings
} // &v1.Deployment{}

// NewValidationHook creates a new instance of deployment validation hook
func NewDeploymentHandler(settings *config.Settings) hook.Handler {
	// TODO: Need to accept a Data object
	// for saving records to the database

	// Need little trick to protect internal data
	d := &DeploymentHandler{settings: settings}
	d.Handler.Create = d.Create()
	d.Handler.Update = d.Update()
	d.Handler.Delete = d.Delete()
	return d.Handler
}

func (d *DeploymentHandler) Create() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		dp, err := d.parseV1(r.Object.Raw)

		go remoteWrite.PushLabels(d.collectMetrics(*dp), d.settings)
		if err != nil {
			return &hook.Result{Msg: err.Error()}, nil
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (d *DeploymentHandler) Update() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		dp, err := d.parseV1(r.Object.Raw)
		go remoteWrite.PushLabels(d.collectMetrics(*dp), d.settings)
		if err != nil {
			return &hook.Result{Msg: err.Error()}, nil
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (d *DeploymentHandler) Delete() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		_, err := d.parseV1(r.OldObject.Raw)
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

func (d *DeploymentHandler) collectMetrics(dp v1.Deployment) []prompb.TimeSeries {
	timeSeries := []prompb.TimeSeries{}
	metrics := map[string]map[string]string{
		"kube_deployment_labels":      dp.GetLabels(),
		"kube_deployment_annotations": dp.GetAnnotations(),
		"kube_pod_labels":             dp.Spec.Template.GetLabels(),
	}
	for metricName, metricLabel := range metrics {
		timeSeries = append(timeSeries, remoteWrite.FormatMetrics(metricName, metricLabel))
	}
	return timeSeries
}
