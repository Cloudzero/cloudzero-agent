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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/handler"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/cloudzero/cloudzero-agent/app/types/mocks"
)

func TestNewPersistentVolumeClaimHandler(t *testing.T) {
	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()
	mockStore := mocks.NewMockResourceStore(mockCtl)
	mockClock := mocks.NewMockClock(time.Now())

	mockSettings := &config.Settings{}

	handler := handler.NewPersistentVolumeClaimHandler(mockStore, mockSettings, mockClock)

	assert.NotNil(t, handler)
	assert.NotNil(t, handler.Create)
	assert.NotNil(t, handler.Update)
	assert.NotNil(t, handler.Delete)
	assert.Equal(t, mockStore, handler.Store)
}

func TestFormatPersistentVolumeClaimData(t *testing.T) {
	tests := []struct {
		name     string
		object   *corev1.PersistentVolumeClaim
		settings *config.Settings
		expected types.ResourceTags
	}{
		{
			name: "ValidDataWithLabelsAndAnnotations",
			object: &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pvc",
					Namespace: "test-namespace",
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
				Type:      config.PersistentVolumeClaim,
				Name:      "test-pvc",
				Namespace: stringPtr("test-namespace"),
				MetricLabels: &config.MetricLabels{
					"workload":      "test-pvc",
					"namespace":     "test-namespace",
					"resource_type": config.ResourceTypeToMetricName[config.PersistentVolumeClaim],
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
			name: "ValidDataWithoutLabelsAndAnnotations",
			object: &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pvc",
					Namespace: "test-namespace",
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
				Type:      config.PersistentVolumeClaim,
				Name:      "test-pvc",
				Namespace: stringPtr("test-namespace"),
				MetricLabels: &config.MetricLabels{
					"workload":      "test-pvc",
					"namespace":     "test-namespace",
					"resource_type": config.ResourceTypeToMetricName[config.PersistentVolumeClaim],
				},
				Labels:      &config.MetricLabelTags{},
				Annotations: &config.MetricLabelTags{},
			},
		},
		{
			name: "NoMatchingLabelsOrAnnotations",
			object: &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pvc",
					Namespace: "test-namespace",
					Labels: map[string]string{
						"other-label": "value",
					},
					Annotations: map[string]string{
						"other-annotation": "value",
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
				Type:      config.PersistentVolumeClaim,
				Name:      "test-pvc",
				Namespace: stringPtr("test-namespace"),
				MetricLabels: &config.MetricLabels{
					"workload":      "test-pvc",
					"namespace":     "test-namespace",
					"resource_type": config.ResourceTypeToMetricName[config.PersistentVolumeClaim],
				},
				Labels:      &config.MetricLabelTags{},
				Annotations: &config.MetricLabelTags{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.FormatPersistentVolumeClaimData(tt.object, tt.settings)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPersistentVolumeClaimHandler_Create(t *testing.T) {
	handler := &handler.PersistentVolumeClaimHandler{}
	admitFunc := handler.Create()

	tests := []struct {
		name     string
		object   metav1.Object
		expected *types.AdmissionResponse
	}{
		{
			name: "ValidPVCObject",
			object: &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pvc",
					Namespace: "test-namespace",
				},
			},
			expected: &types.AdmissionResponse{Allowed: true},
		},
		{
			name:     "InvalidObjectType",
			object:   &metav1.ObjectMeta{},
			expected: &types.AdmissionResponse{Allowed: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := admitFunc(context.Background(), nil, tt.object)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, resp)
		})
	}
}

func TestPersistentVolumeClaimHandler_Update(t *testing.T) {
	handler := &handler.PersistentVolumeClaimHandler{}
	admitFunc := handler.Update()

	tests := []struct {
		name     string
		object   metav1.Object
		expected *types.AdmissionResponse
	}{
		{
			name: "ValidPVCObject",
			object: &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pvc",
					Namespace: "test-namespace",
				},
			},
			expected: &types.AdmissionResponse{Allowed: true},
		},
		{
			name:     "InvalidObjectType",
			object:   &metav1.ObjectMeta{},
			expected: &types.AdmissionResponse{Allowed: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := admitFunc(context.Background(), nil, tt.object)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, resp)
		})
	}
}

func TestPersistentVolumeClaimHandler_Delete(t *testing.T) {
	handler := &handler.PersistentVolumeClaimHandler{}
	admitFunc := handler.Delete()

	tests := []struct {
		name     string
		object   metav1.Object
		expected *types.AdmissionResponse
	}{
		{
			name: "ValidPVCObject",
			object: &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pvc",
					Namespace: "test-namespace",
				},
			},
			expected: &types.AdmissionResponse{Allowed: true},
		},
		{
			name:     "InvalidObjectType",
			object:   &metav1.ObjectMeta{},
			expected: &types.AdmissionResponse{Allowed: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := admitFunc(context.Background(), nil, tt.object)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, resp)
		})
	}
}
