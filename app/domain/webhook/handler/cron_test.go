// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package handler_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	batchv1 "k8s.io/api/batch/v1"

	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/handler"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/hook"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/cloudzero/cloudzero-agent/app/types/mocks"
)

func TestNewCronJobHandler(t *testing.T) {
	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()

	clock := mocks.NewMockClock(time.Now())
	store := mocks.NewMockResourceStore(mockCtl)
	settings := &config.Settings{
		Filters: config.Filters{
			Labels: config.Labels{
				Enabled: true,
				Resources: config.Resources{
					CronJobs: true,
				},
			},
			Annotations: config.Annotations{
				Enabled: true,
				Resources: config.Resources{
					CronJobs: true,
				},
			},
		},
	}

	h := handler.NewCronJobHandler(store, settings, types.TimeProvider(clock), &batchv1.CronJob{})
	assert.NotNil(t, h, "Handler should not be nil")
	assert.IsType(t, &hook.Handler{}, h, "Handler should be of type *hook.Handler")
}

func TestCronJobConfigAccessor(t *testing.T) {
	settings := &config.Settings{
		Filters: config.Filters{
			Labels: config.Labels{
				Enabled: true,
				Resources: config.Resources{
					CronJobs: true,
				},
			},
			Annotations: config.Annotations{
				Enabled: false,
				Resources: config.Resources{
					CronJobs: false,
				},
			},
		},
	}

	accessor := handler.NewCronJobConfigAccessor(settings)

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
		assert.Equal(t, config.CronJob, accessor.ResourceType(), "ResourceType should return config.CronJob")
	})

	t.Run("Settings", func(t *testing.T) {
		assert.Equal(t, settings, accessor.Settings(), "Settings should return the provided settings")
	})
}
