// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package handler

import (
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/hook"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/storage"
)

func TestFormatPodData(t *testing.T) {
	tests := []struct {
		name     string
		pod      *corev1.Pod
		settings *config.Settings
		expected storage.ResourceTags
	}{
		{
			name: "Test with labels and annotations enabled",
			pod: &corev1.Pod{
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
			settings: &config.Settings{
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
				LabelMatches: []regexp.Regexp{
					*regexp.MustCompile("app"),
				},
				AnnotationMatches: []regexp.Regexp{
					*regexp.MustCompile("annotation-key"),
				},
			},
			expected: storage.ResourceTags{
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
			name: "Test with labels and annotations disabled",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "default",
				},
			},
			settings: &config.Settings{
				Filters: config.Filters{
					Labels: config.Labels{
						Enabled: false,
						Resources: config.Resources{
							Pods: false,
						},
					},
					Annotations: config.Annotations{
						Enabled: false,
						Resources: config.Resources{
							Pods: false,
						},
					},
				},
			},
			expected: storage.ResourceTags{
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatPodData(tt.pod, tt.settings)
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
			assert.Equal(t, tt.expected.Namespace, result.Namespace)
		})
	}
}

func TestNewPodHandler(t *testing.T) {
	tests := []struct {
		name     string
		writer   storage.DatabaseWriter
		settings *config.Settings
		errChan  chan<- error
	}{
		{
			name:   "Test with valid settings",
			writer: &mockDatabaseWriter{},
			settings: &config.Settings{
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
			},
			errChan: make(chan error),
		},
		{
			name:     "Test with nil settings",
			writer:   &mockDatabaseWriter{},
			settings: nil,
			errChan:  make(chan error),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewPodHandler(tt.writer, tt.settings, tt.errChan)
			assert.NotNil(t, handler)
			assert.Equal(t, tt.writer, handler.Writer)
			assert.Equal(t, tt.errChan, handler.ErrorChan)
		})
	}
}

func makePodRequest(record TestRecord) *hook.Request {
	namespace := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        record.Name,
			Labels:      record.Labels,
			Annotations: record.Annotations,
		},
	}

	if record.Namespace != nil {
		namespace.Namespace = *record.Namespace
	}

	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)
	codecs := serializer.NewCodecFactory(scheme)
	encoder := codecs.LegacyCodec(corev1.SchemeGroupVersion)
	raw, _ := runtime.Encode(encoder, namespace)

	return &hook.Request{
		Object: runtime.RawExtension{
			Raw: raw,
		},
	}
}

func TestPodHandler_Create(t *testing.T) {
	tests := []struct {
		name     string
		settings *config.Settings
		request  *hook.Request
		expected *hook.Result
	}{
		{
			name: "Test create with labels and annotations enabled",
			settings: &config.Settings{
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
			},
			request: makePodRequest(TestRecord{
				Name:      "test-pod",
				Namespace: stringPtr("default"),
				Labels: map[string]string{
					"app": "test",
				},
				Annotations: map[string]string{
					"annotation-key": "annotation-value",
				},
			}),
			expected: &hook.Result{Allowed: true},
		},
		{
			name: "Test create with labels and annotations disabled",
			settings: &config.Settings{
				Filters: config.Filters{
					Labels: config.Labels{
						Enabled: false,
						Resources: config.Resources{
							Pods: false,
						},
					},
					Annotations: config.Annotations{
						Enabled: false,
						Resources: config.Resources{
							Pods: false,
						},
					},
				},
			},
			request: makePodRequest(TestRecord{
				Name:      "test-pod",
				Namespace: stringPtr("default"),
			}),
			expected: &hook.Result{Allowed: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := &mockDatabaseWriter{}
			handler := NewPodHandler(writer, tt.settings, make(chan error))
			result, err := handler.Create(tt.request)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)

			result, err = handler.Update(tt.request)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)

		})
	}
}

type mockDatabaseWriter struct{}

func (m *mockDatabaseWriter) WriteData(data storage.ResourceTags, isCreate bool) error {
	return nil
}

func (m *mockDatabaseWriter) UpdateSentAtForRecords(data []storage.ResourceTags, ct time.Time) (int64, error) {
	return 0, nil
}

func (m *mockDatabaseWriter) PurgeStaleData(rt time.Duration) error {
	return nil
}

func stringPtr(s string) *string {
	return &s
}
