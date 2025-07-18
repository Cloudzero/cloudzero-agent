// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package integration provides integration tests.
package integration

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"net/http"

	appsv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

type Request struct {
	Method      string
	QueryParams map[string]string
	Body        []byte
	Route       string
}

type BodyParams struct {
	Kind       string
	UID        string
	ObjectName string
}

func GenerateRequest(method, route, url string, req Request) (*http.Request, error) {
	query := "?"
	for k, v := range req.QueryParams {
		query += fmt.Sprintf("%s=%s&", k, v)
	}
	query = query[:len(query)-1]

	bodyBytes, jsonErr := json.Marshal(req.Body)
	if jsonErr != nil {
		return nil, fmt.Errorf("failed to marshal body: %v", jsonErr)
	}

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(bodyBytes); err != nil {
		return nil, fmt.Errorf("failed to compress body: %v", err)
	}
	if err := gz.Close(); err != nil {
		return nil, fmt.Errorf("failed to close gzip writer: %v", err)
	}

	httpReq, err := http.NewRequest(method, url+route+query, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	httpReq.Header.Set("Content-Encoding", "gzip")

	return httpReq, nil
}

func NewAdmissionRequest() []byte {
	admissionRequest := &appsv1.AdmissionRequest{
		UID: types.UID("12345"),
		Kind: metav1.GroupVersionKind{
			Group:   "",
			Version: "v1",
			Kind:    "Pod",
		},
		Resource: metav1.GroupVersionResource{
			Group:    "",
			Version:  "v1",
			Resource: "pods",
		},
		Name:      "example-pod",
		Namespace: "default",
		Operation: appsv1.Create,
		Object: runtime.RawExtension{
			Raw: []byte(`{
                "apiVersion": "v1",
                "kind": "Pod",
                "metadata": {
                    "name": "example-pod",
                    "namespace": "default"
                },
                "spec": {
                    "containers": [{
                        "name": "example-container",
                        "image": "example-image"
                    }]
                }
            }`),
		},
	}

	admissionReview := &appsv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admission.k8s.io/v1",
			Kind:       "AdmissionReview",
		},
		Request: admissionRequest,
	}

	admissionReviewJSON, err := json.Marshal(admissionReview)
	if err != nil {
		fmt.Printf("Error marshaling AdmissionReview: %v\n", err)
		return nil
	}
	return admissionReviewJSON
}
