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

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/handler"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/cloudzero/cloudzero-agent/app/types/mocks"
)

func TestIngressClassHandler_Metrics(t *testing.T) {
	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()

	store := mocks.NewMockResourceStore(mockCtl)
	settings := &config.Settings{}
	clock := mocks.NewMockClock(time.Now())
	objTemplate := &networkingv1.IngressClass{}

	h := handler.NewIngressClassHandler(store, settings, clock, objTemplate)

	t.Run("Metrics incremented correctly with labels (v1)", func(t *testing.T) {
		// Define the IngressClass object for v1
		ingressObj := &networkingv1.IngressClass{
			TypeMeta: metav1.TypeMeta{
				Kind:       "IngressClass",
				APIVersion: "networking.k8s.io/v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-ingress",
			},
			Spec: networkingv1.IngressClassSpec{
				Controller: "cirrus",
				Parameters: &networkingv1.IngressClassParametersReference{
					APIGroup: stringPtr("example.com"),
					Kind:     "IngressParameters",
					Name:     "example-parameters",
				},
			},
		}

		admissionReview := &types.AdmissionReview{
			NewObjectRaw: getRawObject(networkingv1.SchemeGroupVersion, ingressObj),
		}

		// Reset metrics for testing
		handler.IngressTypesTotal.Reset()

		// Call the Create method on the handler
		runtimeObj, err := h.ObjectCreator.NewObject(admissionReview.NewObjectRaw)
		assert.NoError(t, err)
		assert.NotNil(t, runtimeObj)

		validatingObj, ok := runtimeObj.(metav1.Object)
		assert.True(t, ok)
		assert.NotNil(t, validatingObj)

		_, err = h.Create(context.Background(), admissionReview, validatingObj)
		assert.NoError(t, err)

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

		// Assert that the metric "ingress_types_total" with the appropriate labels is present
		assert.Contains(t, string(body), "czo_ingress_types_total{controller=\"cirrus\",name=\"test-ingress\"} 1")
	})

	t.Run("Metrics incremented correctly with labels (v1beta1)", func(t *testing.T) {
		// Define the IngressClass object for v1beta1
		ingressObj := &networkingv1beta1.IngressClass{
			TypeMeta: metav1.TypeMeta{
				Kind:       "IngressClass",
				APIVersion: "networking.k8s.io/v1beta1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-ingress-v1beta1",
			},
			Spec: networkingv1beta1.IngressClassSpec{
				Controller: "cirrus",
				Parameters: &networkingv1beta1.IngressClassParametersReference{
					APIGroup: stringPtr("example.com"),
					Kind:     "IngressParameters",
					Name:     "example-parameters",
				},
			},
		}

		admissionReview := &types.AdmissionReview{
			NewObjectRaw: getRawObject(networkingv1beta1.SchemeGroupVersion, ingressObj),
		}

		// Reset metrics for testing
		handler.IngressTypesTotal.Reset()

		// Call the Create method on the handler
		runtimeObj, err := h.ObjectCreator.NewObject(admissionReview.NewObjectRaw)
		assert.NoError(t, err)
		assert.NotNil(t, runtimeObj)

		validatingObj, ok := runtimeObj.(metav1.Object)
		assert.True(t, ok)
		assert.NotNil(t, validatingObj)

		_, err = h.Create(context.Background(), admissionReview, validatingObj)
		assert.NoError(t, err)

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

		// Assert that the metric "ingress_types_total" with the appropriate labels is present
		assert.Contains(t, string(body), "czo_ingress_types_total{controller=\"cirrus\",name=\"test-ingress-v1beta1\"} 1")
	})
}

func TestNewIngressClassConfigAccessor(t *testing.T) {
	t.Run("Returns a valid IngressClassConfigAccessor instance", func(t *testing.T) {
		settings := &config.Settings{}
		accessor := handler.NewIngressClassConfigAccessor(settings)

		assert.NotNil(t, accessor)
		assert.IsType(t, &handler.IngressClassConfigAccessor{}, accessor)

		ingressAccessor, ok := accessor.(*handler.IngressClassConfigAccessor)
		assert.True(t, ok)
		assert.Equal(t, settings, ingressAccessor.Settings())
	})

	t.Run("Accessor methods return expected values", func(t *testing.T) {
		settings := &config.Settings{}
		accessor := handler.NewIngressClassConfigAccessor(settings)

		assert.False(t, accessor.LabelsEnabled())
		assert.False(t, accessor.AnnotationsEnabled())
		assert.True(t, accessor.LabelsEnabledForType())
		assert.True(t, accessor.AnnotationsEnabledForType())
		assert.Equal(t, config.IngressClass, accessor.ResourceType())
		assert.Equal(t, settings, accessor.Settings())
	})
}
