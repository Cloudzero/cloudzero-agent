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

		sh.writeDataToStorage(s)
		if err != nil {
			return &hook.Result{Msg: err.Error()}, nil
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (sh *StatefulSetHandler) Update() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		s, err := sh.parseV1(r.Object.Raw)
		sh.writeDataToStorage(s)
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

func (sh *StatefulSetHandler) writeDataToStorage(s *v1.StatefulSet) {
	namespace := s.GetNamespace()
	labels := config.Filter(s.GetLabels(), sh.settings.LabelMatches, sh.settings.Filters.Labels.Enabled, *sh.settings)
	metricLabels := config.MetricLabels{
		"workload": s.GetName(), // standard metric labels to attach to metric
	}
	row := storage.ResourceTags{
		Name:         s.GetName(),
		Type:         config.StatefulSet,
		Namespace:    &namespace,
		MetricLabels: &metricLabels,
		Labels:       &labels,
	}
	if err := sh.Writer.WriteData(row); err != nil {
		log.Error().Err(err).Msgf("failed to write data to storage: %v", err)
	}
}
