// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

//nolint:dupl // Duplication is acceptable we expect to extend the definitions later
package handler

import (
	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/hook"
	"github.com/cloudzero/cloudzero-agent/app/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NodeConfigAccessor struct {
	settings *config.Settings
}

func NewNodeConfigAccessor(settings *config.Settings) config.ConfigAccessor {
	return &NodeConfigAccessor{settings: settings}
}

func (n *NodeConfigAccessor) LabelsEnabled() bool {
	return n.settings.Filters.Labels.Enabled
}

func (n *NodeConfigAccessor) AnnotationsEnabled() bool {
	return n.settings.Filters.Annotations.Enabled
}

func (n *NodeConfigAccessor) LabelsEnabledForType() bool {
	return n.settings.Filters.Labels.Resources.Nodes
}

func (n *NodeConfigAccessor) AnnotationsEnabledForType() bool {
	return n.settings.Filters.Annotations.Resources.Nodes
}

func (n *NodeConfigAccessor) ResourceType() config.ResourceType {
	return config.Node
}

func (n *NodeConfigAccessor) Settings() *config.Settings {
	return n.settings
}

// NewNodeHandler creates a new webhook handler for Kubernetes Node resources.
// This handler processes Node objects and applies filters and transformations based on the provided settings.
func NewNodeHandler[T metav1.Object](store types.ResourceStore, settings *config.Settings, clock types.TimeProvider, resource T) *hook.Handler {
	return NewGenericHandler[T](
		store,
		settings,
		clock,
		resource,
		NewNodeConfigAccessor(settings),
		NodeDataFormatter,
	)
}
