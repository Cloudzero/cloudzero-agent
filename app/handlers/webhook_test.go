// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package handlers_test implements unit tests for handlers
package handlers_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/go-obvious/server/test"
	"github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/cloudzero/cloudzero-agent/app/domain/webhook"
	"github.com/cloudzero/cloudzero-agent/app/handlers"
	"github.com/cloudzero/cloudzero-agent/app/types"
)

type MockAdmissionController struct{}

func (a *MockAdmissionController) Review(ctx context.Context, ar *types.AdmissionReview) (*types.AdmissionResponse, error) {
	return &types.AdmissionResponse{Allowed: true, Message: "success"}, nil
}

type MockNilController struct{}

func (a *MockNilController) Review(ctx context.Context, ar *types.AdmissionReview) (*types.AdmissionResponse, error) {
	return nil, nil
}

type MockReviewErrorController struct{}

func (a *MockReviewErrorController) Review(ctx context.Context, ar *types.AdmissionReview) (*types.AdmissionResponse, error) {
	return &types.AdmissionResponse{Allowed: true, Message: "success"}, errors.New("this is an error")
}

type MockNotAllowedController struct{}

func (a *MockNotAllowedController) Review(ctx context.Context, ar *types.AdmissionReview) (*types.AdmissionResponse, error) {
	return &types.AdmissionResponse{Allowed: false, Message: "nope"}, nil
}

func TestServe(t *testing.T) {
	createRequest := func(method, contentType string, validBody bool) *http.Request {
		var buf bytes.Buffer
		if validBody {
			// build the AdmissionReview request
			review := &admissionv1.AdmissionReview{
				TypeMeta: metav1.TypeMeta{Kind: "AdmissionReview"},
				Request: &admissionv1.AdmissionRequest{
					UID:       k8stypes.UID("test-uid-12345"),
					Kind:      metav1.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"},
					Resource:  metav1.GroupVersionResource{Group: "", Version: "v1", Resource: "pods"},
					Operation: admissionv1.Create,
					Object: runtime.RawExtension{
						Object: &corev1.Pod{
							ObjectMeta: metav1.ObjectMeta{
								Name:        "cloudzero-test",
								Namespace:   "default",
								Labels:      map[string]string{"app": "test"},
								Annotations: map[string]string{"app": "test"},
							},
						},
					},
				},
			}

			// set up a codec for JSON
			scheme := runtime.NewScheme()
			_ = admissionv1.AddToScheme(scheme)
			_ = corev1.AddToScheme(scheme)
			codecs := serializer.NewCodecFactory(scheme)
			encoder := codecs.EncoderForVersion(codecs.LegacyCodec(admissionv1.SchemeGroupVersion), admissionv1.SchemeGroupVersion)

			// serialize the review
			err := encoder.Encode(review, &buf)
			assert.NoError(t, err)
		}

		// send it to the webhook
		req, err := http.NewRequestWithContext(context.Background(), method, "http://test", &buf)
		assert.NoError(t, err)
		req.Header.Set("Content-Type", contentType)
		return req
	}

	// Test cases
	tests := []struct {
		name           string
		request        *http.Request
		expectedStatus int
		controller     webhook.WebhookController
	}{
		{
			name:           "Invalid method",
			request:        createRequest(http.MethodGet, "application/json", true),
			expectedStatus: http.StatusMethodNotAllowed,
			controller:     &MockAdmissionController{},
		},
		{
			name:           "Invalid content type",
			request:        createRequest(http.MethodPost, "text/plain", true),
			expectedStatus: http.StatusBadRequest,
			controller:     &MockAdmissionController{},
		},
		{
			name:           "Invalid body",
			request:        createRequest(http.MethodPost, "application/json", false),
			expectedStatus: http.StatusBadRequest,
			controller:     &MockAdmissionController{},
		},
		{
			name:           "Valid request",
			request:        createRequest(http.MethodPost, "application/json", true),
			expectedStatus: http.StatusOK,
			controller:     &MockAdmissionController{},
		},
		{
			name:           "Controller Failure",
			request:        createRequest(http.MethodPost, "application/json", true),
			expectedStatus: http.StatusInternalServerError,
			controller:     &MockNilController{},
		},
		{
			name:           "Not allowed",
			request:        createRequest(http.MethodPost, "application/json", true),
			expectedStatus: http.StatusOK,
			controller:     &MockNotAllowedController{},
		},
		{
			name:           "Error in review",
			request:        createRequest(http.MethodPost, "application/json", true),
			expectedStatus: http.StatusInternalServerError,
			controller:     &MockReviewErrorController{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := handlers.NewValidationWebhookAPI("/", tt.controller).(*handlers.ValidationWebhookAPI)

			resp, err := test.InvokeService(handler.Service, "/", *tt.request)
			assert.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
		})
	}
}
