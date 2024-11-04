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

type CronJobHandler struct {
	hook.Handler
	settings *config.Settings
}

func NewCronJobHandler(writer storage.DatabaseWriter, settings *config.Settings, errChan chan<- error) hook.Handler {
	cjh := &CronJobHandler{settings: settings}
	cjh.Handler.Create = cjh.Create()
	cjh.Handler.Update = cjh.Update()
	cjh.Handler.Writer = writer
	cjh.Handler.ErrorChan = errChan
	return cjh.Handler
}

func (cjh *CronJobHandler) Create() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		cj, err := cjh.parseV1(r.Object.Raw)

		cjh.writeDataToStorage(cj, false)
		if err != nil {
			return &hook.Result{Msg: err.Error()}, nil
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (cjh *CronJobHandler) Update() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		cj, err := cjh.parseV1(r.Object.Raw)
		cjh.writeDataToStorage(cj, false)
		if err != nil {
			return &hook.Result{Msg: err.Error()}, nil
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (cjh *CronJobHandler) parseV1(object []byte) (*batchv1.CronJob, error) {
	var cj batchv1.CronJob
	if err := json.Unmarshal(object, &cj); err != nil {
		return nil, err
	}
	return &cj, nil
}

func (cjh *CronJobHandler) writeDataToStorage(cj *batchv1.CronJob, isCreate bool) {
	namespace := cj.GetNamespace()
	labels := config.Filter(cj.GetLabels(), cjh.settings.LabelMatches, cjh.settings.Filters.Labels.Enabled, *cjh.settings)
	metricLabels := config.MetricLabels{
		"workload":      cj.GetName(), // standard metric labels to attach to metric
		"namespace":     namespace,
		"resource_type": config.ResourceTypeToMetricName[config.CronJob],
	}
	row := storage.ResourceTags{
		Type:         config.CronJob,
		Name:         cj.GetName(),
		Namespace:    &namespace,
		MetricLabels: &metricLabels,
		Labels:       &labels,
	}
	if err := cjh.Writer.WriteData(row, isCreate); err != nil {
		log.Error().Err(err).Msgf("failed to write data to storage: %v", err)
	}
}
