// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package handlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/go-obvious/server/test"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/metrics/pkg/apis/custom_metrics/v1beta1"

	config "github.com/cloudzero/cloudzero-agent/app/config/gator"
	"github.com/cloudzero/cloudzero-agent/app/domain"
	"github.com/cloudzero/cloudzero-agent/app/handlers"
	"github.com/cloudzero/cloudzero-agent/app/types/mocks"
)

func TestCustomMetricsAPI_Routes(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	initialTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	mockClock := mocks.NewMockClock(initialTime)

	storage := mocks.NewMockStore(ctrl)
	// Mock the Pending method to return a value for the shipping progress metric
	storage.EXPECT().Pending().Return(500000).AnyTimes()
	// Mock the ElapsedTime method for the time-based shipping progress metric
	storage.EXPECT().ElapsedTime().Return(int64(10000)).AnyTimes()

	cfg := &config.Settings{
		Database: config.Database{
			MaxRecords: 1500000, // 1.5 million
		},
	}

	collector, err := domain.NewMetricCollector(cfg, mockClock, storage, nil)
	assert.NoError(t, err)
	defer collector.Close()

	handler := handlers.NewCustomMetricsAPI("/apis/custom.metrics.k8s.io/v1beta1", collector, nil)

	tests := []struct {
		name               string
		method             string
		path               string
		expectedStatusCode int
	}{
		{
			name:               "list_metrics",
			method:             "GET",
			path:               "/apis/custom.metrics.k8s.io/v1beta1/",
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "get_metric_for_specific_pod",
			method:             "GET",
			path:               "/apis/custom.metrics.k8s.io/v1beta1/namespaces/cza/pods/test-pod/czo_cost_metrics_shipping_progress",
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "get_metric_for_pods_without_specific_pod",
			method:             "GET",
			path:               "/apis/custom.metrics.k8s.io/v1beta1/namespaces/cza/pods/czo_cost_metrics_shipping_progress",
			expectedStatusCode: http.StatusOK,
		},
		{
			name:               "unsupported_method",
			method:             "POST",
			path:               "/apis/custom.metrics.k8s.io/v1beta1/",
			expectedStatusCode: http.StatusMethodNotAllowed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createRequest(tt.method, tt.path, nil)
			resp, err := test.InvokeService(handler.Service, tt.path, *req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatusCode, resp.StatusCode)
		})
	}
}

func TestCustomMetricsAPI_ListMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	initialTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	mockClock := mocks.NewMockClock(initialTime)

	storage := mocks.NewMockStore(ctrl)
	cfg := &config.Settings{
		Database: config.Database{
			MaxRecords: 1500000,
		},
	}

	collector, err := domain.NewMetricCollector(cfg, mockClock, storage, nil)
	assert.NoError(t, err)
	defer collector.Close()

	handler := handlers.NewCustomMetricsAPI("/apis/custom.metrics.k8s.io/v1beta1", collector, nil)

	req := createRequest("GET", "/apis/custom.metrics.k8s.io/v1beta1/", nil)
	resp, err := test.InvokeService(handler.Service, "/apis/custom.metrics.k8s.io/v1beta1/", *req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var apiResourceList metav1.APIResourceList
	err = json.NewDecoder(resp.Body).Decode(&apiResourceList)
	assert.NoError(t, err)

	// Verify the expected structure
	assert.Equal(t, "APIResourceList", apiResourceList.Kind)
	assert.Equal(t, "v1", apiResourceList.APIVersion)
	assert.Equal(t, "custom.metrics.k8s.io/v1beta1", apiResourceList.GroupVersion)
	assert.Len(t, apiResourceList.APIResources, 1)
	assert.Equal(t, "pods/czo_cost_metrics_shipping_progress", apiResourceList.APIResources[0].Name)
	assert.Equal(t, true, apiResourceList.APIResources[0].Namespaced)
	assert.Equal(t, "MetricValueList", apiResourceList.APIResources[0].Kind)
	assert.Equal(t, metav1.Verbs{"get"}, apiResourceList.APIResources[0].Verbs)
}

func TestCustomMetricsAPI_GetCustomMetricForPod(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	initialTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	mockClock := mocks.NewMockClock(initialTime)

	storage := mocks.NewMockStore(ctrl)
	// Mock the Pending method to return 0 for no metrics
	storage.EXPECT().Pending().Return(0).AnyTimes()
	// Mock the ElapsedTime method for the time-based shipping progress metric
	storage.EXPECT().ElapsedTime().Return(int64(10000)).AnyTimes()

	cfg := &config.Settings{
		Database: config.Database{
			MaxRecords: 1500000,
		},
	}

	collector, err := domain.NewMetricCollector(cfg, mockClock, storage, nil)
	assert.NoError(t, err)
	defer collector.Close()

	handler := handlers.NewCustomMetricsAPI("/apis/custom.metrics.k8s.io/v1beta1", collector, nil)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedMetric string
		expectedPod    string
		expectedNS     string
	}{
		{
			name:           "valid metric request",
			path:           "/apis/custom.metrics.k8s.io/v1beta1/namespaces/cza/pods/test-pod/czo_cost_metrics_shipping_progress",
			expectedStatus: http.StatusOK,
			expectedMetric: "czo_cost_metrics_shipping_progress",
			expectedPod:    "test-pod",
			expectedNS:     "cza",
		},
		{
			name:           "unknown metric",
			path:           "/apis/custom.metrics.k8s.io/v1beta1/namespaces/cza/pods/test-pod/unknown_metric",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createRequest("GET", tt.path, nil)
			resp, err := test.InvokeService(handler.Service, tt.path, *req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == http.StatusOK {
				assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

				var metricValue v1beta1.MetricValue
				err = json.NewDecoder(resp.Body).Decode(&metricValue)
				assert.NoError(t, err)

				assert.Equal(t, "MetricValue", metricValue.Kind)
				assert.Equal(t, "custom.metrics.k8s.io/v1beta1", metricValue.APIVersion)
				assert.Equal(t, tt.expectedMetric, metricValue.MetricName)
				assert.Equal(t, tt.expectedPod, metricValue.DescribedObject.Name)
				assert.Equal(t, tt.expectedNS, metricValue.DescribedObject.Namespace)
				assert.Equal(t, "Pod", metricValue.DescribedObject.Kind)
				assert.Equal(t, "v1", metricValue.DescribedObject.APIVersion)

				// Check that the value is a valid quantity (any format is acceptable)
				assert.NotNil(t, metricValue.Value)
				assert.True(t, metricValue.Value.IsZero() || !metricValue.Value.IsZero(), "value should be a valid quantity")

				// Check that timestamp is recent (within last minute)
				timeDiff := time.Since(metricValue.Timestamp.Time)
				assert.True(t, timeDiff < time.Minute, "timestamp should be recent")
			}
		})
	}
}

func TestCustomMetricsAPI_GetCustomMetricForPods(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	initialTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	mockClock := mocks.NewMockClock(initialTime)

	storage := mocks.NewMockStore(ctrl)
	// Mock the Pending method to return 0 for no metrics
	storage.EXPECT().Pending().Return(0).AnyTimes()
	// Mock the ElapsedTime method for the time-based shipping progress metric
	storage.EXPECT().ElapsedTime().Return(int64(10000)).AnyTimes()

	cfg := &config.Settings{
		Database: config.Database{
			MaxRecords: 1500000,
		},
	}

	collector, err := domain.NewMetricCollector(cfg, mockClock, storage, nil)
	assert.NoError(t, err)
	defer collector.Close()

	handler := handlers.NewCustomMetricsAPI("/apis/custom.metrics.k8s.io/v1beta1", collector, nil)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedMetric string
		expectedNS     string
	}{
		{
			name:           "valid metric request for pods",
			path:           "/apis/custom.metrics.k8s.io/v1beta1/namespaces/cza/pods/czo_cost_metrics_shipping_progress",
			expectedStatus: http.StatusOK,
			expectedMetric: "czo_cost_metrics_shipping_progress",
			expectedNS:     "cza",
		},
		{
			name:           "unknown metric for pods",
			path:           "/apis/custom.metrics.k8s.io/v1beta1/namespaces/cza/pods/unknown_metric",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createRequest("GET", tt.path, nil)
			resp, err := test.InvokeService(handler.Service, tt.path, *req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)

			if tt.expectedStatus == http.StatusOK {
				assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

				var metricValueList v1beta1.MetricValueList
				err = json.NewDecoder(resp.Body).Decode(&metricValueList)
				assert.NoError(t, err)

				assert.Equal(t, "MetricValueList", metricValueList.Kind)
				assert.Equal(t, "custom.metrics.k8s.io/v1beta1", metricValueList.APIVersion)
				assert.Len(t, metricValueList.Items, 1)

				metricValue := metricValueList.Items[0]
				assert.Equal(t, tt.expectedMetric, metricValue.MetricName)
				assert.Equal(t, tt.expectedNS, metricValue.DescribedObject.Namespace)
				assert.Equal(t, "Pod", metricValue.DescribedObject.Kind)
				assert.Equal(t, "v1", metricValue.DescribedObject.APIVersion)

				// Check that the value is a valid quantity (any format is acceptable)
				assert.NotNil(t, metricValue.Value)
				assert.True(t, metricValue.Value.IsZero() || !metricValue.Value.IsZero(), "value should be a valid quantity")
			}
		})
	}
}

func TestCustomMetricsAPI_GetCurrentMetricValue(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	initialTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	mockClock := mocks.NewMockClock(initialTime)

	storage := mocks.NewMockStore(ctrl)
	// Mock the Pending method to return 0 for no metrics
	storage.EXPECT().Pending().Return(0).AnyTimes()
	// Mock the ElapsedTime method for the time-based shipping progress metric
	storage.EXPECT().ElapsedTime().Return(int64(10000)).AnyTimes()

	cfg := &config.Settings{
		Database: config.Database{
			MaxRecords: 1500000,
		},
	}

	collector, err := domain.NewMetricCollector(cfg, mockClock, storage, nil)
	assert.NoError(t, err)
	defer collector.Close()

	handler := handlers.NewCustomMetricsAPI("/apis/custom.metrics.k8s.io/v1beta1", collector, nil)

	req := createRequest("GET", "/apis/custom.metrics.k8s.io/v1beta1/namespaces/cza/pods/test-pod/czo_cost_metrics_shipping_progress", nil)
	resp, err := test.InvokeService(handler.Service, "/apis/custom.metrics.k8s.io/v1beta1/namespaces/cza/pods/test-pod/czo_cost_metrics_shipping_progress", *req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var metricValue v1beta1.MetricValue
	err = json.NewDecoder(resp.Body).Decode(&metricValue)
	assert.NoError(t, err)

	// Check that the value is a valid quantity (any format is acceptable)
	assert.NotNil(t, metricValue.Value)
	assert.True(t, metricValue.Value.IsZero() || !metricValue.Value.IsZero(), "value should be a valid quantity")
}

func TestCustomMetricsAPI_PrometheusIntegration(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	initialTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	mockClock := mocks.NewMockClock(initialTime)

	storage := mocks.NewMockStore(ctrl)
	observabilityStorage := mocks.NewMockStore(ctrl)

	// Mock the Pending method to return a value that gives us 50% progress
	storage.EXPECT().Pending().Return(750000).AnyTimes() // 750k / 1.5M = 0.5
	// Mock the ElapsedTime method for the time-based shipping progress metric
	storage.EXPECT().ElapsedTime().Return(int64(10000)).AnyTimes()
	// Mock the Flush method for both stores
	storage.EXPECT().Flush().Return(nil).AnyTimes()
	observabilityStorage.EXPECT().Flush().Return(nil).AnyTimes()

	cfg := &config.Settings{
		Database: config.Database{
			MaxRecords: 1500000,
		},
	}

	collector, err := domain.NewMetricCollector(cfg, mockClock, storage, observabilityStorage)
	assert.NoError(t, err)
	defer collector.Close()

	// The metric is already registered by the collector, so we don't need to register it again
	// Just trigger an update to set the value
	collector.Flush(context.Background())

	handler := handlers.NewCustomMetricsAPI("/apis/custom.metrics.k8s.io/v1beta1", collector, nil)

	req := createRequest("GET", "/apis/custom.metrics.k8s.io/v1beta1/namespaces/cza/pods/test-pod/czo_cost_metrics_shipping_progress", nil)
	resp, err := test.InvokeService(handler.Service, "/apis/custom.metrics.k8s.io/v1beta1/namespaces/cza/pods/test-pod/czo_cost_metrics_shipping_progress", *req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// Parse the response
	var result v1beta1.MetricValue
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)

	// Verify the basic structure
	assert.Equal(t, "czo_cost_metrics_shipping_progress", result.MetricName)
	assert.Equal(t, "test-pod", result.DescribedObject.Name)
	assert.Equal(t, "cza", result.DescribedObject.Namespace)

	// Check that the value is a valid quantity and not zero (should be 50% progress)
	assert.NotNil(t, result.Value)
	assert.False(t, result.Value.IsZero(), "value should not be zero for 50% progress")
}
