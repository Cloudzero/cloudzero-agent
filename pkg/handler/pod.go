// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package handler

import (
	"encoding/json"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/hook"
	remoteWrite "github.com/cloudzero/cloudzero-insights-controller/pkg/http"
	"github.com/prometheus/prometheus/prompb"
	corev1 "k8s.io/api/core/v1"
)

type PodHandler struct {
	hook.Handler
	settings *config.Settings
} // &corev1.Pod{}

func NewPodHandler(settings *config.Settings) hook.Handler {
	ph := &PodHandler{settings: settings}
	ph.Handler.Create = ph.Create()
	ph.Handler.Update = ph.Update()
	ph.Handler.Delete = ph.Delete()
	return ph.Handler
}

func (ph *PodHandler) Create() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		po, err := ph.parseV1(r.Object.Raw)

		go remoteWrite.PushLabels(ph.collectMetrics(*po), ph.settings)
		if err != nil {
			return &hook.Result{Msg: err.Error()}, nil
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (ph *PodHandler) Update() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		po, err := ph.parseV1(r.Object.Raw)
		go remoteWrite.PushLabels(ph.collectMetrics(*po), ph.settings)
		if err != nil {
			return &hook.Result{Msg: err.Error()}, nil
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (ph *PodHandler) Delete() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		_, err := ph.parseV1(r.OldObject.Raw)
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

func (ph *PodHandler) collectMetrics(po corev1.Pod) []prompb.TimeSeries {
	timeSeries := []prompb.TimeSeries{}
	metrics := map[string]map[string]string{
		"kube_pod_labels": po.GetLabels(),
	}
	for metricName, metricLabel := range metrics {
		timeSeries = append(timeSeries, remoteWrite.FormatMetrics(metricName, metricLabel))
	}
	return timeSeries
}
