// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	admission "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/build"
)

const (
	defaultPort     = ":8443"
	tlsKeyFilePath  = "/etc/certs/tls.key"
	tlsCertFilePath = "/etc/certs/tls.crt"
	readTimeout     = 15 * time.Second
	writeTimeout    = 15 * time.Second
	idleTimeout     = 60 * time.Second
)

const (
	httpHeaderContentType = "Content-Type"
	httpContentTypeJSON   = "application/json"
)

var (
	// Runtime objects for Kubernetes resource processing.
	runtimeScheme = runtime.NewScheme()
	codecFactory  = serializer.NewCodecFactory(runtimeScheme)
	deserializer  = codecFactory.UniversalDeserializer()
)

// init() methods in golang are executed before the main function.
func init() {
	// Initialize Kubernetes schemes for the core, apps, and admission groups.
	_ = corev1.AddToScheme(runtimeScheme)
	_ = admission.AddToScheme(runtimeScheme)
	_ = appsv1.AddToScheme(runtimeScheme)
}

// admitV1Func is a "type" definition for a function signature
// admitV1Func functions take an AdmissionReview object and return an AdmissionResponse object.
type admitV1Func func(admission.AdmissionReview) *admission.AdmissionResponse

// AdmitHandler encapsulates a function that handles an admission review.
// Designed to allow us to provide multiple versions of the handler over time.
type AdmitHandler struct {
	v1 admitV1Func
}

// NewAdmitHandler creates a new AdmitHandler with the given function.
func NewAdmitHandler(f admitV1Func) AdmitHandler {
	return AdmitHandler{v1: f}
}

// serve handles HTTP requests, processes the body, and sends responses.
// main web API logic handler for the associated route
func serve(w http.ResponseWriter, r *http.Request, admit AdmitHandler) {
	var body []byte
	if r.Body != nil {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			log.Error().Msgf("Failed to read request body: %v", err)
			http.Error(w, fmt.Sprintf("Failed to read request body: %v", err), http.StatusBadRequest)
			return
		}
		body = data
	}

	contentType := r.Header.Get(httpHeaderContentType)
	if contentType != httpContentTypeJSON {
		log.Error().Msgf("Expected content-type application/json, got %s", contentType)
		http.Error(w, "Invalid content-type", http.StatusUnsupportedMediaType)
		return
	}

	fmt.Println(string(body))
	responseObj, err := handleAdmissionReview(body, admit)
	if err != nil {
		log.Error().Msgf("Error handling admission review: %v", err)
		http.Error(w, fmt.Sprintf("Error processing request: %v", err), http.StatusInternalServerError)
		return
	}

	respBytes, err := json.Marshal(responseObj)
	if err != nil {
		log.Error().Msgf("Error marshaling response: %v", err)
		http.Error(w, fmt.Sprintf("Error encoding response: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set(httpHeaderContentType, httpContentTypeJSON)
	if _, err := w.Write(respBytes); err != nil {
		log.Error().Msgf("Error sending response: %v", err)
	}
}

// serveValidate handles HTTP requests for validating resources.
// Can use NewAdmitHandler here to add version-specific handlers.
func serveValidate(w http.ResponseWriter, r *http.Request) {
	serve(w, r, NewAdmitHandler(validate))
}

// validate checks the resource type of the AdmissionReview and validates it.
// Must always returns 'Allowed: true' to allow the request.
func validate(ar admission.AdmissionReview) *admission.AdmissionResponse {
	log.Debug().Msgf("Validating deployments")
	deploymentResource := metav1.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}

	// XXX: we are only handling deployment and pods per our webhooks.yaml manifest
	// We may want to do other types...
	if ar.Request.Resource != deploymentResource {
		return &admission.AdmissionResponse{Result: &metav1.Status{Message: fmt.Sprintf("Expected resource to be %s", deploymentResource)}}
	}

	var deployment appsv1.Deployment
	if _, _, err := deserializer.Decode(ar.Request.Object.Raw, nil, &deployment); err != nil {
		// log the decode error only
		log.Error().Msgf("Failed to decode deployment: %v", err)
		return &admission.AdmissionResponse{Allowed: true} // always allow
	}

	log.Debug().Msgf("Deployment validated: %s", deployment.Name)
	return &admission.AdmissionResponse{Allowed: true} // always allow
}

// handleAdmissionReview processes the request body, calling the admit handler
// note that admit handler is wrapping validate() at this time.
//
// validate accepts the admission review object itself.
// ** But here we could switch on types:
//
//	   eg. obj.GetObjectKind() returns a schema.ObjectKind
//		  switch on the type of object handler accordingly
func handleAdmissionReview(body []byte, admit AdmitHandler) (*admission.AdmissionReview, error) {
	var responseObj *admission.AdmissionReview
	obj, gvk, err := deserializer.Decode(body, nil, nil)
	if err != nil {
		log.Error().Msgf("Request could not be decoded: %v", err)
		return nil, fmt.Errorf("request could not be decoded: %w", err)
	}

	requestedAdmissionReview, ok := obj.(*admission.AdmissionReview)
	if !ok {
		return nil, fmt.Errorf("expected AdmissionReview but got: %T", obj)
	}

	responseAdmissionReview := &admission.AdmissionReview{}
	responseAdmissionReview.SetGroupVersionKind(*gvk)
	responseAdmissionReview.Response = admit.v1(*requestedAdmissionReview)
	responseAdmissionReview.Response.UID = requestedAdmissionReview.Request.UID
	responseObj = responseAdmissionReview
	return responseObj, nil
}

func main() {
	// Parse command line flags while providing defaults
	var tlsKey, tlsCert string
	flag.StringVar(&tlsKey, "tlsKey", tlsKeyFilePath, "Path to the TLS key")
	flag.StringVar(&tlsCert, "tlsCert", tlsCertFilePath, "Path to the TLS certificate")
	flag.Parse()

	log.Info().Msgf("Starting CloudZero Insights Controller %s", build.GetVersion())

	// create a new server instance with a sane configuration
	server := &http.Server{
		Addr:         defaultPort,
		Handler:      http.DefaultServeMux,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}

	// Register the validate handler handler for any request to the /validate path.
	http.HandleFunc("/validate", serveValidate)

	// Start the server with TLS enabled, listening on the configured port.
	// NOTE: returns only on error or graceful exit.
	if err := server.ListenAndServeTLS(tlsCert, tlsKey); err != nil {
		// log a fatal message with sets the exit code to 1 and stops the process
		log.Fatal().Err(err).Msg("Server failed to start")
	}

	// Print a message when the server is stopped.
	log.Info().Msg("Server stopped")
}
