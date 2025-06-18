// SPDX-License-IdentifierText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package handler_test

import (
	"context"
	"reflect"
	"regexp"
	"testing"
	"time"

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
			accessor:  makeMockAccessor(true, true, true, true, config.Pod, "app", "annotation-key"),
			obj:       makePodObject(makeMeta("test-pod", "default", map[string]string{"app": "test"}, map[string]string{"annotation-key": "annotation-value"})),
			expected:  makeExpectedTags(config.Pod, "test-pod", "default", "workload", map[string]string{"app": "test"}, map[string]string{"annotation-key": "annotation-value"}),
		},
		{
			name:      "WorkloadDataFormatter with labels and annotations disabled",
			formatter: handler.WorkloadDataFormatter,
			accessor:  makeMockAccessor(false, false, false, false, config.Pod, "", ""),
			obj:       makePodObject(makeMeta("test-pod", "default", nil, nil)),
			expected:  makeExpectedTags(config.Pod, "test-pod", "default", "workload", nil, nil),
		},
		{
			name:      "PodDataFormatter with labels and annotations enabled",
			formatter: handler.PodDataFormatter,
			accessor:  makeMockAccessor(true, true, true, true, config.Pod, "app", "annotation-key"),
			obj:       makePodObject(makeMeta("test-pod", "default", map[string]string{"app": "test"}, map[string]string{"annotation-key": "annotation-value"})),
			expected:  makeExpectedTags(config.Pod, "test-pod", "default", "pod", map[string]string{"app": "test"}, map[string]string{"annotation-key": "annotation-value"}),
		},
		{
			name:      "PodDataFormatter with labels and annotations disabled",
			formatter: handler.PodDataFormatter,
			accessor:  makeMockAccessor(false, false, false, false, config.Pod, "", ""),
			obj:       makePodObject(makeMeta("test-pod", "default", nil, nil)),
			expected:  makeExpectedTags(config.Pod, "test-pod", "default", "pod", nil, nil),
		},

		{
			name:      "NamespaceDataFormatter with labels and annotations enabled",
			formatter: handler.NamespaceDataFormatter,
			accessor:  makeMockAccessor(true, true, true, true, config.Namespace, "env", "annotation-key"),
			obj:       makePodObject(makeMeta("test-namespace", "", map[string]string{"env": "prod"}, map[string]string{"annotation-key": "annotation-value"})),
			expected:  makeExpectedTags(config.Namespace, "test-namespace", "", "namespace", map[string]string{"env": "prod"}, map[string]string{"annotation-key": "annotation-value"}),
		},
		{
			name:      "NamespaceDataFormatter with labels and annotations disabled",
			formatter: handler.NamespaceDataFormatter,
			accessor:  makeMockAccessor(false, false, false, false, config.Namespace, "", ""),
			obj:       makePodObject(makeMeta("test-namespace", "", nil, nil)),
			expected:  makeExpectedTags(config.Namespace, "test-namespace", "", "namespace", nil, nil),
		},
		{
			name:      "NodeDataFormatter with labels and annotations enabled",
			formatter: handler.NodeDataFormatter,
			accessor:  makeMockAccessor(true, true, true, true, config.Node, "role", "annotation-key"),
			obj:       makePodObject(makeMeta("test-node", "", map[string]string{"role": "worker"}, map[string]string{"annotation-key": "annotation-value"})),
			expected:  makeExpectedTags(config.Node, "test-node", "", "node", map[string]string{"role": "worker"}, map[string]string{"annotation-key": "annotation-value"}),
		},
		{
			name:      "NodeDataFormatter with labels and annotations disabled",
			formatter: handler.NodeDataFormatter,
			accessor:  makeMockAccessor(false, false, false, false, config.Node, "", ""),
			obj:       makePodObject(makeMeta("test-node", "", nil, nil)),
			expected:  makeExpectedTags(config.Node, "test-node", "", "node", nil, nil),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.formatter(tt.accessor, tt.obj)

			if !reflect.DeepEqual(tt.expected.MetricLabels, tt.expected.MetricLabels) {
				t.Errorf("Maps are not equal:\nExpected: %v\nGot: %v", tt.expected.MetricLabels, tt.expected.MetricLabels)
			}
			if !reflect.DeepEqual(tt.expected.Labels, tt.expected.Labels) {
				t.Errorf("Maps are not equal:\nExpected: %v\nGot: %v", tt.expected.Labels, tt.expected.Labels)
			}
			if !reflect.DeepEqual(tt.expected.Annotations, tt.expected.Annotations) {
				t.Errorf("Maps are not equal:\nExpected: %v\nGot: %v", tt.expected.Annotations, tt.expected.Annotations)
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
			request: makePodObjectRequest(metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "default",
				Labels: map[string]string{
					"app": "test",
				},
				Annotations: map[string]string{
					"annotation-key": "annotation-value",
				},
			}),
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
			request: makePodObjectRequest(metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "default",
			}),
			expected: &types.AdmissionResponse{Allowed: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtl := gomock.NewController(t)
			defer mockCtl.Finish()

			clock := mocks.NewMockClock(time.Now())
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
			request: makePodObjectRequest(metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "default",
				Labels: map[string]string{
					"app": "test",
				},
				Annotations: map[string]string{
					"annotation-key": "annotation-value",
				},
			}),
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
			request: makePodObjectRequest(metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "default",
				Labels: map[string]string{
					"app": "test",
				},
				Annotations: map[string]string{
					"annotation-key": "annotation-value",
				},
			}),
			dbresult: &types.ResourceTags{
				ID:            "1",
				Type:          config.Pod,
				Name:          "test-pod",
				Labels:        &config.MetricLabelTags{"app": "test"},
				Annotations:   &config.MetricLabelTags{"annotation-key": "annotation-value"},
				RecordCreated: mockClock.GetCurrentTime(),
				RecordUpdated: mockClock.GetCurrentTime(),
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
			request: makePodObjectRequest(metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "default",
			}),
			expected: &types.AdmissionResponse{Allowed: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtl := gomock.NewController(t)
			defer mockCtl.Finish()

			clock := mocks.NewMockClock(time.Now())
			store := mocks.NewMockResourceStore(mockCtl)

			if len(tt.accessor.settings.LabelMatches) > 0 {
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
			request: makePodObjectRequest(metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "default",
				Labels: map[string]string{
					"app": "test",
				},
				Annotations: map[string]string{
					"annotation-key": "annotation-value",
				},
			}),
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
			request: makePodObjectRequest(metav1.ObjectMeta{
				Name:      "test-pod",
				Namespace: "default",
			}),
			expected: &types.AdmissionResponse{Allowed: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtl := gomock.NewController(t)
			defer mockCtl.Finish()

			clock := mocks.NewMockClock(time.Now())
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

func makeMockAccessor(labelsEnabled, annotationsEnabled, labelsEnabledForType, annotationsEnabledForType bool, resourceType config.ResourceType, labelMatch, annotationMatch string) mockConfigAccessor {
	return mockConfigAccessor{
		labelsEnabled:             labelsEnabled,
		annotationsEnabled:        annotationsEnabled,
		labelsEnabledForType:      labelsEnabledForType,
		annotationsEnabledForType: annotationsEnabledForType,
		resourceType:              resourceType,
		settings: &config.Settings{
			LabelMatches: []regexp.Regexp{
				*regexp.MustCompile(labelMatch),
			},
			AnnotationMatches: []regexp.Regexp{
				*regexp.MustCompile(annotationMatch),
			},
		},
	}
}

func makeMeta(name, namespace string, labels, annotations map[string]string) metav1.ObjectMeta {
	return metav1.ObjectMeta{
		Name:        name,
		Namespace:   namespace,
		Labels:      labels,
		Annotations: annotations,
	}
}

func makeExpectedTags(resourceType config.ResourceType, name, namespace, resourceKind string, labels, annotations map[string]string) types.ResourceTags {
	return types.ResourceTags{
		Type:         resourceType,
		Name:         name,
		Namespace:    stringPtr(namespace),
		MetricLabels: &config.MetricLabels{"resource_kind": resourceKind},
		Labels: func() *config.MetricLabelTags {
			if labels == nil {
				return nil
			}
			tags := config.MetricLabelTags(labels)
			return &tags
		}(),
		Annotations: func() *config.MetricLabelTags {
			if annotations == nil {
				return nil
			}
			tags := config.MetricLabelTags(annotations)
			return &tags
		}(),
	}
}

func makePodObject(o metav1.ObjectMeta) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        o.Name,
			Namespace:   o.Namespace,
			Labels:      o.Labels,
			Annotations: o.Annotations,
		},
	}
}

func makePodObjectRequest(o metav1.ObjectMeta) *types.AdmissionReview {
	return &types.AdmissionReview{
		NewObjectRaw: getRawObject(appsv1.SchemeGroupVersion, &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:        o.Name,
				Namespace:   o.Namespace,
				Labels:      o.Labels,
				Annotations: o.Annotations,
			},
		}),
	}
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
