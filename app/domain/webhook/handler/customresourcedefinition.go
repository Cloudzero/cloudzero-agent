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

type CustomResourceDefinitionConfigAccessor struct {
	settings *config.Settings
}

func NewCustomResourceDefinitionConfigAccessor(settings *config.Settings) config.ConfigAccessor {
	return &CustomResourceDefinitionConfigAccessor{settings: settings}
}

func (c *CustomResourceDefinitionConfigAccessor) LabelsEnabled() bool {
	return false
}

func (c *CustomResourceDefinitionConfigAccessor) AnnotationsEnabled() bool {
	return false
}

func (c *CustomResourceDefinitionConfigAccessor) LabelsEnabledForType() bool {
	return false
}

func (c *CustomResourceDefinitionConfigAccessor) AnnotationsEnabledForType() bool {
	return false
}

func (c *CustomResourceDefinitionConfigAccessor) ResourceType() config.ResourceType {
	return config.CustomResourceDefinition
}

func (c *CustomResourceDefinitionConfigAccessor) Settings() *config.Settings {
	return c.settings
}

// NewCustomResourceDefinitionHandler creates a new webhook handler for Kubernetes CustomResourceDefinition resources.
func NewCustomResourceDefinitionHandler[T metav1.Object](store types.ResourceStore, settings *config.Settings, clock types.TimeProvider, resource T) *hook.Handler {
	return NewGenericHandler[T](
		store,
		settings,
		clock,
		resource,
		NewCustomResourceDefinitionConfigAccessor(settings),
		WorkloadDataFormatter, // TODO: Might customize later
	)
}
