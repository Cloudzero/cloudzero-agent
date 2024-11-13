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

func NewDaemonSetHandler(writer storage.DatabaseWriter, settings *config.Settings, errChan chan<- error) hook.Handler {
	// Need little trick to protect internal data
	d := &DaemonSetHandler{settings: settings}
	d.Handler.Create = d.Create()
	d.Handler.Update = d.Update()
	d.Handler.Writer = writer
	d.Handler.ErrorChan = errChan
	return d.Handler
}

func (d *DaemonSetHandler) Create() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		ds, err := d.parseV1(r.Object.Raw)
		d.writeDataToStorage(ds, true)
		if err != nil {
			return &hook.Result{Msg: err.Error()}, nil
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (d *DaemonSetHandler) Update() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		ds, err := d.parseV1(r.Object.Raw)
		d.writeDataToStorage(ds, false)
		if err != nil {
			return &hook.Result{Msg: err.Error()}, nil
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (d *DaemonSetHandler) parseV1(object []byte) (*v1.DaemonSet, error) {
	var ds v1.DaemonSet
	if err := json.Unmarshal(object, &ds); err != nil {
		return nil, err
	}
	return &ds, nil
}

func (d *DaemonSetHandler) writeDataToStorage(ds *v1.DaemonSet, isCreate bool) {
	record := FormatDaemonSetData(ds, d.settings)
	if err := d.Writer.WriteData(record, isCreate); err != nil {
		log.Error().Err(err).Msgf("failed to write data to storage: %v", err)
	}
}

func FormatDaemonSetData(ds *v1.DaemonSet, settings *config.Settings) storage.ResourceTags {
	namespace := ds.GetNamespace()
	labels := config.Filter(ds.GetLabels(), settings.LabelMatches, settings.Filters.Labels.Enabled, *settings)
	annotations := config.Filter(ds.GetAnnotations(), settings.AnnotationMatches, settings.Filters.Annotations.Enabled, *settings)
	metricLabels := config.MetricLabels{
		"workload":      ds.GetName(), // standard metric labels to attach to metric
		"namespace":     namespace,
		"resource_type": config.ResourceTypeToMetricName[config.DaemonSet],
	}
	return storage.ResourceTags{
		Type:         config.DaemonSet,
		Name:         ds.GetName(),
		Namespace:    &namespace,
		MetricLabels: &metricLabels,
		Labels:       &labels,
		Annotations:  &annotations,
	}
}
