// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

//nolint:dupl // Duplication is acceptable we expect to extend the definitions later
package handler

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/hook"
	"github.com/cloudzero/cloudzero-agent/app/types"
)

type PersistentVolumeConfigAccessor struct {
	settings *config.Settings
}

func NewPersistentVolumeConfigAccessor(settings *config.Settings) config.ConfigAccessor {
	return &PersistentVolumeConfigAccessor{settings: settings}
}

func (p *PersistentVolumeConfigAccessor) LabelsEnabled() bool {
	return false
}

func (p *PersistentVolumeConfigAccessor) AnnotationsEnabled() bool {
	return false
}

func (p *PersistentVolumeConfigAccessor) LabelsEnabledForType() bool {
	return false
}

func (p *PersistentVolumeConfigAccessor) AnnotationsEnabledForType() bool {
	return false
}

func (p *PersistentVolumeConfigAccessor) ResourceType() config.ResourceType {
	return config.PersistentVolume
}

func (p *PersistentVolumeConfigAccessor) Settings() *config.Settings {
	return p.settings
}

// NewPersistentVolumeHandler creates a new webhook handler for Kubernetes PersistentVolume resources.
func NewPersistentVolumeHandler[T metav1.Object](store types.ResourceStore, settings *config.Settings, clock types.TimeProvider, resource T) *hook.Handler {
	return NewGenericHandler[T](
		store,
		settings,
		clock,
		resource,
		NewPersistentVolumeConfigAccessor(settings),
		WorkloadDataFormatter, // TODO: using workload for now, I am sure this will change
	)
}
