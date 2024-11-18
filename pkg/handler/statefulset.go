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
	v1 "k8s.io/api/apps/v1"
)

type StatefulSetHandler struct {
	hook.Handler
	settings *config.Settings
}

func NewStatefulsetHandler(writer storage.DatabaseWriter, settings *config.Settings, errChan chan<- error) hook.Handler {
	s := &StatefulSetHandler{settings: settings}
	s.Handler.Create = s.Create()
	s.Handler.Update = s.Update()
	s.Handler.Writer = writer
	s.Handler.ErrorChan = errChan
	return s.Handler
}

func (sh *StatefulSetHandler) Create() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		s, err := sh.parseV1(r.Object.Raw)

		sh.writeDataToStorage(s, true)
		if err != nil {
			return &hook.Result{Msg: err.Error()}, nil
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (sh *StatefulSetHandler) Update() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		s, err := sh.parseV1(r.Object.Raw)
		sh.writeDataToStorage(s, false)
		if err != nil {
			return &hook.Result{Msg: err.Error()}, nil
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (sh *StatefulSetHandler) parseV1(object []byte) (*v1.StatefulSet, error) {
	var s v1.StatefulSet
	if err := json.Unmarshal(object, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (sh *StatefulSetHandler) writeDataToStorage(s *v1.StatefulSet, isCreate bool) {
	record := FormatStatefulsetData(s, sh.settings)
	if err := sh.Writer.WriteData(record, isCreate); err != nil {
		log.Error().Err(err).Msgf("failed to write data to storage: %v", err)
	}
}

func FormatStatefulsetData(s *v1.StatefulSet, settings *config.Settings) storage.ResourceTags {
	namespace := s.GetNamespace()
	labels := config.Filter(s.GetLabels(), settings.LabelMatches, (settings.Filters.Labels.Enabled && settings.Filters.Labels.Resources.StatefulSets), *settings)
	annotations := config.Filter(s.GetAnnotations(), settings.AnnotationMatches, (settings.Filters.Annotations.Enabled && settings.Filters.Annotations.Resources.StatefulSets), *settings)
	metricLabels := config.MetricLabels{
		"workload":      s.GetName(), // standard metric labels to attach to metric
		"namespace":     namespace,
		"resource_type": config.ResourceTypeToMetricName[config.StatefulSet],
	}
	return storage.ResourceTags{
		Name:         s.GetName(),
		Type:         config.StatefulSet,
		Namespace:    &namespace,
		MetricLabels: &metricLabels,
		Labels:       &labels,
		Annotations:  &annotations,
	}
}
