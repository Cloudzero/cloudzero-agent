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

type PersistentVolumeClaimConfigAccessor struct {
	settings *config.Settings
}

func NewPersistentVolumeClaimConfigAccessor(settings *config.Settings) config.ConfigAccessor {
	return &PersistentVolumeClaimConfigAccessor{settings: settings}
}

func (p *PersistentVolumeClaimConfigAccessor) LabelsEnabled() bool {
	return false
}

func (p *PersistentVolumeClaimConfigAccessor) AnnotationsEnabled() bool {
	return false
}

func (p *PersistentVolumeClaimConfigAccessor) LabelsEnabledForType() bool {
	return false
}

func (p *PersistentVolumeClaimConfigAccessor) AnnotationsEnabledForType() bool {
	return false
}

func (p *PersistentVolumeClaimConfigAccessor) ResourceType() config.ResourceType {
	return config.PersistentVolumeClaim
}

func (p *PersistentVolumeClaimConfigAccessor) Settings() *config.Settings {
	return p.settings
}

// NewPersistentVolumeClaimHandler creates a new webhook handler for Kubernetes PersistentVolumeClaim resources.
func NewPersistentVolumeClaimHandler[T metav1.Object](store types.ResourceStore, settings *config.Settings, clock types.TimeProvider, resource T) *hook.Handler {
	return NewGenericHandler[T](
		store,
		settings,
		clock,
		resource,
		NewPersistentVolumeClaimConfigAccessor(settings),
		WorkloadDataFormatter, // TODO: using workload for now, I am sure this will change
	)
}
