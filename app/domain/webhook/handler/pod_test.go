// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package handler_test

import (
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"

	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/handler"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/hook"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/cloudzero/cloudzero-agent/app/types/mocks"
	"github.com/stretchr/testify/assert"
)

func TestNewPodHandler(t *testing.T) {
	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()

	clock := mocks.NewMockClock(time.Now())
	store := mocks.NewMockResourceStore(mockCtl)
	settings := &config.Settings{
		Filters: config.Filters{
			Labels: config.Labels{
				Enabled: true,
				Resources: config.Resources{
					Pods: true,
				},
			},
			Annotations: config.Annotations{
				Enabled: true,
				Resources: config.Resources{
					Pods: true,
				},
			},
		},
	}

	h := handler.NewPodHandler(store, settings, types.TimeProvider(clock), &corev1.Pod{})
	assert.NotNil(t, h, "Handler should not be nil")
	assert.IsType(t, &hook.Handler{}, h, "Handler should be of type *hook.Handler")
}

func TestPodConfigAccessor(t *testing.T) {
	settings := &config.Settings{
		Filters: config.Filters{
			Labels: config.Labels{
				Enabled: true,
				Resources: config.Resources{
					Pods: true,
				},
			},
			Annotations: config.Annotations{
				Enabled: false,
				Resources: config.Resources{
					Pods: false,
				},
			},
		},
	}

	accessor := handler.NewPodConfigAccessor(settings)

	t.Run("LabelsEnabled", func(t *testing.T) {
		assert.True(t, accessor.LabelsEnabled(), "LabelsEnabled should return true")
	})

	t.Run("AnnotationsEnabled", func(t *testing.T) {
		assert.False(t, accessor.AnnotationsEnabled(), "AnnotationsEnabled should return false")
	})

	t.Run("LabelsEnabledForType", func(t *testing.T) {
		assert.True(t, accessor.LabelsEnabledForType(), "LabelsEnabledForType should return true")
	})

	t.Run("AnnotationsEnabledForType", func(t *testing.T) {
		assert.False(t, accessor.AnnotationsEnabledForType(), "AnnotationsEnabledForType should return false")
	})

	t.Run("ResourceType", func(t *testing.T) {
		assert.Equal(t, config.Pod, accessor.ResourceType(), "ResourceType should return config.Pod")
	})

	t.Run("Settings", func(t *testing.T) {
		assert.Equal(t, settings, accessor.Settings(), "Settings should return the provided settings")
	})
}
