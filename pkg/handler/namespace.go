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

type NamespaceHandler struct {
	hook.Handler
	settings *config.Settings
} // &corev1.nsd{}

func NewNamespaceHandler(settings *config.Settings) hook.Handler {
	h := &NamespaceHandler{settings: settings}
	h.Handler.Create = h.Create()
	h.Handler.Update = h.Update()
	return h.Handler
}

func (nh *NamespaceHandler) Create() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		ns, err := nh.parseV1(r.Object.Raw)

		go remoteWrite.PushLabels(nh.collectMetrics(*ns), nh.settings)
		if err != nil {
			return &hook.Result{Msg: err.Error()}, nil
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (nh *NamespaceHandler) Update() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		ns, err := nh.parseV1(r.Object.Raw)
		go remoteWrite.PushLabels(nh.collectMetrics(*ns), nh.settings)
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

func (nh *NamespaceHandler) collectMetrics(ns corev1.Namespace) []prompb.TimeSeries {
	timeSeries := []prompb.TimeSeries{}
	metrics := map[string]map[string]string{
		"kube_namespace_labels": ns.GetLabels(),
	}
	for metricName, metricLabel := range metrics {
		timeSeries = append(timeSeries, remoteWrite.FormatMetrics(metricName, metricLabel))
	}
	return timeSeries
}
