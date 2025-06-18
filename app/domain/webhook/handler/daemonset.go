// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

//nolint:dupl // Duplication is acceptable we expect to extend the definitions later
package handler

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/hook"
	"github.com/cloudzero/cloudzero-agent/app/types"
)

type DaemonSetConfigAccessor struct {
	settings *config.Settings
}

func NewDaemonSetConfigAccessor(settings *config.Settings) config.ConfigAccessor {
	return &DaemonSetConfigAccessor{settings: settings}
}

func (d *DaemonSetConfigAccessor) LabelsEnabled() bool {
	return d.settings.Filters.Labels.Enabled
}

func (d *DaemonSetConfigAccessor) AnnotationsEnabled() bool {
	return d.settings.Filters.Annotations.Enabled
}

func (d *DaemonSetConfigAccessor) LabelsEnabledForType() bool {
	return d.settings.Filters.Labels.Resources.DaemonSets
}

func (d *DaemonSetConfigAccessor) AnnotationsEnabledForType() bool {
	return d.settings.Filters.Annotations.Resources.DaemonSets
}

func (d *DaemonSetConfigAccessor) ResourceType() config.ResourceType {
	return config.DaemonSet
}

func (d *DaemonSetConfigAccessor) Settings() *config.Settings {
	return d.settings
}

// NewDaemonSetHandler creates a new webhook handler for Kubernetes DaemonSet resources.
// This handler is responsible for processing DaemonSet objects and applying the necessary
// filters and transformations based on the provided settings.
//
// Type Parameter:
//   - T: The type of the Kubernetes resource, which must implement the metav1.Object interface.
//     For this handler, it should be a DaemonSet resource, such as *appsv1.DaemonSet.
//
// Parameters:
//   - store: A ResourceStore instance used to manage the state of resources.
//   - settings: A pointer to the configuration settings that define filters and other options.
//   - clock: A TimeProvider instance used for time-related operations.
//   - resource: The DaemonSet resource to be processed by the handler.
//
// Returns:
//   - A pointer to a hook.Handler configured for DaemonSet resources.
func NewDaemonSetHandler[T metav1.Object](store types.ResourceStore, settings *config.Settings, clock types.TimeProvider, resource T) *hook.Handler {
	return NewGenericHandler[T](
		store,
		settings,
		clock,
		resource,
		NewDaemonSetConfigAccessor(settings),
		WorkloadDataFormatter,
	)
}
