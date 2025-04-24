// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package handler_test

import (
	"context"
	"reflect"
	"regexp"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/handler"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/stretchr/testify/assert"
)

func TestFormatReplicaSetData(t *testing.T) {
	tests := []struct {
		name       string
		replicaset *appsv1.ReplicaSet
		settings   *config.Settings
		expected   types.ResourceTags
	}{
		{
			name: "Test with labels and annotations enabled",
			replicaset: &appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-replicaset",
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
				Type:      config.ReplicaSet,
				Name:      "test-replicaset",
				Namespace: stringPtr("default"),
				MetricLabels: &config.MetricLabels{
					"replicaset":    "test-replicaset",
					"namespace":     "default",
					"resource_type": "replicaset",
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
			replicaset: &appsv1.ReplicaSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-replicaset",
					Namespace: "default",
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
				Type:      config.ReplicaSet,
				Name:      "test-replicaset",
				Namespace: stringPtr("default"),
				MetricLabels: &config.MetricLabels{
					"replicaset":    "test-replicaset",
					"namespace":     "default",
					"resource_type": "replicaset",
				},
				Labels:      &config.MetricLabelTags{},
				Annotations: &config.MetricLabelTags{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.FormatReplicaSetData(tt.replicaset, tt.settings)
			if !reflect.DeepEqual(tt.expected.MetricLabels, result.MetricLabels) {
				t.Errorf("MetricLabels are not equal:\nExpected: %v\nGot: %v", tt.expected.MetricLabels, result.MetricLabels)
			}
			if !reflect.DeepEqual(tt.expected.Labels, result.Labels) {
				t.Errorf("Labels are not equal:\nExpected: %v\nGot: %v", tt.expected.Labels, result.Labels)
			}
			if !reflect.DeepEqual(tt.expected.Annotations, result.Annotations) {
				t.Errorf("Annotations are not equal:\nExpected: %v\nGot: %v", tt.expected.Annotations, result.Annotations)
			}
			assert.Equal(t, tt.expected.Type, result.Type)
			assert.Equal(t, tt.expected.Name, result.Name)
			assert.Equal(t, tt.expected.Namespace, result.Namespace)
		})
	}
}

func TestReplicaSetHandler_Create(t *testing.T) {
	handler := &handler.ReplicaSetHandler{}
	admissionReview := &types.AdmissionReview{}
	replicaSet := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-replicaset",
			Namespace: "default",
		},
	}

	admitFunc := handler.Create()
	response, err := admitFunc(context.Background(), admissionReview, replicaSet)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.True(t, response.Allowed)
}

func TestReplicaSetHandler_Update(t *testing.T) {
	handler := &handler.ReplicaSetHandler{}
	admissionReview := &types.AdmissionReview{}
	replicaSet := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-replicaset",
			Namespace: "default",
		},
	}

	admitFunc := handler.Update()
	response, err := admitFunc(context.Background(), admissionReview, replicaSet)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.True(t, response.Allowed)
}

func TestReplicaSetHandler_Delete(t *testing.T) {
	handler := &handler.ReplicaSetHandler{}
	admissionReview := &types.AdmissionReview{}
	replicaSet := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-replicaset",
			Namespace: "default",
		},
	}

	admitFunc := handler.Delete()
	response, err := admitFunc(context.Background(), admissionReview, replicaSet)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.True(t, response.Allowed)
}
