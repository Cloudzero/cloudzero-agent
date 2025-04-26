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

type ServiceConfigAccessor struct {
	settings *config.Settings
}

func NewServiceConfigAccessor(settings *config.Settings) config.ConfigAccessor {
	return &ServiceConfigAccessor{settings: settings}
}

func (s *ServiceConfigAccessor) LabelsEnabled() bool {
	return false
}

func (s *ServiceConfigAccessor) AnnotationsEnabled() bool {
	return false
}

func (s *ServiceConfigAccessor) LabelsEnabledForType() bool {
	return false
}

func (s *ServiceConfigAccessor) AnnotationsEnabledForType() bool {
	return false
}

func (s *ServiceConfigAccessor) ResourceType() config.ResourceType {
	return config.Service
}

func (s *ServiceConfigAccessor) Settings() *config.Settings {
	return s.settings
}

// NewServiceHandler creates a new webhook handler for Kubernetes Service resources.
// This handler is responsible for processing Service objects and applying the necessary
// filters and transformations based on the provided settings.
//
// Type Parameter:
//   - T: The type of the Kubernetes resource, which must implement the metav1.Object interface.
//     For this handler, it should be a Service resource, such as *corev1.Service.
//
// Parameters:
//   - store: A ResourceStore instance used to manage the state of resources.
//   - settings: A pointer to the configuration settings that define filters and other options.
//   - clock: A TimeProvider instance used for time-related operations.
//   - resource: The Service resource to be processed by the handler.
//
// Returns:
//   - A pointer to a hook.Handler configured for Service resources.
func NewServiceHandler[T metav1.Object](store types.ResourceStore, settings *config.Settings, clock types.TimeProvider, resource T) *hook.Handler {
	return NewGenericHandler[T](
		store,
		settings,
		clock,
		resource,
		NewServiceConfigAccessor(settings),
		WorkloadDataFormatter,
	)
}
