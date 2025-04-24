// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

//nolint:dupl // There is currently substantial duplication in the handlers :(
package handler

import (
	corev1 "k8s.io/api/core/v1"

	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/hook"
	"github.com/cloudzero/cloudzero-agent/app/types"
)

type PodConfigAccessor struct {
	settings *config.Settings
}

func (p *PodConfigAccessor) LabelsEnabled() bool {
	return p.settings.Filters.Labels.Enabled
}

func (p *PodConfigAccessor) AnnotationsEnabled() bool {
	return p.settings.Filters.Annotations.Enabled
}

func (p *PodConfigAccessor) LabelsEnabledForType() bool {
	return p.settings.Filters.Labels.Resources.Pods
}

func (p *PodConfigAccessor) AnnotationsEnabledForType() bool {
	return p.settings.Filters.Annotations.Resources.Pods
}

func (p *PodConfigAccessor) ResourceType() string {
	return config.ResourceTypeToMetricName[config.Pod]
}

func (p *PodConfigAccessor) Settings() *config.Settings {
	return p.settings
}

func NewPodNewHandler(store types.ResourceStore, settings *config.Settings, clock types.TimeProvider) *hook.Handler {
	accessor := &PodConfigAccessor{settings: settings}
	return NewGenericHandler[*corev1.Pod](
		store,
		settings,
		clock,
		&corev1.Pod{},
		accessor,
		WorkloadDataFormatter,
	)
}
