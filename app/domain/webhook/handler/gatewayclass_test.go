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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/handler"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/cloudzero/cloudzero-agent/app/types/mocks"
)

func TestGatewayClassHandler_Metrics(t *testing.T) {
	mockCtl := gomock.NewController(t)
	defer mockCtl.Finish()

	store := mocks.NewMockResourceStore(mockCtl)
	settings := &config.Settings{}
	clock := mocks.NewMockClock(time.Now())
	objTemplate := &gatewayv1.GatewayClass{}

	h := handler.NewGatewayClassHandler(store, settings, clock, objTemplate)

	t.Run("Metrics incremented correctly with labels", func(t *testing.T) {
		// Define the GatewayClass object
		gatewayObj := &gatewayv1.GatewayClass{
			TypeMeta: metav1.TypeMeta{
				Kind:       "GatewayClass",
				APIVersion: "gateway.networking.k8s.io/v1beta1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-gateway",
			},
			Spec: gatewayv1.GatewayClassSpec{
				ControllerName: "example-controller",
			},
		}

		admissionReview := &types.AdmissionReview{
			NewObjectRaw: getRawObject(gatewayv1.SchemeGroupVersion, gatewayObj),
		}

		// Reset metrics for testing
		handler.GatewayClassTotal.Reset()

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

		// Assert that the metric "gateway_types_total" with the appropriate labels is present
		assert.Contains(t, string(body), "czo_gateway_types_total{controller=\"example-controller\",name=\"test-gateway\"} 1")
	})
}

func TestNewGatewayClassConfigAccessor(t *testing.T) {
	t.Run("Returns a valid GatewayClassConfigAccessor instance", func(t *testing.T) {
		settings := &config.Settings{}
		accessor := handler.NewGatewayClassConfigAccessor(settings)

		assert.NotNil(t, accessor)
		assert.IsType(t, &handler.GatewayClassConfigAccessor{}, accessor)

		gatewayAccessor, ok := accessor.(*handler.GatewayClassConfigAccessor)
		assert.True(t, ok)
		assert.Equal(t, settings, gatewayAccessor.Settings())
	})

	t.Run("Accessor methods return expected values", func(t *testing.T) {
		settings := &config.Settings{}
		accessor := handler.NewGatewayClassConfigAccessor(settings)

		assert.False(t, accessor.LabelsEnabled())
		assert.False(t, accessor.AnnotationsEnabled())
		assert.True(t, accessor.LabelsEnabledForType())
		assert.True(t, accessor.AnnotationsEnabledForType())
		assert.Equal(t, config.GatewayClass, accessor.ResourceType())
		assert.Equal(t, settings, accessor.Settings())
	})
}
