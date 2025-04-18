// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package webhook contains code for checking a CloudZero API token.
package webhook

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	net "net/http"

	"github.com/sirupsen/logrus"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/types"

	config "github.com/cloudzero/cloudzero-agent/app/config/validator"
	"github.com/cloudzero/cloudzero-agent/app/domain/diagnostic"
	logging "github.com/cloudzero/cloudzero-agent/app/logging/validator"
	"github.com/cloudzero/cloudzero-agent/app/types/status"
)

const DiagnosticInsightsIngress = config.DiagnosticInsightsIngress

type checker struct {
	cfg    *config.Settings
	logger *logrus.Entry
}

func NewProvider(ctx context.Context, cfg *config.Settings) diagnostic.Provider {
	return &checker{
		cfg: cfg,
		logger: logging.NewLogger().
			WithContext(ctx).WithField(logging.OpField, "cz"),
	}
}

func (c *checker) Check(ctx context.Context, client *net.Client, accessor status.Accessor) error {
	// Hit an authenticated API to verify the API token
	url := fmt.Sprintf("https://%s.%s.svc.cluster.local/validate/pod", c.cfg.Services.InsightsService, c.cfg.Services.Namespace)
	if _, err := SendPodToValidatingWebhook(ctx, url); err != nil {
		accessor.AddCheck(&status.StatusCheck{Name: DiagnosticInsightsIngress, Passing: false, Error: err.Error()})
		return nil
	}
	accessor.AddCheck(&status.StatusCheck{Name: DiagnosticInsightsIngress, Passing: true})
	return nil
}

// SendPodToValidatingWebhook encodes the given Pod into an AdmissionReview and submits it
// to webhookURL. It returns the AdmissionResponse for inspection in your tests.
func SendPodToValidatingWebhook(ctx context.Context, webhookURL string) (*admissionv1.AdmissionResponse, error) {
	// build the AdmissionReview request
	review := &admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{Kind: "AdmissionReview"},
		Request: &admissionv1.AdmissionRequest{
			UID:       types.UID("test-uid-12345"),
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
	var body bytes.Buffer
	if err := encoder.Encode(review, &body); err != nil {
		return nil, fmt.Errorf("could not encode AdmissionReview: %w", err)
	}

	// send it to the webhook
	req, err := net.NewRequestWithContext(ctx, net.MethodPost, webhookURL, &body)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// disable TLS verification - we are trying to see if we can get to the API
	// not validate the cert chain.
	// #nosec G402: InsecureSkipVerify is set to true intentionally for testing purposes
	client := &net.Client{Transport: &net.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("webhook POST failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != net.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("webhook returned %d: %s", resp.StatusCode, string(b))
	}

	// parse the response AdmissionReview
	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read webhook response: %w", err)
	}

	var reviewResponse admissionv1.AdmissionReview
	if err := json.Unmarshal(respData, &reviewResponse); err != nil {
		return nil, fmt.Errorf("could not unmarshal response AdmissionReview: %w", err)
	}

	if reviewResponse.Response == nil {
		return nil, errors.New("no AdmissionResponse in webhook reply")
	}

	return reviewResponse.Response, nil
}
