// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package http implements an admission webhook handler.
package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/rs/zerolog/log"
	admission "k8s.io/api/admission/v1"
	meta "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"github.com/cloudzero/cloudzero-agent/app/http/hook"
)

// admissionHandler represents the HTTP handler for an admission webhook
type admissionHandler struct {
	decoder runtime.Decoder
}

// handler returns an instance of AdmissionHandler
func handler() *admissionHandler {
	return &admissionHandler{
		decoder: serializer.NewCodecFactory(runtime.NewScheme()).UniversalDeserializer(),
	}
}

// Serve returns a http.HandlerFunc for an admission webhook
func (h *admissionHandler) Serve(handler hook.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		log.Ctx(r.Context()).Debug().Msg("Handling admissions request ...")

		if r.Method != http.MethodPost {
			http.Error(w, "invalid method only POST requests are allowed", http.StatusMethodNotAllowed)
			return
		}

		if contentType := r.Header.Get("Content-Type"); contentType != "application/json" {
			http.Error(w, "only content type 'application/json' is supported", http.StatusBadRequest)
			return
		}

		log.Ctx(r.Context()).Debug().Msg("Parsing the request body ...")
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("could not read request body: %v", err), http.StatusBadRequest)
			return
		}

		var review admission.AdmissionReview
		if _, _, err = h.decoder.Decode(body, nil, &review); err != nil {
			http.Error(w, fmt.Sprintf("could not deserialize request: %v", err), http.StatusBadRequest)
			return
		}

		if review.Request == nil {
			http.Error(w, "malformed admission review: request is nil", http.StatusBadRequest)
			return
		}

		log.Ctx(r.Context()).Debug().Str("operation", string(review.Request.Operation)).Msg("Executing the review request ...")
		result, err := handler.Execute(r.Context(), review.Request)
		if err != nil {
			log.Ctx(r.Context()).Error().Err(err).Send()
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		admissionResponse := admission.AdmissionReview{
			TypeMeta: meta.TypeMeta{
				Kind:       "AdmissionReview",
				APIVersion: "admission.k8s.io/v1",
			},
			Response: &admission.AdmissionResponse{
				UID:     review.Request.UID,
				Allowed: result.Allowed,
				Result:  &meta.Status{Message: result.Msg},
			},
		}

		res, err := json.Marshal(admissionResponse)
		if err != nil {
			log.Ctx(r.Context()).Error().Err(err).Msg("failed to marshal")
			http.Error(w, fmt.Sprintf("could not marshal response: %v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(res) // ignore return values

		log.Ctx(r.Context()).
			Debug().
			Str("path", r.URL.Path).
			Str("operation", string(review.Request.Operation)).
			Bool("allowed", result.Allowed).
			Msg("Webhook Handled")
	}
}
