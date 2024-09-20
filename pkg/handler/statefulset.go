// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
// nolint
package handler

import (
	"encoding/json"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/hook"
	remoteWrite "github.com/cloudzero/cloudzero-insights-controller/pkg/http"
	"github.com/prometheus/prometheus/prompb"
	v1 "k8s.io/api/apps/v1"
)

type StatefulSetHandler struct {
	hook.Handler
	settings *config.Settings
}

func NewStatefulsetHandler(settings *config.Settings) hook.Handler {
	s := &StatefulSetHandler{settings: settings}
	s.Handler.Create = s.Create()
	s.Handler.Update = s.Update()
	return s.Handler
}

func (sh *StatefulSetHandler) Create() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		s, err := sh.parseV1(r.Object.Raw)

		go remoteWrite.PushLabels(sh.collectMetrics(*s), sh.settings)
		if err != nil {
			return &hook.Result{Msg: err.Error()}, nil
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (sh *StatefulSetHandler) Update() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		s, err := sh.parseV1(r.Object.Raw)
		go remoteWrite.PushLabels(sh.collectMetrics(*s), sh.settings)
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

func (sh *StatefulSetHandler) collectMetrics(s v1.StatefulSet) []prompb.TimeSeries {
	additionalMetricLabels := config.MetricLabels{
		"workload": s.GetName(), // standard metric labels to attach to metric
	}
	metrics := map[string]map[string]string{
		"kube_statefulset_labels":      config.Filter(s.GetLabels(), sh.settings.LabelMatches, sh.settings.Filters.Labels.Enabled, *sh.settings),
		"kube_statefulset_annotations": config.Filter(s.GetAnnotations(), sh.settings.AnnotationMatches, sh.settings.Filters.Annotations.Enabled, *sh.settings),
	}
	return remoteWrite.FormatMetrics(metrics, additionalMetricLabels)
}
