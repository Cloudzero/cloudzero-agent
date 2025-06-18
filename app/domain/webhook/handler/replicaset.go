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

type ReplicaSetConfigAccessor struct {
	settings *config.Settings
}

func NewReplicaSetConfigAccessor(settings *config.Settings) config.ConfigAccessor {
	return &ReplicaSetConfigAccessor{settings: settings}
}

func (r *ReplicaSetConfigAccessor) LabelsEnabled() bool {
	return false
}

func (r *ReplicaSetConfigAccessor) AnnotationsEnabled() bool {
	return false
}

func (r *ReplicaSetConfigAccessor) LabelsEnabledForType() bool {
	return false
}

func (r *ReplicaSetConfigAccessor) AnnotationsEnabledForType() bool {
	return false
}

func (r *ReplicaSetConfigAccessor) ResourceType() config.ResourceType {
	return config.ReplicaSet
}

func (r *ReplicaSetConfigAccessor) Settings() *config.Settings {
	return r.settings
}

// NewReplicaSetHandler creates a new webhook handler for Kubernetes ReplicaSet resources.
// This handler is responsible for processing ReplicaSet objects and applying the necessary
// filters and transformations based on the provided settings.
//
// Type Parameter:
//   - T: The type of the Kubernetes resource, which must implement the metav1.Object interface.
//     For this handler, it should be a ReplicaSet resource, such as *apps/v1.ReplicaSet.
//
// Parameters:
//   - store: A ResourceStore instance used to manage the state of resources.
//   - settings: A pointer to the configuration settings that define filters and other options.
//   - clock: A TimeProvider instance used for time-related operations.
//   - resource: The ReplicaSet resource to be processed by the handler.
//
// Returns:
//   - A pointer to a hook.Handler configured for ReplicaSet resources.
func NewReplicaSetHandler[T metav1.Object](store types.ResourceStore, settings *config.Settings, clock types.TimeProvider, resource T) *hook.Handler {
	return NewGenericHandler[T](
		store,
		settings,
		clock,
		resource,
		NewReplicaSetConfigAccessor(settings),
		WorkloadDataFormatter,
	)
}
