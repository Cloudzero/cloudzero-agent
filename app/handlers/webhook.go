// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package handlers implements HTTP request handlers for the CloudZero agent's APIs.
//
// This package provides the HTTP layer that exposes agent functionality through
// REST endpoints. The handlers serve as the boundary between external HTTP clients
// (Kubernetes API server, Prometheus, monitoring tools) and the internal domain logic.
//
// Key handler types:
//   - ValidationWebhookAPI: Kubernetes admission controller webhook endpoints
//   - Remote write handlers: Prometheus metrics ingestion endpoints
//   - Monitoring handlers: Health checks and metrics exposure
//   - Profiling handlers: Performance debugging endpoints
//
// Architecture:
//   - HTTP routing via go-chi framework
//   - Request validation and parameter parsing
//   - Domain service delegation for business logic
//   - Response marshalling and error handling
//   - Timeout management and graceful degradation
//
// The handlers implement fail-open patterns for critical paths like admission
// control, ensuring that Kubernetes operations continue even if the agent
// experiences temporary issues.
//
// Integration points:
//   - app/domain/webhook: Webhook processing business logic
//   - app/domain: Metric collection and filtering services
//   - Kubernetes API server: Admission review requests
//   - Prometheus: Remote write protocol requests
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-obvious/server"
	"github.com/go-obvious/server/api"
	"github.com/go-obvious/server/request"

	"github.com/rs/zerolog/log"
	admissionv1 "k8s.io/api/admission/v1"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/cloudzero/cloudzero-agent/app/domain/webhook"
	"github.com/cloudzero/cloudzero-agent/app/types"
)

// Webhook admission controller configuration constants and limits.
//
// These values are carefully chosen to balance security, performance, and reliability:
//
// MaxRequestBodyBytes represents the maximum size of Kubernetes objects we read.
// Kubernetes allows a 2x buffer on the max etcd size (3MB default), and we allow
// an additional 2x buffer since 6MB is still reasonable for memory usage.
//
// References:
//   - Kubernetes API server config: https://github.com/kubernetes/kubernetes/blob/0afa569499d480df4977568454a50790891860f5/staging/src/k8s.io/apiserver/pkg/server/config.go#L362
//   - Istio webhook implementation: https://github.com/istio/istio/commit/6ca5055a4db6695ef5504eabdfde3799f2ea91fd
//
// Timeout values are designed to prevent admission controller bottlenecks while
// allowing sufficient time for complex validation operations.
const (
	// minimalAllowResponse is a fail-safe JSON response that always allows admission requests.
	// Used when normal JSON marshalling fails to ensure fail-open behavior that prevents
	// blocking critical Kubernetes operations even during agent errors.
	minimalAllowResponse = `{"response":{"allowed":true}}`
	
	// MaxRequestBodyBytes limits the size of admission review requests to prevent memory exhaustion.
	// Set to 6MB based on Kubernetes and Istio best practices for admission controller safety.
	MaxRequestBodyBytes = int64(6 * 1024 * 1024)
	
	// DefaultTimeout is the maximum time allowed for processing admission requests.
	// Long enough for complex operations but short enough to avoid API server timeouts.
	DefaultTimeout = 15 * time.Second
	
	// MinTimeout is the minimum allowed timeout for admission request processing.
	// Prevents excessively short timeouts that could cause premature failures.
	MinTimeout = 5 * time.Second
)

// Kubernetes API version metadata for admission review responses.
// These type metadata objects ensure responses use the correct API version
// that matches the original admission review request format.
var (
	// v1beta1AdmissionReviewTypeMeta provides metadata for legacy admission review responses.
	// Used when responding to v1beta1 admission review requests from older Kubernetes versions.
	v1beta1AdmissionReviewTypeMeta = metav1.TypeMeta{
		Kind:       "AdmissionReview",
		APIVersion: "admission.k8s.io/v1beta1",
	}

	// v1AdmissionReviewTypeMeta provides metadata for current admission review responses.
	// Used when responding to v1 admission review requests from modern Kubernetes versions.
	v1AdmissionReviewTypeMeta = metav1.TypeMeta{
		Kind:       "AdmissionReview",
		APIVersion: "admission.k8s.io/v1",
	}
)

// ValidationWebhookAPI implements the HTTP API for Kubernetes admission controller webhooks.
// It handles admission review requests from the Kubernetes API server, processes them
// through the webhook controller, and returns admission responses.
//
// The API supports both v1 and v1beta1 admission review formats for compatibility
// across different Kubernetes versions. It implements fail-open behavior to ensure
// that webhook failures don't block critical Kubernetes operations.
//
// Key features:
//   - Multi-version support: Handles both v1 and v1beta1 admission review formats
//   - Fail-open safety: Always allows admission when webhook processing fails
//   - Timeout management: Configurable request timeouts with sensible defaults
//   - Load balancing: Implements connection rotation for HTTP/1.x traffic distribution
//   - Security: Request size limits and content type validation
//   - Monitoring: Comprehensive logging for debugging and audit trails
//
// Request flow:
//   1. Kubernetes API server sends admission review request
//   2. HTTP handler validates content type and size limits
//   3. Request body is decoded to admission review object
//   4. Domain webhook controller processes the admission request
//   5. Response is marshalled and returned to API server
//   6. Connection management for load distribution (HTTP/1.x only)
type ValidationWebhookAPI struct {
	// api.Service provides common HTTP service functionality
	api.Service
	
	// controller handles the business logic for webhook admission processing
	controller webhook.WebhookController
	
	// decoder handles deserialization of Kubernetes admission review objects,
	// supporting both v1 and v1beta1 formats for version compatibility
	decoder runtime.Decoder
}

func NewValidationWebhookAPI(base string, controller webhook.WebhookController) server.API {
	a := &ValidationWebhookAPI{
		controller: controller,
		decoder: func() runtime.Decoder {
			r := runtime.NewScheme()
			r.AddKnownTypes(admissionv1beta1.SchemeGroupVersion, &admissionv1beta1.AdmissionReview{})
			r.AddKnownTypes(admissionv1.SchemeGroupVersion, &admissionv1.AdmissionReview{})
			codecs := serializer.NewCodecFactory(r)
			return codecs.UniversalDeserializer()
		}(),
		Service: api.Service{
			APIName: "webhook",
			Mounts:  map[string]*chi.Mux{},
		},
	}

	a.Service.Mounts[base] = a.Routes()
	return a
}

func (a *ValidationWebhookAPI) Register(app server.Server) error {
	if err := a.Service.Register(app); err != nil {
		return err
	}
	return nil
}

func (a *ValidationWebhookAPI) Routes() *chi.Mux {
	r := chi.NewRouter()
	r.Post("/", a.PostAdmissionRequest)
	return r
}

func (a *ValidationWebhookAPI) requestBodyToModelReview(body []byte) (*types.AdmissionReview, error) {
	review, _, err := a.decoder.Decode(body, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("could not decode the admission review from the request: %w", err)
	}

	switch ar := review.(type) {
	case *admissionv1beta1.AdmissionReview:
		res := types.NewAdmissionReviewV1Beta1(ar)
		return &res, nil
	case *admissionv1.AdmissionReview:
		res := types.NewAdmissionReviewV1(ar)
		return &res, nil
	}

	return nil, errors.New("invalid admission review type")
}

func (a *ValidationWebhookAPI) PostAdmissionRequest(w http.ResponseWriter, r *http.Request) {
	// Webhook execution logic. This is how we are dealing with the different responses:
	// |                        | HTTP Code             | status.Code | status.Status | status.Message |
	// |------------------------|-----------------------| ------------|---------------|----------------|
	// | Validating Allowed     | 200                   | -           | -             | -              |
	// | Validating not allowed | 200                   | 400         | Failure       | Custom message |
	// | Err                    | 500
	ctx := r.Context()
	defer r.Body.Close()

	// Our API should never time more than 15 seconds
	timeout := DefaultTimeout
	if to := request.QS(r, "timeout"); to != "" {
		if duration, err := time.ParseDuration(to); err == nil {
			timeout = duration
		}
		if timeout < MinTimeout {
			timeout = MinTimeout
		}
		if timeout > DefaultTimeout {
			timeout = DefaultTimeout
		}
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	contentType := r.Header.Get("Content-Type")
	if contentType != "application/json" {
		log.Ctx(ctx).Error().Msg("only content type 'application/json' is supported")
		request.Reply(r, w, "failed to read request body", http.StatusBadRequest)
		return
	}

	// Get webhook body with the admission review.
	var body []byte
	body, err := configReader(r)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to parse body")
		request.Reply(r, w, "failed to read request body", http.StatusBadRequest)
		return
	}
	if len(body) == 0 {
		log.Ctx(ctx).Err(err).Msg("no body in request")
		request.Reply(r, w, "no body in request", http.StatusBadRequest)
		return
	}

	review, err := a.requestBodyToModelReview(body)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("could not read request body")
		request.Reply(r, w, fmt.Sprintf("could not read request body: %v", err), http.StatusBadRequest)
		return
	}
	if review == nil {
		log.Ctx(ctx).Error().Msg("malformed admission review: request is nil")
		request.Reply(r, w, "malformed admission review: request is nil", http.StatusBadRequest)
		return
	}

	sendAllowResponse := func(w http.ResponseWriter, r *http.Request) {
		allowResponse := &types.AdmissionResponse{Allowed: true}
		resp, err := a.marshallResponseToJSON(ctx, review, allowResponse)
		if err != nil {
			// Log the error but still allow the request - fail-open behavior
			log.Ctx(ctx).Err(err).Msg("could not marshal allow response to json, allowing request anyway")

			// Use minimal JSON response to ensure we always allow
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(minimalAllowResponse))
			return
		}

		// Only set headers when we know we'll succeed
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(resp)
	}

	log.Ctx(ctx).Trace().
		Int("content_length", int(r.ContentLength)).
		Str("operation", string(review.Operation)).
		Msg("processing review request")

	if _, err := a.controller.Review(ctx, review); err != nil {
		log.Ctx(ctx).Error().Err(err).Send()
		sendAllowResponse(w, r)
		return
	}

	// If we're using HTTP/1.x, we want to periodically close the connection to
	// help distribute the load across the various webhook replicas.
	//
	// Unfortunately this won't work for HTTP/2, but currently all traffic we're
	// seeing from the Kubernetes API server is HTTP/1.1.
	if r.ProtoMajor == 1 {
		rf := a.controller.Settings().Server.ReconnectFrequency
		if rf > 0 && rand.Intn(rf) == 0 { //nolint:gosec // a weak PRNG is fine here
			w.Header().Set("Connection", "close")
		}
	}

	sendAllowResponse(w, r)
}

// configReader is reads an HTTP request, imposing size restrictions aligned with Kubernetes limits.
func configReader(req *http.Request) ([]byte, error) {
	defer req.Body.Close()
	lr := &io.LimitedReader{
		R: req.Body,
		N: MaxRequestBodyBytes + 1,
	}
	data, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if lr.N <= 0 {
		return nil, apierrors.NewRequestEntityTooLargeError(fmt.Sprintf("limit is %d", MaxRequestBodyBytes))
	}
	return data, nil
}

func (a *ValidationWebhookAPI) marshallResponseToJSON(ctx context.Context, review *types.AdmissionReview, resp *types.AdmissionResponse) (data []byte, err error) {
	if resp == nil {
		log.Ctx(ctx).Warn().Msg("admission response is nil")
		return nil, errors.New("invalid admission response")
	}

	// Set the satus code and result based on the validation result.
	var resultStatus *metav1.Status
	if !resp.Allowed {
		resultStatus = &metav1.Status{
			Message: resp.Message,
			Status:  metav1.StatusFailure,
			Code:    http.StatusBadRequest,
		}
	}

	switch review.OriginalAdmissionReview.(type) {
	case *admissionv1beta1.AdmissionReview:
		if len(resp.Warnings) > 0 {
			log.Ctx(ctx).Warn().Msg("warnings used in a 'v1beta1' webhook")
		}

		data, err := json.Marshal(admissionv1beta1.AdmissionReview{
			TypeMeta: v1beta1AdmissionReviewTypeMeta,
			Response: &admissionv1beta1.AdmissionResponse{
				UID:     k8stypes.UID(review.ID),
				Allowed: resp.Allowed,
				Result:  resultStatus,
			},
		})
		return data, err

	case *admissionv1.AdmissionReview:
		data, err := json.Marshal(admissionv1.AdmissionReview{
			TypeMeta: v1AdmissionReviewTypeMeta,
			Response: &admissionv1.AdmissionResponse{
				UID:      k8stypes.UID(review.ID),
				Warnings: resp.Warnings,
				Allowed:  resp.Allowed,
				Result:   resultStatus,
			},
		})
		return data, err
	}

	log.Ctx(ctx).Warn().
		Str("type", fmt.Sprintf("%T", review.OriginalAdmissionReview)).
		Msg("unsupported admission review type")
	return nil, errors.New("invalid admission response type")
}
