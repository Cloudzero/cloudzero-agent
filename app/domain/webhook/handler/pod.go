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

type PodConfigAccessor struct {
	settings *config.Settings
}

func NewPodConfigAccessor(settings *config.Settings) config.ConfigAccessor {
	return &PodConfigAccessor{settings: settings}
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

func (p *PodConfigAccessor) ResourceType() config.ResourceType {
	return config.Pod
}

func (p *PodConfigAccessor) Settings() *config.Settings {
	return p.settings
}

func NewPodHandler[T metav1.Object](store types.ResourceStore, settings *config.Settings, clock types.TimeProvider, resource T) *hook.Handler {
	return NewGenericHandler[T](
		store,
		settings,
		clock,
		resource,
		NewPodConfigAccessor(settings),
		PodDataFormatter,
	)
}
