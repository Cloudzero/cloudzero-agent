// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

//nolint:dupl // Duplication is acceptable we expect to extend the definitions later
package handler

import (
	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/hook"
	"github.com/cloudzero/cloudzero-agent/app/types"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type IngressConfigAccessor struct {
	settings *config.Settings
}

func NewIngressConfigAccessor(settings *config.Settings) config.ConfigAccessor {
	return &IngressConfigAccessor{settings: settings}
}

func (i *IngressConfigAccessor) LabelsEnabled() bool {
	return false
}

func (i *IngressConfigAccessor) AnnotationsEnabled() bool {
	return false
}

func (i *IngressConfigAccessor) LabelsEnabledForType() bool {
	return false
}

func (i *IngressConfigAccessor) AnnotationsEnabledForType() bool {
	return false
}

func (i *IngressConfigAccessor) ResourceType() config.ResourceType {
	return config.Ingress
}

func (i *IngressConfigAccessor) Settings() *config.Settings {
	return i.settings
}

// NewIngressHandler creates a new webhook handler for Kubernetes Ingress resources.
func NewIngressHandler[T metav1.Object](store types.ResourceStore, settings *config.Settings, clock types.TimeProvider, resource T) *hook.Handler {
	return NewGenericHandler[*networkingv1.Ingress](
		store,
		settings,
		clock,
		resource,
		NewIngressConfigAccessor(settings),
		WorkloadDataFormatter,
	)
}
