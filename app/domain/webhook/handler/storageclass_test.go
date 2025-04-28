// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package handler_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/handler"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/cloudzero/cloudzero-agent/app/types/mocks"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/assert"
)

func TestStorageClassConfigAccessor(t *testing.T) {
	settings := &config.Settings{}
	accessor := handler.NewStorageClassConfigAccessor(settings)

	t.Run("LabelsEnabled", func(t *testing.T) {
		assert.False(t, accessor.LabelsEnabled())
	})

	t.Run("AnnotationsEnabled", func(t *testing.T) {
		assert.False(t, accessor.AnnotationsEnabled())
	})

	t.Run("LabelsEnabledForType", func(t *testing.T) {
		assert.True(t, accessor.LabelsEnabledForType())
	})

	t.Run("AnnotationsEnabledForType", func(t *testing.T) {
		assert.True(t, accessor.AnnotationsEnabledForType())
	})

	t.Run("ResourceType", func(t *testing.T) {
		assert.Equal(t, config.StorageClass, accessor.ResourceType())
	})

	t.Run("Settings", func(t *testing.T) {
		assert.Equal(t, settings, accessor.Settings())
	})
}

func TestStorageClassHandler_PrometheusMetrics(t *testing.T) {
	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()

	store := mocks.NewMockResourceStore(mockCtl)
	settings := &config.Settings{}
	clock := mocks.NewMockClock(time.Now())
	objTemplate := &storagev1.StorageClass{}

	h := handler.NewStorageClassHandler(store, settings, clock, objTemplate)

	t.Run("Create StorageClass - Valid Object", func(t *testing.T) {
		storageClass := &storagev1.StorageClass{
			TypeMeta: metav1.TypeMeta{
				Kind:       "StorageClass",
				APIVersion: "storage.k8s.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-storage-class",
			},
			Provisioner: "alb",
		}

		admissionReview := &types.AdmissionReview{
			NewObjectRaw: getRawObject(storagev1.SchemeGroupVersion, storageClass),
		}

		response, err := h.Create(context.Background(), admissionReview, storageClass)

		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.True(t, response.Allowed)
	})

	t.Run("Create StorageClass - Valid Object (v1beta1)", func(t *testing.T) {
		storageClass := &storagev1beta1.StorageClass{
			TypeMeta: metav1.TypeMeta{
				Kind:       "StorageClass",
				APIVersion: "storage.k8s.io/v1beta1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-storage-class-v1beta1",
			},
			Provisioner: "kubernetes.io/gce-pd",
		}

		admissionReview := &types.AdmissionReview{
			NewObjectRaw: getRawObject(storagev1beta1.SchemeGroupVersion, storageClass),
		}

		response, err := h.Create(context.Background(), admissionReview, storageClass)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.True(t, response.Allowed)
	})

	t.Run("Create StorageClass - Invalid Object (v1beta1)", func(t *testing.T) {
		invalidObject := &corev1.Pod{}

		admissionReview := &types.AdmissionReview{
			NewObjectRaw: getRawObject(storagev1beta1.SchemeGroupVersion, invalidObject),
		}

		response, err := h.Create(context.Background(), admissionReview, invalidObject)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.True(t, response.Allowed)
	})

	t.Run("Create StorageClass - Invalid Object", func(t *testing.T) {
		invalidObject := &corev1.Pod{}

		admissionReview := &types.AdmissionReview{
			NewObjectRaw: getRawObject(storagev1.SchemeGroupVersion, invalidObject),
		}

		response, err := h.Create(context.Background(), admissionReview, invalidObject)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.True(t, response.Allowed)
	})
}

func TestBaseStorageClassHandler_Create(t *testing.T) {
	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()

	store := mocks.NewMockResourceStore(mockCtl)
	settings := &config.Settings{}
	clock := mocks.NewMockClock(time.Now())
	objTemplate := &storagev1.StorageClass{}

	h := handler.NewStorageClassHandler(store, settings, clock, objTemplate)

	t.Run("Create StorageClass - Metrics incremented correctly", func(t *testing.T) {
		// Define the StorageClass object
		storageClass := &storagev1.StorageClass{
			TypeMeta: metav1.TypeMeta{
				Kind:       "StorageClass",
				APIVersion: "storage.k8s.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-storage-class",
			},
			Provisioner: "alb",
		}

		admissionReview := &types.AdmissionReview{
			NewObjectRaw: getRawObject(storagev1.SchemeGroupVersion, storageClass),
		}

		// Reset metrics for testing
		handler.StorageInfoTotal.Reset()

		// Call the Create method on the handler
		response, err := h.Create(context.Background(), admissionReview, storageClass)
		assert.NoError(t, err)
		assert.NotNil(t, response)
		assert.True(t, response.Allowed)

		// Set up the Prometheus HTTP handler to serve the metrics
		metricHandler := promhttp.Handler()

		// Create a test HTTP server to expose the /metrics endpoint
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			metricHandler.ServeHTTP(w, r)
		}))
		defer ts.Close()

		// Make an HTTP GET request to the /metrics endpoint
		resp, err := http.Get(ts.URL + "/metrics")
		assert.NoError(t, err)
		defer resp.Body.Close()

		// Read the response body
		body, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)

		// Debugging: Print the response body to see the raw metrics output
		fmt.Println(string(body))

		// Assert that the metric "storage_info_total" with the appropriate labels is present
		assert.Contains(t, string(body), "czo_storage_type_totals{name=\"test-storage-class\",provisioner=\"alb\"} 1")
	})
}
