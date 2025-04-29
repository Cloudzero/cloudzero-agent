// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

// MaxRequestBodyBytes represents the max size of Kubernetes objects we read. Kubernetes allows a 2x
// buffer on the max etcd size
// (https://github.com/kubernetes/kubernetes/blob/0afa569499d480df4977568454a50790891860f5/staging/src/k8s.io/apiserver/pkg/server/config.go#L362).
// We allow an additional 2x buffer, as it is still fairly cheap (6mb)
// Taken from https://github.com/istio/istio/commit/6ca5055a4db6695ef5504eabdfde3799f2ea91fd
const (
	MaxRequestBodyBytes = int64(6 * 1024 * 1024)
	DefaultTimeout      = 15 * time.Second
	MinTimeout          = 5 * time.Second
)

var (
	v1beta1AdmissionReviewTypeMeta = metav1.TypeMeta{
		Kind:       "AdmissionReview",
		APIVersion: "admission.k8s.io/v1beta1",
	}

	v1AdmissionReviewTypeMeta = metav1.TypeMeta{
		Kind:       "AdmissionReview",
		APIVersion: "admission.k8s.io/v1",
	}
)

type ValidationWebhookAPI struct {
	api.Service
	controller webhook.WebhookController
	decoder    runtime.Decoder
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

	log.Ctx(ctx).Trace().
		Int("content_length", int(r.ContentLength)).
		Str("operation", string(review.Operation)).
		Msg("processing review request")

	result, err := a.controller.Review(ctx, review)
	if err != nil {
		log.Ctx(ctx).Error().Err(err).Send()
		request.Reply(r, w, fmt.Sprintf("review failure: %v", err), http.StatusInternalServerError)
		return
	}

	resp, err := a.marshallResponseToJSON(ctx, review, result)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("could no map response to json")
		errResp, err2 := a.errorToJSON(review, err)
		if err2 != nil {
			log.Ctx(ctx).Err(err2).Msg("could not marshall status error on admission response")
			request.Reply(r, w, fmt.Sprintf("could not marshall status error on admission response: %v", err), http.StatusInternalServerError)
			return
		}
		request.Reply(r, w, errResp, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(resp)
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

func (a *ValidationWebhookAPI) errorToJSON(review *types.AdmissionReview, err error) ([]byte, error) {
	switch review.OriginalAdmissionReview.(type) {
	case *admissionv1beta1.AdmissionReview:
		r := &admissionv1beta1.AdmissionResponse{
			UID: k8stypes.UID(review.ID),
			Result: &metav1.Status{
				Message: err.Error(),
				Status:  metav1.StatusFailure,
			},
		}

		return json.Marshal(admissionv1beta1.AdmissionReview{
			TypeMeta: v1beta1AdmissionReviewTypeMeta,
			Response: r,
		})
	case *admissionv1.AdmissionReview:
		r := &admissionv1.AdmissionResponse{
			UID: k8stypes.UID(review.ID),
			Result: &metav1.Status{
				Message: err.Error(),
				Status:  metav1.StatusFailure,
			},
		}

		return json.Marshal(admissionv1.AdmissionReview{
			TypeMeta: v1AdmissionReviewTypeMeta,
			Response: r,
		})
	}

	return nil, errors.New("invalid admission response type")
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
