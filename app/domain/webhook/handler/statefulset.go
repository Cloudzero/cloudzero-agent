// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/hook"
	"github.com/cloudzero/cloudzero-agent/app/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type StatefulSetConfigAccessor struct {
	settings *config.Settings
}

func NewStatefulSetConfigAccessor(settings *config.Settings) config.ConfigAccessor {
	return &StatefulSetConfigAccessor{settings: settings}
}

func (s *StatefulSetConfigAccessor) LabelsEnabled() bool {
	return s.settings.Filters.Labels.Enabled
}

func (s *StatefulSetConfigAccessor) AnnotationsEnabled() bool {
	return s.settings.Filters.Annotations.Enabled
}

func (s *StatefulSetConfigAccessor) LabelsEnabledForType() bool {
	return s.settings.Filters.Labels.Resources.StatefulSets
}

func (s *StatefulSetConfigAccessor) AnnotationsEnabledForType() bool {
	return s.settings.Filters.Annotations.Resources.StatefulSets
}

func (s *StatefulSetConfigAccessor) ResourceType() config.ResourceType {
	return config.StatefulSet
}

func (s *StatefulSetConfigAccessor) Settings() *config.Settings {
	return s.settings
}

// NewStatefulSetHandler creates a new webhook handler for Kubernetes StatefulSet resources.
// This handler is responsible for processing StatefulSet objects and applying the necessary
// filters and transformations based on the provided settings.
//
// Type Parameter:
//   - T: The type of the Kubernetes resource, which must implement the metav1.Object interface.
//     For this handler, it should be a StatefulSet resource, such as *apps/v1.StatefulSet, *apps/v1beta2.StatefulSet, or *apps/v1beta1.StatefulSet.
//
// Parameters:
//   - store: A ResourceStore instance used to manage the state of resources.
//   - settings: A pointer to the configuration settings that define filters and other options.
//   - clock: A TimeProvider instance used for time-related operations.
//   - resource: The StatefulSet resource to be processed by the handler.
//
// Returns:
//   - A pointer to a hook.Handler configured for StatefulSet resources.
func NewStatefulSetHandler[T metav1.Object](store types.ResourceStore, settings *config.Settings, clock types.TimeProvider, resource T) *hook.Handler {
	return NewGenericHandler[T](
		store,
		settings,
		clock,
		resource,
		NewStatefulSetConfigAccessor(settings),
		WorkloadDataFormatter,
	)
}
