// SPDX-License-IdentifierText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package handler_test

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/handler"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/helper"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/hook"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/cloudzero/cloudzero-agent/app/types/mocks"
)

func TestDataFormatters(t *testing.T) {
	tests := []struct {
		name      string
		formatter handler.DataFormatter
		accessor  mockConfigAccessor
		obj       metav1.Object
		expected  types.ResourceTags
	}{
		{
			name:      "WorkloadDataFormatter with labels and annotations enabled",
			formatter: handler.WorkloadDataFormatter,
			accessor: mockConfigAccessor{
				labelsEnabled:             true,
				annotationsEnabled:        true,
				labelsEnabledForType:      true,
				annotationsEnabledForType: true,
				resourceType:              config.Pod,
				settings: &config.Settings{
					LabelMatches: []regexp.Regexp{
						*regexp.MustCompile("app"),
					},
					AnnotationMatches: []regexp.Regexp{
						*regexp.MustCompile("annotation-key"),
					},
				},
			},
			obj: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
					Labels: map[string]string{
						"app": "test",
					},
					Annotations: map[string]string{
						"annotation-key": "annotation-value",
					},
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
			name:      "WorkloadDataFormatter with labels and annotations disabled",
			formatter: handler.WorkloadDataFormatter,
			accessor: mockConfigAccessor{
				labelsEnabled:             false,
				annotationsEnabled:        false,
				labelsEnabledForType:      false,
				annotationsEnabledForType: false,
				resourceType:              config.Pod,
				settings:                  &config.Settings{},
			},
			obj: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
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
				Labels:      &config.MetricLabelTags{},
				Annotations: &config.MetricLabelTags{},
			},
		},
		{
			name:      "PodDataFormatter with labels and annotations enabled",
			formatter: handler.PodDataFormatter,
			accessor: mockConfigAccessor{
				labelsEnabled:             true,
				annotationsEnabled:        true,
				labelsEnabledForType:      true,
				annotationsEnabledForType: true,
				resourceType:              config.Pod,
				settings: &config.Settings{
					LabelMatches: []regexp.Regexp{
						*regexp.MustCompile("app"),
					},
					AnnotationMatches: []regexp.Regexp{
						*regexp.MustCompile("annotation-key"),
					},
				},
			},
			obj: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
					Labels: map[string]string{
						"app": "test",
					},
					Annotations: map[string]string{
						"annotation-key": "annotation-value",
					},
				},
			},
			expected: types.ResourceTags{
				Type:      config.Pod,
				Name:      "test-pod",
				Namespace: stringPtr("default"),
				MetricLabels: &config.MetricLabels{
					"pod":           "test-pod",
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
			name:      "PodDataFormatter with labels and annotations disabled",
			formatter: handler.PodDataFormatter,
			accessor: mockConfigAccessor{
				labelsEnabled:             false,
				annotationsEnabled:        false,
				labelsEnabledForType:      false,
				annotationsEnabledForType: false,
				resourceType:              config.Pod,
				settings:                  &config.Settings{},
			},
			obj: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
				},
			},
			expected: types.ResourceTags{
				Type:      config.Pod,
				Name:      "test-pod",
				Namespace: stringPtr("default"),
				MetricLabels: &config.MetricLabels{
					"pod":           "test-pod",
					"namespace":     "default",
					"resource_type": "pod",
				},
				Labels:      &config.MetricLabelTags{},
				Annotations: &config.MetricLabelTags{},
			},
		},
		{
			name:      "NamespaceDataFormatter with labels and annotations enabled",
			formatter: handler.NamespaceDataFormatter,
			accessor: mockConfigAccessor{
				labelsEnabled:             true,
				annotationsEnabled:        true,
				labelsEnabledForType:      true,
				annotationsEnabledForType: true,
				resourceType:              config.Namespace,
				settings: &config.Settings{
					LabelMatches: []regexp.Regexp{
						*regexp.MustCompile("env"),
					},
					AnnotationMatches: []regexp.Regexp{
						*regexp.MustCompile("annotation-key"),
					},
				},
			},
			obj: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-namespace",
					Labels: map[string]string{
						"env": "prod",
					},
					Annotations: map[string]string{
						"annotation-key": "annotation-value",
					},
				},
			},
			expected: types.ResourceTags{
				Type:      config.Namespace,
				Name:      "test-namespace",
				Namespace: nil,
				MetricLabels: &config.MetricLabels{
					"namespace":     "test-namespace",
					"resource_type": "namespace",
				},
				Labels: &config.MetricLabelTags{
					"env": "prod",
				},
				Annotations: &config.MetricLabelTags{
					"annotation-key": "annotation-value",
				},
			},
		},
		{
			name:      "NamespaceDataFormatter with labels and annotations disabled",
			formatter: handler.NamespaceDataFormatter,
			accessor: mockConfigAccessor{
				labelsEnabled:             false,
				annotationsEnabled:        false,
				labelsEnabledForType:      false,
				annotationsEnabledForType: false,
				resourceType:              config.Namespace,
				settings:                  &config.Settings{},
			},
			obj: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-namespace",
				},
			},
			expected: types.ResourceTags{
				Type:      config.Namespace,
				Name:      "test-namespace",
				Namespace: nil,
				MetricLabels: &config.MetricLabels{
					"namespace":     "test-namespace",
					"resource_type": "namespace",
				},
				Labels:      &config.MetricLabelTags{},
				Annotations: &config.MetricLabelTags{},
			},
		},
		{
			name:      "NodeDataFormatter with labels and annotations enabled",
			formatter: handler.NodeDataFormatter,
			accessor: mockConfigAccessor{
				labelsEnabled:             true,
				annotationsEnabled:        true,
				labelsEnabledForType:      true,
				annotationsEnabledForType: true,
				resourceType:              config.Node,
				settings: &config.Settings{
					LabelMatches: []regexp.Regexp{
						*regexp.MustCompile("role"),
					},
					AnnotationMatches: []regexp.Regexp{
						*regexp.MustCompile("annotation-key"),
					},
				},
			},
			obj: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node",
					Labels: map[string]string{
						"role": "worker",
					},
					Annotations: map[string]string{
						"annotation-key": "annotation-value",
					},
				},
			},
			expected: types.ResourceTags{
				Type:      config.Node,
				Name:      "test-node",
				Namespace: nil,
				MetricLabels: &config.MetricLabels{
					"node":          "test-node",
					"resource_type": "node",
				},
				Labels: &config.MetricLabelTags{
					"role": "worker",
				},
				Annotations: &config.MetricLabelTags{
					"annotation-key": "annotation-value",
				},
			},
		},
		{
			name:      "NodeDataFormatter with labels and annotations disabled",
			formatter: handler.NodeDataFormatter,
			accessor: mockConfigAccessor{
				labelsEnabled:             false,
				annotationsEnabled:        false,
				labelsEnabledForType:      false,
				annotationsEnabledForType: false,
				resourceType:              config.Node,
				settings:                  &config.Settings{},
			},
			obj: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node",
				},
			},
			expected: types.ResourceTags{
				Type:      config.Node,
				Name:      "test-node",
				Namespace: nil,
				MetricLabels: &config.MetricLabels{
					"node":          "test-node",
					"resource_type": "node",
				},
				Labels:      &config.MetricLabelTags{},
				Annotations: &config.MetricLabelTags{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.formatter(tt.accessor, tt.obj)

			if diff := cmp.Diff(tt.expected.MetricLabels, result.MetricLabels); diff != "" {
				t.Errorf("MetricLabels mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.expected.Labels, result.Labels); diff != "" {
				t.Errorf("Labels mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.expected.Annotations, result.Annotations); diff != "" {
				t.Errorf("Annotations mismatch (-want +got):\n%s", diff)
			}
			assert.Equal(t, tt.expected.Type, result.Type)
			assert.Equal(t, tt.expected.Name, result.Name)
			if tt.expected.Namespace != nil && *tt.expected.Namespace != "" {
				assert.Equal(t, tt.expected.Namespace, result.Namespace)
			}
		})
	}
}

func TestGenericHandler_Create(t *testing.T) {
	initialTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	mockClock := mocks.NewMockClock(initialTime)

	tests := []struct {
		name     string
		accessor mockConfigAccessor
		request  *types.AdmissionReview
		expected *types.AdmissionResponse
	}{
		{
			name: "Create with labels and annotations enabled",
			accessor: mockConfigAccessor{
				labelsEnabled:             true,
				annotationsEnabled:        true,
				labelsEnabledForType:      true,
				annotationsEnabledForType: true,
				resourceType:              config.Pod,
				settings: &config.Settings{
					LabelMatches: []regexp.Regexp{
						*regexp.MustCompile("app"),
					},
					AnnotationMatches: []regexp.Regexp{
						*regexp.MustCompile("annotation-key"),
					},
				},
			},
			request: &types.AdmissionReview{
				NewObjectRaw: getRawObject(appsv1.SchemeGroupVersion, &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
						Labels: map[string]string{
							"app": "test",
						},
						Annotations: map[string]string{
							"annotation-key": "annotation-value",
						},
					},
				}),
			},
			expected: &types.AdmissionResponse{Allowed: true},
		},
		{
			name: "Create with labels and annotations disabled",
			accessor: mockConfigAccessor{
				labelsEnabled:             false,
				annotationsEnabled:        false,
				labelsEnabledForType:      false,
				annotationsEnabledForType: false,
				resourceType:              config.Pod,
				settings:                  &config.Settings{},
			},
			request: &types.AdmissionReview{
				NewObjectRaw: getRawObject(appsv1.SchemeGroupVersion, &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
					},
				}),
			},
			expected: &types.AdmissionResponse{Allowed: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtl := gomock.NewController(t)
			defer mockCtl.Finish()

			clock := mockClock
			store := mocks.NewMockResourceStore(mockCtl)

			if tt.accessor.settings != nil && len(tt.accessor.settings.LabelMatches) > 0 {
				store.EXPECT().FindFirstBy(gomock.Any(), gomock.Any()).Return(nil, nil)
				store.EXPECT().Tx(gomock.Any(), gomock.Any()).Return(nil)
				store.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
			}

			h := handler.NewGenericHandler[*corev1.Pod](store, tt.accessor.settings, clock, &corev1.Pod{}, tt.accessor, handler.PodDataFormatter)
			result, err := h.Create(context.Background(), tt.request, encodeObject(t, h, tt.request.NewObjectRaw))
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenericHandler_Update(t *testing.T) {
	initialTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	mockClock := mocks.NewMockClock(initialTime)

	tests := []struct {
		name     string
		accessor mockConfigAccessor
		dbresult *types.ResourceTags
		request  *types.AdmissionReview
		expected *types.AdmissionResponse
	}{
		{
			name: "Update with labels and annotations enabled",
			accessor: mockConfigAccessor{
				labelsEnabled:             true,
				annotationsEnabled:        true,
				labelsEnabledForType:      true,
				annotationsEnabledForType: true,
				resourceType:              config.Pod,
				settings: &config.Settings{
					LabelMatches: []regexp.Regexp{
						*regexp.MustCompile("app"),
					},
					AnnotationMatches: []regexp.Regexp{
						*regexp.MustCompile("annotation-key"),
					},
				},
			},
			request: &types.AdmissionReview{
				NewObjectRaw: getRawObject(appsv1.SchemeGroupVersion, &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
						Labels: map[string]string{
							"app": "test",
						},
						Annotations: map[string]string{
							"annotation-key": "annotation-value",
						},
					},
				}),
			},
			expected: &types.AdmissionResponse{Allowed: true},
		},
		{
			name: "Update with labels and annotations enabled with previous data",
			accessor: mockConfigAccessor{
				labelsEnabled:             true,
				annotationsEnabled:        true,
				labelsEnabledForType:      true,
				annotationsEnabledForType: true,
				resourceType:              config.Pod,
				settings: &config.Settings{
					LabelMatches: []regexp.Regexp{
						*regexp.MustCompile("app"),
					},
					AnnotationMatches: []regexp.Regexp{
						*regexp.MustCompile("annotation-key"),
					},
				},
			},
			dbresult: &types.ResourceTags{
				Type:      config.Pod,
				Name:      "test-pod",
				Namespace: stringPtr("default"),
				MetricLabels: &config.MetricLabels{
					"pod":           "test-pod",
					"namespace":     "default",
					"resource_type": "pod",
				},
				Labels: &config.MetricLabelTags{
					"app": "old-value",
				},
				Annotations: &config.MetricLabelTags{
					"annotation-key": "old-annotation-value",
				},
			},
			request: &types.AdmissionReview{
				NewObjectRaw: getRawObject(appsv1.SchemeGroupVersion, &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
						Labels: map[string]string{
							"app": "test",
						},
						Annotations: map[string]string{
							"annotation-key": "annotation-value",
						},
					},
				}),
			},
			expected: &types.AdmissionResponse{Allowed: true},
		},
		{
			name: "Update with labels and annotations disabled",
			accessor: mockConfigAccessor{
				labelsEnabled:             false,
				annotationsEnabled:        false,
				labelsEnabledForType:      false,
				annotationsEnabledForType: false,
				resourceType:              config.Pod,
				settings:                  &config.Settings{},
			},
			request: &types.AdmissionReview{
				NewObjectRaw: getRawObject(appsv1.SchemeGroupVersion, &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
					},
				}),
			},
			expected: &types.AdmissionResponse{Allowed: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtl := gomock.NewController(t)
			defer mockCtl.Finish()

			clock := mockClock
			store := mocks.NewMockResourceStore(mockCtl)

			if tt.accessor.settings != nil && len(tt.accessor.settings.LabelMatches) > 0 {
				store.EXPECT().FindFirstBy(gomock.Any(), gomock.Any()).Return(tt.dbresult, nil)
				store.EXPECT().Tx(gomock.Any(), gomock.Any()).Return(nil)
				if tt.dbresult == nil {
					store.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
				} else {
					store.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
				}
			}

			h := handler.NewGenericHandler[*corev1.Pod](store, tt.accessor.settings, clock, &corev1.Pod{}, tt.accessor, handler.PodDataFormatter)
			result, err := h.Update(context.Background(), tt.request, encodeObject(t, h, tt.request.NewObjectRaw))
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenericHandler_Delete(t *testing.T) {
	initialTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	mockClock := mocks.NewMockClock(initialTime)

	tests := []struct {
		name     string
		accessor mockConfigAccessor
		request  *types.AdmissionReview
		expected *types.AdmissionResponse
	}{
		{
			name: "Delete with labels and annotations enabled",
			accessor: mockConfigAccessor{
				labelsEnabled:             true,
				annotationsEnabled:        true,
				labelsEnabledForType:      true,
				annotationsEnabledForType: true,
				resourceType:              config.Pod,
				settings: &config.Settings{
					LabelMatches: []regexp.Regexp{
						*regexp.MustCompile("app"),
					},
					AnnotationMatches: []regexp.Regexp{
						*regexp.MustCompile("annotation-key"),
					},
				},
			},
			request: &types.AdmissionReview{
				NewObjectRaw: getRawObject(appsv1.SchemeGroupVersion, &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
						Labels: map[string]string{
							"app": "test",
						},
						Annotations: map[string]string{
							"annotation-key": "annotation-value",
						},
					},
				}),
			},
			expected: &types.AdmissionResponse{Allowed: true},
		},
		{
			name: "Delete with labels and annotations disabled",
			accessor: mockConfigAccessor{
				labelsEnabled:             false,
				annotationsEnabled:        false,
				labelsEnabledForType:      false,
				annotationsEnabledForType: false,
				resourceType:              config.Pod,
				settings:                  &config.Settings{},
			},
			request: &types.AdmissionReview{
				NewObjectRaw: getRawObject(appsv1.SchemeGroupVersion, &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-pod",
						Namespace: "default",
					},
				}),
			},
			expected: &types.AdmissionResponse{Allowed: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtl := gomock.NewController(t)
			defer mockCtl.Finish()

			clock := mockClock
			store := mocks.NewMockResourceStore(mockCtl)

			if tt.accessor.settings != nil && len(tt.accessor.settings.LabelMatches) > 0 {
				store.EXPECT().FindFirstBy(gomock.Any(), gomock.Any()).Return(nil, nil)
				store.EXPECT().Tx(gomock.Any(), gomock.Any()).Return(nil)
				store.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
			}

			h := handler.NewGenericHandler[*corev1.Pod](store, tt.accessor.settings, clock, &corev1.Pod{}, tt.accessor, handler.PodDataFormatter)
			result, err := h.Delete(context.Background(), tt.request, encodeObject(t, h, tt.request.NewObjectRaw))
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// //////////////////////////////////////////////////
// TEST SUPPORT HELPERS
type mockConfigAccessor struct {
	labelsEnabled             bool
	annotationsEnabled        bool
	labelsEnabledForType      bool
	annotationsEnabledForType bool
	resourceType              config.ResourceType
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

func (m mockConfigAccessor) ResourceType() config.ResourceType {
	return m.resourceType
}

func (m mockConfigAccessor) Settings() *config.Settings {
	return m.settings
}

func encodeObject(t *testing.T, handler *hook.Handler, rawOjb []byte) metav1.Object {
	// Create a new object from the raw type.
	runtimeObj, err := handler.ObjectCreator.NewObject(rawOjb)
	assert.NoError(t, err)
	assert.NotNil(t, runtimeObj)

	validatingObj, ok := runtimeObj.(metav1.Object)
	assert.True(t, ok)
	assert.NotNil(t, validatingObj)

	return validatingObj
}

func getRawObject(s schema.GroupVersion, o runtime.Object) []byte {
	raw, _ := helper.EncodeRuntimeObject(o)
	return raw
}

func stringPtr(s string) *string {
	return &s
}
