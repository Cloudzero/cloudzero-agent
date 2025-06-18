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

type DeploymentConfigAccessor struct {
	settings *config.Settings
}

func NewDeploymentConfigAccessor(settings *config.Settings) config.ConfigAccessor {
	return &DeploymentConfigAccessor{
		settings: settings,
	}
}

func (d *DeploymentConfigAccessor) LabelsEnabled() bool {
	return d.settings.Filters.Labels.Enabled
}

func (d *DeploymentConfigAccessor) AnnotationsEnabled() bool {
	return d.settings.Filters.Annotations.Enabled
}

func (d *DeploymentConfigAccessor) LabelsEnabledForType() bool {
	return d.settings.Filters.Labels.Resources.Deployments
}

func (d *DeploymentConfigAccessor) AnnotationsEnabledForType() bool {
	return d.settings.Filters.Annotations.Resources.Deployments
}

func (d *DeploymentConfigAccessor) ResourceType() config.ResourceType {
	return config.Deployment
}

func (d *DeploymentConfigAccessor) Settings() *config.Settings {
	return d.settings
}

// NewDeploymentHandler creates a new webhook handler for Kubernetes Deployment resources.
// This handler is responsible for processing Deployment objects and applying the necessary
// filters and transformations based on the provided settings.
//
// Type Parameter:
//   - T: The type of the Kubernetes resource, which must implement the metav1.Object interface.
//     For this handler, it should be a Deployment resource, such as *appsv1.Deployment.
//
// Parameters:
//   - store: A ResourceStore instance used to manage the state of resources.
//   - settings: A pointer to the configuration settings that define filters and other options.
//   - clock: A TimeProvider instance used for time-related operations.
//   - resource: The Deployment resource to be processed by the handler.
//
// Returns:
//   - A pointer to a hook.Handler configured for Deployment resources.
func NewDeploymentHandler[T metav1.Object](store types.ResourceStore, settings *config.Settings, clock types.TimeProvider, resource T) *hook.Handler {
	return NewGenericHandler[T](
		store,
		settings,
		clock,
		resource,
		NewDeploymentConfigAccessor(settings),
		WorkloadDataFormatter,
	)
}
