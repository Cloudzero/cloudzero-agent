// SPDX-License-IdentifierText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package handler_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/handler"
	"github.com/cloudzero/cloudzero-agent/app/types"
)

func TestWorkloadDataFormatter(t *testing.T) {
	tests := []struct {
		name     string
		accessor mockConfigAccessor
		obj      metav1.Object
		expected types.ResourceTags
	}{
		{
			name: "Labels and annotations enabled",
			accessor: mockConfigAccessor{
				labelsEnabled:             true,
				annotationsEnabled:        true,
				labelsEnabledForType:      true,
				annotationsEnabledForType: true,
				resourceType:              "pod",
				settings: &config.Settings{
					LabelMatches:      []string{"app"},
					AnnotationMatches: []string{"annotation-key"},
				},
			},
			obj: &metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "default",
				Labels: map[string]string{
					"app": "test",
				},
				Annotations: map[string]string{
					"annotation-key": "annotation-value",
				},
			},
			expected: types.ResourceTags{
				Type:      config.Pod,
				Name:      "test-pod",
				Namespace: stringPtr("default"),
				MetricLabels: &config.MetricLabels{
					"workload":      "test-pod",
					"namespace":     "default",
					"resource_type": "pod",
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
			name: "Labels and annotations disabled",
			accessor: mockConfigAccessor{
				labelsEnabled:             false,
				annotationsEnabled:        false,
				labelsEnabledForType:      false,
				annotationsEnabledForType: false,
				resourceType:              "pod",
				settings:                  &config.Settings{},
			},
			obj: &metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "default",
			},
			expected: types.ResourceTags{
				Type:      config.Pod,
				Name:      "test-pod",
				Namespace: stringPtr("default"),
				MetricLabels: &config.MetricLabels{
					"workload":      "test-pod",
					"namespace":     "default",
					"resource_type": "pod",
				},
				Labels:      &config.MetricLabelTags{},
				Annotations: &config.MetricLabelTags{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.WorkloadDataFormatter(tt.accessor, tt.obj)
			assert.Equal(t, tt.expected, result)
		})
	}
}

type mockConfigAccessor struct {
	labelsEnabled             bool
	annotationsEnabled        bool
	labelsEnabledForType      bool
	annotationsEnabledForType bool
	resourceType              string
	settings                  *config.Settings
}

func (m mockConfigAccessor) LabelsEnabled() bool {
	return m.labelsEnabled
}

func (m mockConfigAccessor) AnnotationsEnabled() bool {
	return m.annotationsEnabled
}

func (m mockConfigAccessor) LabelsEnabledForType() bool {
	return m.labelsEnabledForType
}

func (m mockConfigAccessor) AnnotationsEnabledForType() bool {
	return m.annotationsEnabledForType
}

func (m mockConfigAccessor) ResourceType() string {
	return m.resourceType
}

func (m mockConfigAccessor) Settings() *config.Settings {
	return m.settings
}

func stringPtr(s string) *string {
	return &s
}
