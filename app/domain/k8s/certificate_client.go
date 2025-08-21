// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package k8s

//go:generate mockgen -destination=mocks/certificate_client_mock.go -package=mocks . CertificateClient

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/cloudzero/cloudzero-agent/app/domain/certificate"
)

// CertificateClient defines the interface for Kubernetes certificate operations
type CertificateClient interface {
	// GetTLSSecret retrieves a TLS secret from the specified namespace
	GetTLSSecret(ctx context.Context, namespace, secretName string) (map[string]interface{}, error)
	// GetWebhookCABundle retrieves the CA bundle from a webhook configuration
	GetWebhookCABundle(ctx context.Context, webhookName string) (string, error)
	// PatchSecret applies a patch to a secret in the specified namespace
	PatchSecret(ctx context.Context, namespace, secretName string, patchData map[string]interface{}) error
	// PatchWebhookConfiguration applies patches to a webhook configuration
	PatchWebhookConfiguration(ctx context.Context, webhookName string, patches []certificate.WebhookPatch) error
}

// certificateClient implements CertificateClient using the existing k8s client
type certificateClient struct {
	clientset kubernetes.Interface
}

// NewCertificateClient creates a new certificate client using the existing k8s client
func NewCertificateClient() (CertificateClient, error) {
	clientset, err := GetClient()
	if err != nil {
		return nil, fmt.Errorf("failed to get k8s client: %w", err)
	}

	return &certificateClient{
		clientset: clientset,
	}, nil
}

// NewCertificateClientWithConfig creates a new certificate client with provided config and clientset
func NewCertificateClientWithConfig(config *rest.Config, clientset kubernetes.Interface) CertificateClient {
	return &certificateClient{
		clientset: clientset,
	}
}

// GetTLSSecret retrieves a TLS secret from Kubernetes
func (c *certificateClient) GetTLSSecret(ctx context.Context, namespace, secretName string) (map[string]interface{}, error) {
	secret, err := c.clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get secret %s in namespace %s: %w", secretName, namespace, err)
	}

	// Convert to map[string]interface{} for JSON patch operations
	secretBytes, err := json.Marshal(secret)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal secret: %w", err)
	}

	var secretMap map[string]interface{}
	err = json.Unmarshal(secretBytes, &secretMap)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal secret: %w", err)
	}

	return secretMap, nil
}

// GetWebhookCABundle retrieves the CA bundle from a webhook configuration
func (c *certificateClient) GetWebhookCABundle(ctx context.Context, webhookName string) (string, error) {
	webhook, err := c.clientset.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(ctx, webhookName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get webhook configuration %s: %w", webhookName, err)
	}

	if len(webhook.Webhooks) == 0 {
		return "", fmt.Errorf("webhook configuration %s has no webhooks", webhookName)
	}

	caBundle := webhook.Webhooks[0].ClientConfig.CABundle
	if caBundle == nil {
		return "", nil
	}

	return base64.StdEncoding.EncodeToString(caBundle), nil
}

// PatchSecret patches a secret with the provided data
func (c *certificateClient) PatchSecret(ctx context.Context, namespace, secretName string, patchData map[string]interface{}) error {
	patchBytes, err := json.Marshal(patchData)
	if err != nil {
		return fmt.Errorf("failed to marshal patch data: %w", err)
	}

	_, err = c.clientset.CoreV1().Secrets(namespace).Patch(
		ctx,
		secretName,
		types.StrategicMergePatchType,
		patchBytes,
		metav1.PatchOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to patch secret: %w", err)
	}

	return nil
}

// PatchWebhookConfiguration patches a webhook configuration with the provided patches
func (c *certificateClient) PatchWebhookConfiguration(ctx context.Context, webhookName string, patches []certificate.WebhookPatch) error {
	patchBytes, err := json.Marshal(patches)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook patches: %w", err)
	}

	_, err = c.clientset.AdmissionregistrationV1().ValidatingWebhookConfigurations().Patch(
		ctx,
		webhookName,
		types.JSONPatchType,
		patchBytes,
		metav1.PatchOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to patch webhook configuration: %w", err)
	}

	return nil
}
