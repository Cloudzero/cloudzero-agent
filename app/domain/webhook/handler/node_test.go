// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/handler"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/cloudzero/cloudzero-agent/app/types/mocks"
)

func makeNodeRequest(record TestRecord) *types.AdmissionReview {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:        record.Name,
			Labels:      record.Labels,
			Annotations: record.Annotations,
		},
	}

	return &types.AdmissionReview{
		NewObjectRaw: getRawObject(corev1.SchemeGroupVersion, node),
	}
}

func TestFormatNodeData(t *testing.T) {
	tests := []struct {
		name     string
		node     *corev1.Node
		settings *config.Settings
		expected types.ResourceTags
	}{
		{
			name: "Test with labels and annotations enabled",
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node",
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
							Nodes: true,
						},
					},
					Annotations: config.Annotations{
						Enabled: true,
						Resources: config.Resources{
							Nodes: true,
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
				Type: config.Node,
				Name: "test-node",
				MetricLabels: &config.MetricLabels{
					"node":          "test-node",
					"resource_type": "node",
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
			node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-node",
				},
			},
			settings: &config.Settings{
				Filters: config.Filters{
					Labels: config.Labels{
						Enabled: false,
						Resources: config.Resources{
							Nodes: false,
						},
					},
					Annotations: config.Annotations{
						Enabled: false,
						Resources: config.Resources{
							Nodes: false,
						},
					},
				},
			},
			expected: types.ResourceTags{
				Type: config.Node,
				Name: "test-node",
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
			result := handler.FormatNodeData(tt.node, tt.settings)
			if !reflect.DeepEqual(tt.expected.MetricLabels, result.MetricLabels) {
				t.Errorf("Maps are not equal:\nExpected: %v\nGot: %v", tt.expected.MetricLabels, result.MetricLabels)
			}
			if !reflect.DeepEqual(tt.expected.Labels, result.Labels) {
				t.Errorf("Maps are not equal:\nExpected: %v\nGot: %v", tt.expected.Labels, result.Labels)
			}
			if !reflect.DeepEqual(tt.expected.Annotations, result.Annotations) {
				t.Errorf("Maps are not equal:\nExpected: %v\nGot: %v", tt.expected.Annotations, result.Annotations)
			}
			assert.Equal(t, tt.expected.Type, result.Type)
			assert.Equal(t, tt.expected.Name, result.Name)
		})
	}
}

func TestNewNodeHandler(t *testing.T) {
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
						Resources: config.Resources{
							Nodes: true,
						},
					},
					Annotations: config.Annotations{
						Enabled: true,
						Resources: config.Resources{
							Nodes: true,
						},
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
			writer := mocks.NewMockResourceStore(mockCtl)
			mockClock := mocks.NewMockClock(time.Now())
			h := handler.NewNodeHandler(writer, tt.settings, mockClock)
			assert.NotNil(t, h)
			assert.Equal(t, writer, h.Store)
		})
	}
}

func TestNodeHandler_Create(t *testing.T) {
	tests := []struct {
		name     string
		settings *config.Settings
		request  *types.AdmissionReview
		expected *types.AdmissionResponse
	}{
		{
			name: "Test create with labels and annotations enabled",
			settings: &config.Settings{
				Filters: config.Filters{
					Labels: config.Labels{
						Enabled: true,
						Resources: config.Resources{
							Nodes: true,
						},
					},
					Annotations: config.Annotations{
						Enabled: true,
						Resources: config.Resources{
							Nodes: true,
						},
					},
				},
			},
			request: makeNodeRequest(TestRecord{
				Name: "test-node",
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
			name: "Test create with labels and annotations disabled",
			settings: &config.Settings{
				Filters: config.Filters{
					Labels: config.Labels{
						Enabled: false,
						Resources: config.Resources{
							Nodes: false,
						},
					},
					Annotations: config.Annotations{
						Enabled: false,
						Resources: config.Resources{
							Nodes: false,
						},
					},
				},
			},
			request: makeNodeRequest(TestRecord{
				Name: "test-node",
			}),
			expected: &types.AdmissionResponse{Allowed: true},
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

			h := handler.NewNodeHandler(writer, tt.settings, mockClock)
			result, err := h.Create(context.Background(), tt.request, encodeObject(t, h, tt.request.NewObjectRaw))
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNodeHandler_Update(t *testing.T) {
	initialTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	mockClock := mocks.NewMockClock(initialTime)

	tests := []struct {
		name     string
		settings *config.Settings
		request  *types.AdmissionReview
		dbresult *types.ResourceTags
		expected *types.AdmissionResponse
	}{
		{
			name: "Test update with labels and annotations enabled no previous record",
			settings: &config.Settings{
				Filters: config.Filters{
					Labels: config.Labels{
						Enabled: true,
						Resources: config.Resources{
							Nodes: true,
						},
					},
					Annotations: config.Annotations{
						Enabled: true,
						Resources: config.Resources{
							Nodes: true,
						},
					},
				},
			},
			request: makeNodeRequest(TestRecord{
				Name: "test-node",
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
			name: "Test update with labels and annotations enabled with previous record",
			settings: &config.Settings{
				Filters: config.Filters{
					Labels: config.Labels{
						Enabled: true,
						Resources: config.Resources{
							Nodes: true,
						},
					},
					Annotations: config.Annotations{
						Enabled: true,
						Resources: config.Resources{
							Nodes: true,
						},
					},
				},
			},
			request: makeNodeRequest(TestRecord{
				Name: "test-node",
				Labels: map[string]string{
					"app": "test",
				},
				Annotations: map[string]string{
					"annotation-key": "annotation-value",
				},
			}),
			dbresult: &types.ResourceTags{
				ID:            "1",
				Type:          config.Node,
				Name:          "test-node",
				Labels:        &config.MetricLabelTags{"app": "test"},
				Annotations:   &config.MetricLabelTags{"annotation-key": "annotation-value"},
				RecordCreated: mockClock.GetCurrentTime(),
				RecordUpdated: mockClock.GetCurrentTime(),
			},
			expected: &types.AdmissionResponse{Allowed: true},
		},
		{
			name: "Test update with labels and annotations disabled",
			settings: &config.Settings{
				Filters: config.Filters{
					Labels: config.Labels{
						Enabled: false,
						Resources: config.Resources{
							Nodes: false,
						},
					},
					Annotations: config.Annotations{
						Enabled: false,
						Resources: config.Resources{
							Nodes: false,
						},
					},
				},
			},
			request: makeNodeRequest(TestRecord{
				Name: "test-node",
			}),
			expected: &types.AdmissionResponse{Allowed: true},
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

			h := handler.NewNodeHandler(writer, tt.settings, mockClock)
			result, err := h.Update(context.Background(), tt.request, encodeObject(t, h, tt.request.NewObjectRaw))
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
