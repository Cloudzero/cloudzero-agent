// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"context"
	"reflect"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	config "github.com/cloudzero/cloudzero-agent-validator/app/config/insights-controller"
	"github.com/cloudzero/cloudzero-agent-validator/app/http/hook"
	"github.com/cloudzero/cloudzero-agent-validator/app/types"
	"github.com/cloudzero/cloudzero-agent-validator/app/types/mocks"
)

func makeJobRequest(record TestRecord) *hook.Request {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        record.Name,
			Labels:      record.Labels,
			Annotations: record.Annotations,
		},
	}

	if record.Namespace != nil {
		job.Namespace = *record.Namespace
	}

	scheme := runtime.NewScheme()
	batchv1.AddToScheme(scheme)
	codecs := serializer.NewCodecFactory(scheme)
	encoder := codecs.LegacyCodec(batchv1.SchemeGroupVersion)
	raw, _ := runtime.Encode(encoder, job)

	return &hook.Request{
		Object: runtime.RawExtension{
			Raw: raw,
		},
	}
}

func TestFormatJobData(t *testing.T) {
	tests := []struct {
		name     string
		job      *batchv1.Job
		settings *config.Settings
		expected types.ResourceTags
	}{
		{
			name: "Test with labels and annotations enabled",
			job: &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
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
							Jobs: true,
						},
					},
					Annotations: config.Annotations{
						Enabled: true,
						Resources: config.Resources{
							Jobs: true,
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
			expected: types.ResourceTags{
				Type:      config.Job,
				Name:      "test-job",
				Namespace: stringPtr("default"),
				MetricLabels: &config.MetricLabels{
					"job":           "test-job",
					"namespace":     "default",
					"resource_type": "job",
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
			job: &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-job",
					Namespace: "default",
				},
			},
			settings: &config.Settings{
				Filters: config.Filters{
					Labels: config.Labels{
						Enabled: false,
						Resources: config.Resources{
							Jobs: false,
						},
					},
					Annotations: config.Annotations{
						Enabled: false,
						Resources: config.Resources{
							Jobs: false,
						},
					},
				},
			},
			expected: types.ResourceTags{
				Type:      config.Job,
				Name:      "test-job",
				Namespace: stringPtr("default"),
				MetricLabels: &config.MetricLabels{
					"job":           "test-job",
					"namespace":     "default",
					"resource_type": "job",
				},
				Labels:      &config.MetricLabelTags{},
				Annotations: &config.MetricLabelTags{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatJobData(tt.job, tt.settings)
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

func TestNewJobHandler(t *testing.T) {
	tests := []struct {
		name     string
		settings *config.Settings
		errChan  chan<- error
	}{
		{
			name: "Test with valid settings",
			settings: &config.Settings{
				Filters: config.Filters{
					Labels: config.Labels{
						Enabled: true,
						Resources: config.Resources{
							Jobs: true,
						},
					},
					Annotations: config.Annotations{
						Enabled: true,
						Resources: config.Resources{
							Jobs: true,
						},
					},
				},
			},
			errChan: make(chan error),
		},
		{
			name:     "Test with nil settings",
			settings: nil,
			errChan:  make(chan error),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtl := gomock.NewController(t)
			defer mockCtl.Finish()
			writer := mocks.NewMockResourceStore(mockCtl)
			mockClock := mocks.NewMockClock(time.Now())
			handler := NewJobHandler(writer, tt.settings, mockClock, tt.errChan)
			assert.NotNil(t, handler)
			assert.Equal(t, writer, handler.Store)
			assert.Equal(t, tt.errChan, handler.ErrorChan)
		})
	}
}

func TestJobHandler_Create(t *testing.T) {
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
							Jobs: true,
						},
					},
					Annotations: config.Annotations{
						Enabled: true,
						Resources: config.Resources{
							Jobs: true,
						},
					},
				},
			},
			request: makeJobRequest(TestRecord{
				Name:      "test-job",
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
							Jobs: false,
						},
					},
					Annotations: config.Annotations{
						Enabled: false,
						Resources: config.Resources{
							Jobs: false,
						},
					},
				},
			},
			request: makeJobRequest(TestRecord{
				Name:      "test-job",
				Namespace: stringPtr("default"),
			}),
			expected: &hook.Result{Allowed: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtl := gomock.NewController(t)
			defer mockCtl.Finish()
			writer := mocks.NewMockResourceStore(mockCtl)

			if tt.settings.Filters.Labels.Enabled {
				writer.EXPECT().FindFirstBy(gomock.Any(), gomock.Any()).Return(nil, nil)
				writer.EXPECT().Tx(gomock.Any(), gomock.Any()).Return(nil)
				writer.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
			}
			mockClock := mocks.NewMockClock(time.Now())
			handler := NewJobHandler(writer, tt.settings, mockClock, make(chan error))
			result, err := handler.Create(context.Background(), tt.request)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestJobHandler_Update(t *testing.T) {
	initialTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	mockClock := mocks.NewMockClock(initialTime)

	tests := []struct {
		name     string
		settings *config.Settings
		request  *hook.Request
		dbresult *types.ResourceTags
		expected *hook.Result
	}{
		{
			name: "Test update with labels and annotations enabled no previous record",
			settings: &config.Settings{
				Filters: config.Filters{
					Labels: config.Labels{
						Enabled: true,
						Resources: config.Resources{
							Jobs: true,
						},
					},
					Annotations: config.Annotations{
						Enabled: true,
						Resources: config.Resources{
							Jobs: true,
						},
					},
				},
			},
			request: makeJobRequest(TestRecord{
				Name:      "test-job",
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
			name: "Test update with labels and annotations enabled with previous record",
			settings: &config.Settings{
				Filters: config.Filters{
					Labels: config.Labels{
						Enabled: true,
						Resources: config.Resources{
							Jobs: true,
						},
					},
					Annotations: config.Annotations{
						Enabled: true,
						Resources: config.Resources{
							Jobs: true,
						},
					},
				},
			},
			request: makeJobRequest(TestRecord{
				Name:      "test-job",
				Namespace: stringPtr("default"),
				Labels: map[string]string{
					"app": "test",
				},
				Annotations: map[string]string{
					"annotation-key": "annotation-value",
				},
			}),
			dbresult: &types.ResourceTags{
				ID:            "1",
				Type:          config.Job,
				Name:          "test-job",
				Labels:        &config.MetricLabelTags{"app": "test"},
				Annotations:   &config.MetricLabelTags{"annotation-key": "annotation-value"},
				RecordCreated: mockClock.GetCurrentTime(),
				RecordUpdated: mockClock.GetCurrentTime(),
			},
			expected: &hook.Result{Allowed: true},
		},
		{
			name: "Test update with labels and annotations disabled",
			settings: &config.Settings{
				Filters: config.Filters{
					Labels: config.Labels{
						Enabled: false,
						Resources: config.Resources{
							Jobs: false,
						},
					},
					Annotations: config.Annotations{
						Enabled: false,
						Resources: config.Resources{
							Jobs: false,
						},
					},
				},
			},
			request: makeJobRequest(TestRecord{
				Name:      "test-job",
				Namespace: stringPtr("default"),
			}),
			expected: &hook.Result{Allowed: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtl := gomock.NewController(t)
			defer mockCtl.Finish()
			writer := mocks.NewMockResourceStore(mockCtl)

			if tt.settings.Filters.Labels.Enabled {
				writer.EXPECT().FindFirstBy(gomock.Any(), gomock.Any()).Return(tt.dbresult, nil)
				writer.EXPECT().Tx(gomock.Any(), gomock.Any()).Return(nil)
				if tt.dbresult == nil {
					writer.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
				} else {
					writer.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
				}
			}
			mockClock := mocks.NewMockClock(time.Now())
			handler := NewJobHandler(writer, tt.settings, mockClock, make(chan error))
			result, err := handler.Update(context.Background(), tt.request)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
