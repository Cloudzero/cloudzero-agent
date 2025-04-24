// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package handler_test

import (
	"context"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/handler"
	"github.com/cloudzero/cloudzero-agent/app/types"
)

func TestFormatCustomResourceDefinitionData(t *testing.T) {
	tests := []struct {
		name     string
		object   *metav1.PartialObjectMetadata
		settings *config.Settings
		expected types.ResourceTags
	}{
		{
			name: "ValidDataWithLabelsAndAnnotations",
			object: &metav1.PartialObjectMetadata{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-crd",
					Namespace: "test-crd-namespace",
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
				Type:      config.CustomResourceDefinition,
				Name:      "test-crd",
				Namespace: stringPtr("test-crd-namespace"),
				MetricLabels: &config.MetricLabels{
					"workload":      "test-crd",
					"namespace":     "test-crd-namespace",
					"resource_type": config.ResourceTypeToMetricName[config.CustomResourceDefinition],
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
			object: &metav1.PartialObjectMetadata{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-crd",
					Namespace: "test-crd-namespace",
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
				Type:      config.CustomResourceDefinition,
				Name:      "test-crd",
				Namespace: stringPtr("test-crd-namespace"),
				MetricLabels: &config.MetricLabels{
					"workload":      "test-crd",
					"namespace":     "test-crd-namespace",
					"resource_type": config.ResourceTypeToMetricName[config.CustomResourceDefinition],
				},
				Labels:      &config.MetricLabelTags{},
				Annotations: &config.MetricLabelTags{},
			},
		},
		{
			name: "NoMatchingLabelsOrAnnotations",
			object: &metav1.PartialObjectMetadata{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-crd",
					Namespace: "test-crd-namespace",
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
				Type:      config.CustomResourceDefinition,
				Name:      "test-crd",
				Namespace: stringPtr("test-crd-namespace"),
				MetricLabels: &config.MetricLabels{
					"workload":      "test-crd",
					"namespace":     "test-crd-namespace",
					"resource_type": config.ResourceTypeToMetricName[config.CustomResourceDefinition],
				},
				Labels:      &config.MetricLabelTags{},
				Annotations: &config.MetricLabelTags{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.FormatCustomResourceDefinitionData(tt.object, tt.settings)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCustomResourceDefinitionHandler_Create(t *testing.T) {
	handler := &handler.CustomResourceDefinitionHandler{}
	admitFunc := handler.Create()

	tests := []struct {
		name     string
		object   metav1.Object
		expected *types.AdmissionResponse
	}{
		{
			name: "ValidCRDObject",
			object: &metav1.PartialObjectMetadata{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-crd",
					Namespace: "test-crd-namespace",
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

func TestCustomResourceDefinitionHandler_Update(t *testing.T) {
	handler := &handler.CustomResourceDefinitionHandler{}
	admitFunc := handler.Update()

	tests := []struct {
		name     string
		object   metav1.Object
		expected *types.AdmissionResponse
	}{
		{
			name: "ValidCRDObject",
			object: &metav1.PartialObjectMetadata{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-crd",
					Namespace: "test-crd-namespace",
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

func TestCustomResourceDefinitionHandler_Delete(t *testing.T) {
	handler := &handler.CustomResourceDefinitionHandler{}
	admitFunc := handler.Delete()

	tests := []struct {
		name     string
		object   metav1.Object
		expected *types.AdmissionResponse
	}{
		{
			name: "ValidCRDObject",
			object: &metav1.PartialObjectMetadata{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-crd",
					Namespace: "test-crd-namespace",
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
