// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/hook"
	"github.com/cloudzero/cloudzero-agent/app/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

type GatewayConfigAccessor struct {
	settings *config.Settings
}

func NewGatewayConfigAccessor(settings *config.Settings) config.ConfigAccessor {
	return &GatewayConfigAccessor{settings: settings}
}

func (g *GatewayConfigAccessor) LabelsEnabled() bool {
	return false
}

func (g *GatewayConfigAccessor) AnnotationsEnabled() bool {
	return false
}

func (g *GatewayConfigAccessor) LabelsEnabledForType() bool {
	return false
}

func (g *GatewayConfigAccessor) AnnotationsEnabledForType() bool {
	return false
}

func (g *GatewayConfigAccessor) ResourceType() config.ResourceType {
	return config.Gateway
}

func (g *GatewayConfigAccessor) Settings() *config.Settings {
	return g.settings
}

// NewGatewayHandler creates a new webhook handler for Kubernetes Gateway resources.
func NewGatewayHandler[T metav1.Object](store types.ResourceStore, settings *config.Settings, clock types.TimeProvider, resource T) *hook.Handler {
	return NewGenericHandler[*gatewayv1.Gateway](
		store,
		settings,
		clock,
		resource,
		NewGatewayConfigAccessor(settings),
		WorkloadDataFormatter, // TODO: use workflow for now
	)
}
