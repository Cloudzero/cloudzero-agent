// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-Licenoe-Identifier: Apache-2.0
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

type NodeHandler struct {
	hook.Handler
	settings *config.Settings
} // &corev1.Node{}

func NewNodeHandler(writer storage.DatabaseWriter, settings *config.Settings, errChan chan<- error) hook.Handler {
	nh := &NodeHandler{settings: settings}
	nh.Handler.Create = nh.Create()
	nh.Handler.Update = nh.Update()
	nh.Handler.Writer = writer
	nh.Handler.ErrorChan = errChan
	return nh.Handler
}

func (nh *NodeHandler) Create() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		node, err := nh.parseV1(r.Object.Raw)

		nh.writeDataToStorage(node, true)
		if err != nil {
			return &hook.Result{Msg: err.Error()}, nil
		}
		return &hook.Result{Allowed: true}, nil
	}
}

func (nh *NodeHandler) Update() hook.AdmitFunc {
	return func(r *hook.Request) (*hook.Result, error) {
		node, err := nh.parseV1(r.Object.Raw)
		nh.writeDataToStorage(node, false)
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

func (nh *NodeHandler) writeDataToStorage(n *corev1.Node, isCreate bool) {
	record := FormatNodeData(n, nh.settings)
	if err := nh.Writer.WriteData(record, isCreate); err != nil {
		log.Error().Err(err).Msgf("failed to write data to storage: %v", err)
	}
}

func FormatNodeData(n *corev1.Node, settings *config.Settings) storage.ResourceTags {
	labels := config.Filter(n.GetLabels(), settings.LabelMatches, (settings.Filters.Labels.Enabled && settings.Filters.Labels.Resources.Nodes), settings)
	annotations := config.Filter(n.GetAnnotations(), settings.AnnotationMatches, (settings.Filters.Annotations.Enabled && settings.Filters.Annotations.Resources.Nodes), settings)
	metricLabels := config.MetricLabels{
		"node":          n.GetName(), // standard metric labels to attach to metric
		"resource_type": config.ResourceTypeToMetricName[config.Node],
	}
	return storage.ResourceTags{
		Name:         n.GetName(),
		Namespace:    nil,
		Type:         config.Node,
		MetricLabels: &metricLabels,
		Labels:       &labels,
		Annotations:  &annotations,
	}
}
