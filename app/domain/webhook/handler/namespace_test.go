// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package handler_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	config "github.com/cloudzero/cloudzero-agent/app/config/insights-controller"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/handler"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/cloudzero/cloudzero-agent/app/types/mocks"
)

func makeNamespaceRequest(record TestRecord) *types.AdmissionReview {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        record.Name,
			Labels:      record.Labels,
			Annotations: record.Annotations,
		},
	}

	return &types.AdmissionReview{
		NewObjectRaw: getRawObject(corev1.SchemeGroupVersion, namespace),
	}
}

func TestNamespaceHandler_Create(t *testing.T) {
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
					},
					Annotations: config.Annotations{
						Enabled: true,
					},
				},
			},
			request: makeNamespaceRequest(TestRecord{
				Name: "test-namespace",
				Labels: map[string]string{
					"app": "test",
				},
				Annotations: map[string]string{
					"annotation-key": "annotation-value",
				},
			}),
			expected: &types.AdmissionResponse{Allowed: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtl := gomock.NewController(t)
			defer mockCtl.Finish()
			writer := mocks.NewMockResourceStore(mockCtl)

			writer.EXPECT().FindFirstBy(gomock.Any(), gomock.Any()).Return(nil, nil)
			writer.EXPECT().Tx(gomock.Any(), gomock.Any()).Return(nil)
			writer.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)

			mockClock := mocks.NewMockClock(time.Now())
			h := handler.NewNamespaceHandler(writer, tt.settings, mockClock)

			result, err := h.Create(context.Background(), tt.request, encodeObject(t, h, tt.request.NewObjectRaw))
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNamespaceHandler_Update(t *testing.T) {
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
					},
					Annotations: config.Annotations{
						Enabled: true,
					},
				},
			},
			request: makeNamespaceRequest(TestRecord{
				Name: "test-namespace",
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
					},
					Annotations: config.Annotations{
						Enabled: true,
					},
				},
			},
			request: makeNamespaceRequest(TestRecord{
				Name: "test-namespace",
				Labels: map[string]string{
					"app": "test",
				},
				Annotations: map[string]string{
					"annotation-key": "annotation-value",
				},
			}),
			dbresult: &types.ResourceTags{
				ID:            "1",
				Type:          config.Namespace,
				Name:          "test-namespace",
				Labels:        &config.MetricLabelTags{"app": "test"},
				Annotations:   &config.MetricLabelTags{"annotation-key": "annotation-value"},
				RecordCreated: mockClock.GetCurrentTime(),
				RecordUpdated: mockClock.GetCurrentTime(),
			},
			expected: &types.AdmissionResponse{Allowed: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtl := gomock.NewController(t)
			defer mockCtl.Finish()
			writer := mocks.NewMockResourceStore(mockCtl)

			writer.EXPECT().FindFirstBy(gomock.Any(), gomock.Any()).Return(tt.dbresult, nil)
			writer.EXPECT().Tx(gomock.Any(), gomock.Any()).Return(nil)
			if tt.dbresult == nil {
				writer.EXPECT().Create(gomock.Any(), gomock.Any()).Return(nil)
			} else {
				writer.EXPECT().Update(gomock.Any(), gomock.Any()).Return(nil)
			}

			mockClock := mocks.NewMockClock(time.Now())
			h := handler.NewNamespaceHandler(writer, tt.settings, mockClock)

			result, err := h.Update(context.Background(), tt.request, encodeObject(t, h, tt.request.NewObjectRaw))
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}
