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
	corev1 "k8s.io/api/core/v1"
)

type PodHandler struct {
	hook.Handler
	settings *config.Settings
} // &corev1.Pod{}

func NewPodHandler(writer storage.DatabaseWriter, settings *config.Settings, errChan chan<- error) hook.Handler {
	ph := &PodHandler{settings: settings}
	ph.Handler.Create = ph.Create()
	ph.Handler.Update = ph.Update()
	ph.Handler.Writer = writer
	ph.Handler.ErrorChan = errChan
	return ph.Handler
}

func (ph *PodHandler) Create() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		po, err := ph.parseV1(r.Object.Raw)

		ph.writeDataToStorage(po, false)
		if err != nil {
			return &hook.Result{Msg: err.Error()}, nil
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (ph *PodHandler) Update() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		po, err := ph.parseV1(r.Object.Raw)
		ph.writeDataToStorage(po, false)
		if err != nil {
			return &hook.Result{Msg: err.Error()}, nil
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (ph *PodHandler) parseV1(object []byte) (*corev1.Pod, error) {
	var po corev1.Pod
	if err := json.Unmarshal(object, &po); err != nil {
		return nil, err
	}
	return &po, nil
}

func (ph *PodHandler) writeDataToStorage(po *corev1.Pod, isCreate bool) {
	record := FormatPodData(po, ph.settings)
	if err := ph.Writer.WriteData(record, isCreate); err != nil {
		log.Error().Err(err).Msgf("failed to write data to storage: %v", err)
	}
}

func FormatPodData(po *corev1.Pod, settings *config.Settings) storage.ResourceTags {
	namespace := po.GetNamespace()
	labels := config.Filter(po.GetLabels(), settings.LabelMatches, (settings.Filters.Labels.Enabled && settings.Filters.Labels.Resources.Pods), *settings)
	annotations := config.Filter(po.GetAnnotations(), settings.AnnotationMatches, (settings.Filters.Annotations.Enabled && settings.Filters.Annotations.Resources.Pods), *settings)
	metricLabels := config.MetricLabels{
		"pod":           po.GetName(), // standard metric labels to attach to metric
		"namespace":     namespace,
		"resource_type": config.ResourceTypeToMetricName[config.Pod],
	}
	return storage.ResourceTags{
		Type:         config.Pod,
		Name:         po.GetName(),
		Namespace:    &namespace,
		MetricLabels: &metricLabels,
		Labels:       &labels,
		Annotations:  &annotations,
	}
}
