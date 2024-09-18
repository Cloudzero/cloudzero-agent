// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-Licenoe-Identifier: Apache-2.0
// nolint
package handler

import (
	"encoding/json"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/hook"
	remoteWrite "github.com/cloudzero/cloudzero-insights-controller/pkg/http"
	"github.com/prometheus/prometheus/prompb"
	corev1 "k8s.io/api/core/v1"
)

type NodeHandler struct {
	hook.Handler
	settings *config.Settings
} // &corev1.Node{}

func NewNodeHandler(settings *config.Settings) hook.Handler {
	nh := &NodeHandler{settings: settings}
	nh.Handler.Create = nh.Create()
	nh.Handler.Update = nh.Update()
	return nh.Handler
}

func (nh *NodeHandler) Create() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		node, err := nh.parseV1(r.Object.Raw)

		go remoteWrite.PushLabels(nh.collectMetrics(*node), nh.settings)
		if err != nil {
			return &hook.Result{Msg: err.Error()}, nil
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (nh *NodeHandler) Update() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		node, err := nh.parseV1(r.Object.Raw)
		go remoteWrite.PushLabels(nh.collectMetrics(*node), nh.settings)
		if err != nil {
			return &hook.Result{Msg: err.Error()}, nil
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (nh *NodeHandler) parseV1(object []byte) (*corev1.Node, error) {
	var node corev1.Node
	if err := json.Unmarshal(object, &node); err != nil {
		return nil, err
	}
	return &node, nil
}

func (nh *NodeHandler) collectMetrics(n corev1.Node) []prompb.TimeSeries {
	additionalMetricLabels := config.MetricLabels{
		"node": n.GetName(), // standard metric labels to attach to metric
	}
	metrics := map[string]map[string]string{
		"kube_node_labels": config.Filter(n.GetLabels(), nh.settings.LabelMatches, nh.settings.Filters.Labels.Enabled),
	}
	return remoteWrite.FormatMetrics(metrics, additionalMetricLabels)
}
