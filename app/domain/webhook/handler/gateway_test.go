// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package handler_test

import (
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/handler"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/hook"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/cloudzero/cloudzero-agent/app/types/mocks"
	"github.com/stretchr/testify/assert"
)

func TestNewGatewayHandler(t *testing.T) {
	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()

	clock := mocks.NewMockClock(time.Now())
	store := mocks.NewMockResourceStore(mockCtl)
	settings := &config.Settings{
		Filters: config.Filters{
			Labels: config.Labels{
				Enabled: true,
			},
			Annotations: config.Annotations{
				Enabled: true,
			},
		},
	}

	h := handler.NewGatewayHandler(store, settings, types.TimeProvider(clock), &gatewayv1.Gateway{})
	assert.NotNil(t, h, "Handler should not be nil")
	assert.IsType(t, &hook.Handler{}, h, "Handler should be of type *hook.Handler")
}

func TestNewGatewayConfigAccessor(t *testing.T) {
	settings := &config.Settings{
		Filters: config.Filters{
			Labels: config.Labels{
				Enabled: true,
			},
			Annotations: config.Annotations{
				Enabled: true,
			},
		},
	}

	accessor := handler.NewGatewayConfigAccessor(settings)

	t.Run("LabelsEnabled", func(t *testing.T) {
		assert.False(t, accessor.LabelsEnabled(), "LabelsEnabled should return false")
	})

	t.Run("AnnotationsEnabled", func(t *testing.T) {
		assert.False(t, accessor.AnnotationsEnabled(), "AnnotationsEnabled should return false")
	})

	t.Run("LabelsEnabledForType", func(t *testing.T) {
		assert.False(t, accessor.LabelsEnabledForType(), "LabelsEnabledForType should return false")
	})

	t.Run("AnnotationsEnabledForType", func(t *testing.T) {
		assert.False(t, accessor.AnnotationsEnabledForType(), "AnnotationsEnabledForType should return false")
	})

	t.Run("ResourceType", func(t *testing.T) {
		assert.Equal(t, config.Gateway, accessor.ResourceType(), "ResourceType should return config.CustomResourceDefinition")
	})

	t.Run("Settings", func(t *testing.T) {
		assert.Equal(t, settings, accessor.Settings(), "Settings should return the provided settings")
	})
}
