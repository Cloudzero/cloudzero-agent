// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/hook"
	"github.com/cloudzero/cloudzero-agent/app/types"
)

type JobConfigAccessor struct {
	settings *config.Settings
}

func NewJobConfigAccessor(settings *config.Settings) *JobConfigAccessor {
	return &JobConfigAccessor{settings: settings}
}

func (j *JobConfigAccessor) LabelsEnabled() bool {
	return j.settings.Filters.Labels.Enabled
}

func (j *JobConfigAccessor) AnnotationsEnabled() bool {
	return j.settings.Filters.Annotations.Enabled
}

func (j *JobConfigAccessor) LabelsEnabledForType() bool {
	return j.settings.Filters.Labels.Resources.Jobs
}

func (j *JobConfigAccessor) AnnotationsEnabledForType() bool {
	return j.settings.Filters.Annotations.Resources.Jobs
}

func (j *JobConfigAccessor) ResourceType() config.ResourceType {
	return config.Job
}

func (j *JobConfigAccessor) Settings() *config.Settings {
	return j.settings
}

// NewJobHandler creates a new webhook handler for Kubernetes Job resources.
// This handler is responsible for processing Job objects and applying the necessary
// filters and transformations based on the provided settings.
//
// Type Parameter:
//   - T: The type of the Kubernetes resource, which must implement the metav1.Object interface.
//     For this handler, it should be a Job resource, such as *batchv1.Job.
//
// Parameters:
//   - store: A ResourceStore instance used to manage the state of resources.
//   - settings: A pointer to the configuration settings that define filters and other options.
//   - clock: A TimeProvider instance used for time-related operations.
//   - resource: The Job resource to be processed by the handler.
//
// Returns:
//   - A pointer to a hook.Handler configured for Job resources.
func NewJobHandler[T metav1.Object](store types.ResourceStore, settings *config.Settings, clock types.TimeProvider, resource T) *hook.Handler {
	return NewGenericHandler[T](
		store,
		settings,
		clock,
		resource,
		NewJobConfigAccessor(settings),
		WorkloadDataFormatter,
	)
}
