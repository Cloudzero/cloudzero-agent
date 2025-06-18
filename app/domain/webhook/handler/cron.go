// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	batchv1 "k8s.io/api/batch/v1"

	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/hook"
	"github.com/cloudzero/cloudzero-agent/app/types"
)

type CronJobConfigAccessor struct {
	settings *config.Settings
}

func NewCronJobConfigAccessor(settings *config.Settings) config.ConfigAccessor {
	return &CronJobConfigAccessor{settings: settings}
}

func (c *CronJobConfigAccessor) LabelsEnabled() bool {
	return c.settings.Filters.Labels.Enabled
}

func (c *CronJobConfigAccessor) AnnotationsEnabled() bool {
	return c.settings.Filters.Annotations.Enabled
}

func (c *CronJobConfigAccessor) LabelsEnabledForType() bool {
	return c.settings.Filters.Labels.Resources.CronJobs
}

func (c *CronJobConfigAccessor) AnnotationsEnabledForType() bool {
	return c.settings.Filters.Annotations.Resources.CronJobs
}

func (c *CronJobConfigAccessor) ResourceType() config.ResourceType {
	return config.CronJob
}

func (c *CronJobConfigAccessor) Settings() *config.Settings {
	return c.settings
}

// NewCronJobHandler creates a new webhook handler for Kubernetes CronJob resources.
// This handler is responsible for processing CronJob objects and applying the necessary
// filters and transformations based on the provided settings.
//
// Type Parameter:
//   - T: The type of the Kubernetes resource, which must implement the metav1.Object interface.
//     For this handler, it should be a CronJob resource, such as *batchv1.CronJob.
//
// Parameters:
//   - store: A ResourceStore instance used to manage the state of resources.
//   - settings: A pointer to the configuration settings that define filters and other options.
//   - clock: A TimeProvider instance used for time-related operations.
//   - resource: The CronJob resource to be processed by the handler.
//
// Returns:
//   - A pointer to a hook.Handler configured for CronJob resources.
func NewCronJobHandler[T metav1.Object](store types.ResourceStore, settings *config.Settings, clock types.TimeProvider, resource T) *hook.Handler {
	return NewGenericHandler[T](
		store,
		settings,
		clock,
		resource,
		NewCronJobConfigAccessor(settings),
		WorkloadDataFormatter,
	)
}

// FormatCronJobData formats the data for a CronJob resource based on the provided settings.
func FormatCronJobData(o *batchv1.CronJob, settings *config.Settings) types.ResourceTags {
	var (
		labels      = config.MetricLabelTags{}
		annotations = config.MetricLabelTags{}
		namespace   = o.GetNamespace()
		workload    = o.GetName()
	)
	if settings.Filters.Labels.Enabled {
		labels = config.Filter(o.GetLabels(), settings.LabelMatches, (settings.Filters.Labels.Enabled && settings.Filters.Labels.Resources.CronJobs), settings)
	}
	if settings.Filters.Annotations.Enabled {
		annotations = config.Filter(o.GetAnnotations(), settings.AnnotationMatches, (settings.Filters.Annotations.Enabled && settings.Filters.Annotations.Resources.CronJobs), settings)
	}
	metricLabels := config.MetricLabels{
		"workload":      workload, // standard metric labels to attach to metric
		"namespace":     namespace,
		"resource_type": config.ResourceTypeToMetricName[config.CronJob],
	}
	return types.ResourceTags{
		Type:         config.CronJob,
		Name:         workload,
		Namespace:    &namespace,
		MetricLabels: &metricLabels,
		Labels:       &labels,
		Annotations:  &annotations,
	}
}
