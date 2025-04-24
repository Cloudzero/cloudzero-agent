// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package handler_test

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"

	config "github.com/cloudzero/cloudzero-agent/app/config/insights-controller"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/handler"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/cloudzero/cloudzero-agent/app/types/mocks"
)

func TestFormatGatewayData(t *testing.T) {
	tests := []struct {
		name     string
		gateway  *gatewayv1.Gateway
		settings *config.Settings
		expected types.ResourceTags
	}{
		{
			name: "Test with labels and annotations enabled",
			gateway: &gatewayv1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-gateway",
					Namespace: "default",
					Labels: map[string]string{
						"app": "test",
					},
					Annotations: map[string]string{
						"annotation-key": "annotation-value",
					},
				},
			},
			settings: &config.Settings{
				Filters: config.Filters{
					Labels: config.Labels{
						Enabled: true,
					},
					Annotations: config.Annotations{
						Enabled: true,
					},
				},
				LabelMatches: []regexp.Regexp{
					*regexp.MustCompile("app"),
				},
				AnnotationMatches: []regexp.Regexp{
					*regexp.MustCompile("annotation-key"),
				},
			},
			expected: types.ResourceTags{
				Type:      config.Gateway,
				Name:      "test-gateway",
				Namespace: stringPtr("default"),
				MetricLabels: &config.MetricLabels{
					"workload":      "test-gateway",
					"namespace":     "default",
					"resource_type": "gateway",
				},
				Labels: &config.MetricLabelTags{
					"app": "test",
				},
				Annotations: &config.MetricLabelTags{
					"annotation-key": "annotation-value",
				},
			},
		},
		{
			name: "Test with labels and annotations disabled",
			gateway: &gatewayv1.Gateway{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-gateway",
					Namespace: "default",
					Labels: map[string]string{
						"app": "test",
					},
					Annotations: map[string]string{
						"annotation-key": "annotation-value",
					},
				},
			},
			settings: &config.Settings{
				Filters: config.Filters{
					Labels: config.Labels{
						Enabled: false,
					},
					Annotations: config.Annotations{
						Enabled: false,
					},
				},
			},
			expected: types.ResourceTags{
				Type:      config.Gateway,
				Name:      "test-gateway",
				Namespace: stringPtr("default"),
				MetricLabels: &config.MetricLabels{
					"workload":      "test-gateway",
					"namespace":     "default",
					"resource_type": "gateway",
				},
				Labels:      &config.MetricLabelTags{},
				Annotations: &config.MetricLabelTags{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.FormatGatewayData(tt.gateway, tt.settings)
			assert.Equal(t, tt.expected.Type, result.Type)
			assert.Equal(t, tt.expected.Name, result.Name)
			assert.Equal(t, tt.expected.Namespace, result.Namespace)
			assert.Equal(t, tt.expected.MetricLabels, result.MetricLabels)
			assert.Equal(t, tt.expected.Labels, result.Labels)
			assert.Equal(t, tt.expected.Annotations, result.Annotations)
		})
	}
}

func TestNewGatewayHandler(t *testing.T) {
	tests := []struct {
		name     string
		settings *config.Settings
	}{
		{
			name: "Test with valid settings",
			settings: &config.Settings{
				Filters: config.Filters{
					Labels: config.Labels{
						Enabled: true,
					},
					Annotations: config.Annotations{
						Enabled: true,
					},
				},
			},
		},
		{
			name:     "Test with nil settings",
			settings: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtl := gomock.NewController(t)
			defer mockCtl.Finish()
			store := mocks.NewMockResourceStore(mockCtl)
			clock := mocks.NewMockClock(time.Now())

			h := handler.NewGatewayHandler(store, tt.settings, clock)
			assert.NotNil(t, h)
			assert.Equal(t, store, h.Store)
			assert.NotNil(t, h.Create)
			assert.NotNil(t, h.Update)
			assert.NotNil(t, h.Delete)
		})
	}
}

func TestGatewayHandler_Create(t *testing.T) {
	handler := &handler.GatewayHandler{}
	createFunc := handler.Create()

	t.Run("Valid Gateway Object", func(t *testing.T) {
		gateway := &gatewayv1.Gateway{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-gateway",
				Namespace: "default",
			},
		}
		response, err := createFunc(context.TODO(), nil, gateway)
		assert.NoError(t, err)
		assert.True(t, response.Allowed)
	})

	t.Run("Invalid Object Type", func(t *testing.T) {
		invalidObj := &metav1.ObjectMeta{}
		response, err := createFunc(context.TODO(), nil, invalidObj)
		assert.NoError(t, err)
		assert.True(t, response.Allowed)
	})
}

func TestGatewayHandler_Update(t *testing.T) {
	handler := &handler.GatewayHandler{}
	updateFunc := handler.Update()

	t.Run("Valid Gateway Object", func(t *testing.T) {
		gateway := &gatewayv1.Gateway{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-gateway",
				Namespace: "default",
			},
		}
		response, err := updateFunc(context.TODO(), nil, gateway)
		assert.NoError(t, err)
		assert.True(t, response.Allowed)
	})

	t.Run("Invalid Object Type", func(t *testing.T) {
		invalidObj := &metav1.ObjectMeta{}
		response, err := updateFunc(context.TODO(), nil, invalidObj)
		assert.NoError(t, err)
		assert.True(t, response.Allowed)
	})
}

func TestGatewayHandler_Delete(t *testing.T) {
	handler := &handler.GatewayHandler{}
	deleteFunc := handler.Delete()

	t.Run("Valid Gateway Object", func(t *testing.T) {
		gateway := &gatewayv1.Gateway{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-gateway",
				Namespace: "default",
			},
		}
		response, err := deleteFunc(context.TODO(), nil, gateway)
		assert.NoError(t, err)
		assert.True(t, response.Allowed)
	})

	t.Run("Invalid Object Type", func(t *testing.T) {
		invalidObj := &metav1.ObjectMeta{}
		response, err := deleteFunc(context.TODO(), nil, invalidObj)
		assert.NoError(t, err)
		assert.True(t, response.Allowed)
	})
}
