// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package webhook contains code for checking a CloudZero API token.
package webhook

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/rand"
	net "net/http"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	config "github.com/cloudzero/cloudzero-agent/app/config/validator"
	"github.com/cloudzero/cloudzero-agent/app/domain/diagnostic"
	"github.com/cloudzero/cloudzero-agent/app/domain/webhook/helper"
	logging "github.com/cloudzero/cloudzero-agent/app/logging/validator"
	"github.com/cloudzero/cloudzero-agent/app/types/status"
)

const (
	DiagnosticInsightsIngress = config.DiagnosticInsightsIngress
	MaxConnectionAttempts     = 6
	ConnectionTimeout         = 5 * time.Second
	ValidateURLPathProtocol   = "https"
	ValidateURLPath           = "/validate"
)

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
	url := ValidateURLPathProtocol + "://" + c.cfg.Services.InsightsService + "." + c.cfg.Services.Namespace + ".svc.cluster.local" + ValidateURLPath

	var err error
	for attempt := range MaxConnectionAttempts {
		logrus.WithFields(logrus.Fields{
			"attempt": attempt + 1,
			"url":     url,
		}).Info("Attempt to validate webhook")

		// Create a context with a timeout for the current attempt
		ctxWithTimeout, cancel := context.WithTimeout(ctx, ConnectionTimeout)
		defer cancel()

		// Attempt to send a pod to the validating webhook
		_, err = SendPodToValidatingWebhook(ctxWithTimeout, url)
		if err == nil {
			accessor.AddCheck(&status.StatusCheck{Name: DiagnosticInsightsIngress, Passing: true})
			return nil // Validation succeeded
		}

		// Apply exponential backoff with jitter
		backoff := time.Duration(math.Pow(2, float64(attempt))) * time.Second
		jitter := time.Duration(rand.Int63n(int64(time.Second))) // #nosec G404

		logrus.WithFields(logrus.Fields{
			"attempt": attempt + 1,
			"error":   err.Error(),
			"nextTry": time.Now().Add(backoff + jitter),
		}).Warn("Validation attempt failed, retrying")

		// Total time ~= 70 seconds max plus or minus
		time.Sleep(backoff + jitter)
	}

	err = fmt.Errorf("received non-2xx response from %s after %d retries: %w", url, MaxConnectionAttempts, err)
	logrus.WithError(err).WithField("url", url).Error("unable to contact webhook")
	accessor.AddCheck(&status.StatusCheck{Name: DiagnosticInsightsIngress, Passing: false, Error: err.Error()})
	return nil
}

// SendPodToValidatingWebhook sends a Pod to the validating webhook and returns the AdmissionResponse.
func SendPodToValidatingWebhook(ctx context.Context, webhookURL string) (*admissionv1.AdmissionResponse, error) {
	review, err := buildAdmissionReview()
	if err != nil {
		return nil, fmt.Errorf("failed to build AdmissionReview: %w", err)
	}

	resp, err := sendWebhookRequest(ctx, webhookURL, review)
	if err != nil {
		return nil, fmt.Errorf("webhook request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != net.StatusOK {
		return nil, handleNonOKResponse(resp)
	}

	return parseWebhookResponse(resp)
}

func buildAdmissionReview() (*admissionv1.AdmissionReview, error) {
	return &admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{Kind: "AdmissionReview"},
		Request: &admissionv1.AdmissionRequest{
			UID:       types.UID("test-" + uuid.NewString()),
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
	}, nil
}

func sendWebhookRequest(ctx context.Context, webhookURL string, review *admissionv1.AdmissionReview) (*net.Response, error) {
	body, err := helper.EncodeRuntimeObject(review)
	if err != nil {
		return nil, fmt.Errorf("failed to encode AdmissionReview: %w", err)
	}

	req, err := net.NewRequestWithContext(ctx, net.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &net.Client{Transport: &net.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec G402
	}}
	return client.Do(req)
}

func handleNonOKResponse(resp *net.Response) error {
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("webhook returned %d: %s", resp.StatusCode, string(body))
}

func parseWebhookResponse(resp *net.Response) (*admissionv1.AdmissionResponse, error) {
	reader, err := getResponseReader(resp)
	if err != nil {
		return nil, err
	}

	respData, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read webhook response: %w", err)
	}

	var reviewResponse admissionv1.AdmissionReview
	if err := json.Unmarshal(respData, &reviewResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal AdmissionReview: %w", err)
	}

	if reviewResponse.Response == nil {
		return nil, errors.New("no AdmissionResponse in webhook reply")
	}
	return reviewResponse.Response, nil
}

func getResponseReader(resp *net.Response) (io.Reader, error) {
	contentEncoding := resp.Header.Get("Content-Encoding")
	switch contentEncoding {
	case "base64":
		decodedBody, err := base64.StdEncoding.DecodeString(string(readResponseBody(resp)))
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64 response: %w", err)
		}
		return bytes.NewReader(decodedBody), nil
	case "gzip":
		return gzip.NewReader(resp.Body)
	case "deflate":
		return flate.NewReader(resp.Body), nil
	case "":
		return resp.Body, nil
	default:
		return nil, fmt.Errorf("unsupported Content-Encoding: %s", contentEncoding)
	}
}

func readResponseBody(resp *net.Response) []byte {
	body, _ := io.ReadAll(resp.Body)
	return body
}
