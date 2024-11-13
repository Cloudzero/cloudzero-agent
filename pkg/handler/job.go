// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
// nolint
package handler

import (
	"encoding/json"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/hook"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/storage"
	"github.com/rs/zerolog/log"
	batchv1 "k8s.io/api/batch/v1"
)

type JobHandler struct {
	hook.Handler
	settings *config.Settings
}

func NewJobHandler(writer storage.DatabaseWriter, settings *config.Settings, errChan chan<- error) hook.Handler {
	jh := &JobHandler{settings: settings}
	jh.Handler.Create = jh.Create()
	jh.Handler.Update = jh.Update()
	jh.Handler.Writer = writer
	jh.Handler.ErrorChan = errChan
	return jh.Handler
}

func (jh *JobHandler) Create() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		jo, err := jh.parseV1(r.Object.Raw)

		jh.writeDataToStorage(jo, false)
		if err != nil {
			return &hook.Result{Msg: err.Error()}, nil
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (jh *JobHandler) Update() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		jo, err := jh.parseV1(r.Object.Raw)
		jh.writeDataToStorage(jo, false)
		if err != nil {
			return &hook.Result{Msg: err.Error()}, nil
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (jh *JobHandler) parseV1(object []byte) (*batchv1.Job, error) {
	var jo batchv1.Job
	if err := json.Unmarshal(object, &jo); err != nil {
		return nil, err
	}
	return &jo, nil
}

func (jh *JobHandler) writeDataToStorage(jo *batchv1.Job, isCreate bool) {
	record := FormatJobData(jo, jh.settings)
	if err := jh.Writer.WriteData(record, isCreate); err != nil {
		log.Error().Err(err).Msgf("failed to write data to storage: %v", err)
	}
}

func FormatJobData(jo *batchv1.Job, settings *config.Settings) storage.ResourceTags {
	namespace := jo.GetNamespace()
	labels := config.Filter(jo.GetLabels(), settings.LabelMatches, settings.Filters.Labels.Enabled, *settings)
	annotations := config.Filter(jo.GetAnnotations(), settings.AnnotationMatches, settings.Filters.Annotations.Enabled, *settings)
	metricLabels := config.MetricLabels{
		"workload":      jo.GetName(), // standard metric labels to attach to metric
		"namespace":     namespace,
		"resource_type": config.ResourceTypeToMetricName[config.Job],
	}
	return storage.ResourceTags{
		Type:         config.Job,
		Name:         jo.GetName(),
		Namespace:    &namespace,
		MetricLabels: &metricLabels,
		Labels:       &labels,
		Annotations:  &annotations,
	}
}
