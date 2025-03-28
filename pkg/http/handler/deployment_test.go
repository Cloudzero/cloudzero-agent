// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package handler

import (
	"context"
	"regexp"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	config "github.com/cloudzero/cloudzero-insights-controller/app/config/insights-controller"
	"github.com/cloudzero/cloudzero-insights-controller/app/types"
	"github.com/cloudzero/cloudzero-insights-controller/app/types/mocks"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/http/hook"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

type TestRecord struct {
	Type         config.ResourceType
	Name         string
	Namespace    *string
	MetricLabels map[string]string
	Labels       map[string]string
	Annotations  map[string]string
}

func makeDeploymentRequest(record TestRecord) *hook.Request {
	deployment := &v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        record.Name,
			Namespace:   *record.Namespace,
			Labels:      record.Labels,
			Annotations: record.Annotations,
		},
	}

	scheme := runtime.NewScheme()
	v1.AddToScheme(scheme)
	codecs := serializer.NewCodecFactory(scheme)
	encoder := codecs.LegacyCodec(v1.SchemeGroupVersion)
	raw, _ := runtime.Encode(encoder, deployment)

	return &hook.Request{
		Object: runtime.RawExtension{
			Raw: raw,
		},
	}
}

func NewTestSettings() *config.Settings {
	filter := config.Filters{Labels: config.Labels{Enabled: true, Patterns: []string{"*"}}, Annotations: config.Annotations{Enabled: true, Patterns: []string{"*"}}}
	filter.Labels.Resources.Deployments = true
	filter.Annotations.Resources.Deployments = true
	filter.Labels.Resources.Namespaces = true
	filter.Annotations.Resources.Namespaces = true
	compiledPatterns := []regexp.Regexp{}
	testPattern, _ := regexp.Compile(".*")
	compiledPatterns = append(compiledPatterns, *testPattern)
	return &config.Settings{RemoteWrite: config.RemoteWrite{MaxBytesPerSend: 10000, SendInterval: 3}, Filters: filter, LabelMatches: compiledPatterns, AnnotationMatches: compiledPatterns}
}

func TestDeploymentHandler_Create(t *testing.T) {
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
			expected: &hook.Result{Allowed: true},
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
			handler := NewDeploymentHandler(writer, tt.settings, mockClock, make(chan error))
			result, err := handler.Create(context.Background(), tt.request)
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
			expected: &hook.Result{Allowed: true},
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
			expected: &hook.Result{Allowed: true},
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
			handler := NewDeploymentHandler(writer, tt.settings, mockClock, make(chan error))
			result, err := handler.Update(context.Background(), tt.request)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
