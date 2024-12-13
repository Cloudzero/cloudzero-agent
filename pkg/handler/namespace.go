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

type NamespaceHandler struct {
	hook.Handler
	settings *config.Settings
} // &corev1.nsd{}

func NewNamespaceHandler(writer storage.DatabaseWriter, settings *config.Settings, errChan chan<- error) hook.Handler {
	h := &NamespaceHandler{settings: settings}
	h.Handler.Create = h.Create()
	h.Handler.Update = h.Update()
	h.Handler.Writer = writer
	h.Handler.ErrorChan = errChan
	return h.Handler
}

func (nh *NamespaceHandler) Create() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		ns, err := nh.parseV1(r.Object.Raw)

		nh.writeDataToStorage(ns, false)
		if err != nil {
			return &hook.Result{Msg: err.Error()}, nil
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (nh *NamespaceHandler) Update() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		ns, err := nh.parseV1(r.Object.Raw)
		nh.writeDataToStorage(ns, false)
		if err != nil {
			return &hook.Result{Msg: err.Error()}, nil
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (nh *NamespaceHandler) parseV1(object []byte) (*corev1.Namespace, error) {
	var ns corev1.Namespace
	if err := json.Unmarshal(object, &ns); err != nil {
		return nil, err
	}
	return &ns, nil
}

func (nh *NamespaceHandler) writeDataToStorage(ns *corev1.Namespace, isCreate bool) {
	record := FormatNamespaceData(ns, nh.settings)
	if err := nh.Writer.WriteData(record, isCreate); err != nil {
		log.Error().Err(err).Msgf("failed to write data to storage: %v", err)
	}
}

func FormatNamespaceData(ns *corev1.Namespace, settings *config.Settings) storage.ResourceTags {
	labels := config.Filter(ns.GetLabels(), settings.LabelMatches, (settings.Filters.Labels.Enabled && settings.Filters.Labels.Resources.Namespaces), settings)
	annotations := config.Filter(ns.GetAnnotations(), settings.AnnotationMatches, (settings.Filters.Annotations.Enabled && settings.Filters.Annotations.Resources.Namespaces), settings)
	metricLabels := config.MetricLabels{
		"namespace":     ns.GetName(), // standard metric labels to attach to metric
		"resource_type": config.ResourceTypeToMetricName[config.Namespace],
	}
	return storage.ResourceTags{
		Name:         ns.GetName(),
		Namespace:    nil,
		Type:         config.Namespace,
		MetricLabels: &metricLabels,
		Labels:       &labels,
		Annotations:  &annotations,
	}
}
