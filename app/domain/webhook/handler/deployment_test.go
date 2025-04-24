// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package handler_test

import (
	"context"
	"regexp"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/handler"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/cloudzero/cloudzero-agent/app/types/mocks"
	"github.com/stretchr/testify/assert"
)

type TestRecord struct {
	Type         config.ResourceType
	Name         string
	Namespace    *string
	MetricLabels map[string]string
	Labels       map[string]string
	Annotations  map[string]string
}

func makeDeploymentRequest(record TestRecord) *types.AdmissionReview {
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        record.Name,
			Namespace:   *record.Namespace,
			Labels:      record.Labels,
			Annotations: record.Annotations,
		},
	}

	return &types.AdmissionReview{
		NewObjectRaw: getRawObject(appsv1.SchemeGroupVersion, deployment),
	}
}

func NewTestSettings() *config.Settings {
	filter := config.Filters{Labels: config.Labels{Enabled: true, Patterns: []string{"*"}}, Annotations: config.Annotations{Enabled: true, Patterns: []string{"*"}}}
	compiledPatterns := []regexp.Regexp{}
	testPattern, _ := regexp.Compile(".*")
	compiledPatterns = append(compiledPatterns, *testPattern)
	return &config.Settings{RemoteWrite: config.RemoteWrite{MaxBytesPerSend: 10000, SendInterval: 3}, Filters: filter, LabelMatches: compiledPatterns, AnnotationMatches: compiledPatterns}
}

func TestDeploymentHandler_Create(t *testing.T) {
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
							Deployments: true,
						},
					},
					Annotations: config.Annotations{
						Enabled: true,
						Resources: config.Resources{
							Deployments: true,
						},
					},
				},
			},
			request: makeDeploymentRequest(TestRecord{
				Name:      "test-deployment",
				Namespace: stringPtr("default"),
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
							Deployments: false,
						},
					},
					Annotations: config.Annotations{
						Enabled: false,
						Resources: config.Resources{
							Deployments: false,
						},
					},
				},
			},
			request: makeDeploymentRequest(TestRecord{
				Name:      "test-deployment",
				Namespace: stringPtr("default"),
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
			h := handler.NewDeploymentHandler(writer, tt.settings, mockClock)

			result, err := h.Create(context.Background(), tt.request, encodeObject(t, h, tt.request.NewObjectRaw))
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDeploymentHandler_Update(t *testing.T) {
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
							Deployments: true,
						},
					},
					Annotations: config.Annotations{
						Enabled: true,
						Resources: config.Resources{
							Deployments: true,
						},
					},
				},
			},
			request: makeDeploymentRequest(TestRecord{
				Name:      "test-deployment",
				Namespace: stringPtr("default"),
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
							Deployments: true,
						},
					},
					Annotations: config.Annotations{
						Enabled: true,
						Resources: config.Resources{
							Deployments: true,
						},
					},
				},
			},
			request: makeDeploymentRequest(TestRecord{
				Name:      "test-deployment",
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
				Type:          config.Deployment,
				Name:          "test-deployment",
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
							Deployments: false,
						},
					},
					Annotations: config.Annotations{
						Enabled: false,
						Resources: config.Resources{
							Deployments: false,
						},
					},
				},
			},
			request: makeDeploymentRequest(TestRecord{
				Name:      "test-deployment",
				Namespace: stringPtr("default"),
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
			h := handler.NewDeploymentHandler(writer, tt.settings, mockClock)

			result, err := h.Update(context.Background(), tt.request, encodeObject(t, h, tt.request.NewObjectRaw))
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
