// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

//nolint:dupl // Duplication is acceptable we expect to extend the definitions later
package handler

import (
	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/hook"
	"github.com/cloudzero/cloudzero-agent/app/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type NamespaceConfigAccessor struct {
	settings *config.Settings
}

func NewNamespaceConfigAccessor(settings *config.Settings) config.ConfigAccessor {
	return &NamespaceConfigAccessor{settings: settings}
}

func (n *NamespaceConfigAccessor) LabelsEnabled() bool {
	return n.settings.Filters.Labels.Enabled
}

func (n *NamespaceConfigAccessor) AnnotationsEnabled() bool {
	return n.settings.Filters.Annotations.Enabled
}

func (n *NamespaceConfigAccessor) LabelsEnabledForType() bool {
	return n.settings.Filters.Labels.Resources.Namespaces
}

func (n *NamespaceConfigAccessor) AnnotationsEnabledForType() bool {
	return n.settings.Filters.Annotations.Resources.Namespaces
}

func (n *NamespaceConfigAccessor) ResourceType() config.ResourceType {
	return config.Namespace
}

func (n *NamespaceConfigAccessor) Settings() *config.Settings {
	return n.settings
}

// NewNamespaceHandler creates a new webhook handler for Kubernetes Namespace resources.
// This handler is responsible for processing Namespace objects and applying the necessary
// filters and transformations based on the provided settings.
//
// Type Parameter:
//   - T: The type of the Kubernetes resource, which must implement the metav1.Object interface.
//     For this handler, it should be a Namespace resource, such as *corev1.Namespace.
//
// Parameters:
//   - store: A ResourceStore instance used to manage the state of resources.
//   - settings: A pointer to the configuration settings that define filters and other options.
//   - clock: A TimeProvider instance used for time-related operations.
//   - resource: The Namespace resource to be processed by the handler.
//
// Returns:
//   - A pointer to a hook.Handler configured for Namespace resources.
func NewNamespaceHandler[T metav1.Object](store types.ResourceStore, settings *config.Settings, clock types.TimeProvider, resource T) *hook.Handler {
	return NewGenericHandler[T](
		store,
		settings,
		clock,
		resource,
		NewNamespaceConfigAccessor(settings),
		NamespaceDataFormatter,
	)
}
