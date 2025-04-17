// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package webhook_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	admissionv1 "k8s.io/api/admission/v1"

	config "github.com/cloudzero/cloudzero-agent/app/config/validator"
	"github.com/cloudzero/cloudzero-agent/app/diagnostic/webhook"
)

func TestNewProvider(t *testing.T) {
	ctx := context.Background()
	cfg := &config.Settings{}

	provider := webhook.NewProvider(ctx, cfg)
	assert.NotNil(t, provider)
}

func TestSendPodToValidatingWebhook_Success(t *testing.T) {
	ctx := context.Background()

	// Mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Mock response
		response := admissionv1.AdmissionReview{
			Response: &admissionv1.AdmissionResponse{
				UID:     "test-uid-12345",
				Allowed: true,
			},
		}
		respBytes, err := json.Marshal(response)
		assert.NoError(t, err)

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(respBytes)
	}))
	defer server.Close()

	response, err := webhook.SendPodToValidatingWebhook(ctx, server.URL)
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.True(t, response.Allowed)
}

func TestSendPodToValidatingWebhook_Failure(t *testing.T) {
	ctx := context.Background()
	// Mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("invalid request"))
	}))
	defer server.Close()

	response, err := webhook.SendPodToValidatingWebhook(ctx, server.URL)
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "webhook returned 400")
}

func TestSendPodToValidatingWebhook_InvalidResponse(t *testing.T) {
	ctx := context.Background()
	// Mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	response, err := webhook.SendPodToValidatingWebhook(ctx, server.URL)
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "could not unmarshal response AdmissionReview")
}
