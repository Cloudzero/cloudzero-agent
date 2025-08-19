// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package certificate provides certificate generation and management functionality
// for the CloudZero Agent, including TLS certificate creation and Kubernetes
// webhook configuration management.
package certificate

import "context"

// KubernetesClient defines the interface for Kubernetes operations
type KubernetesClient interface {
	// GetTLSSecret retrieves a TLS secret from the specified namespace
	GetTLSSecret(ctx context.Context, namespace, secretName string) (map[string]interface{}, error)
	// GetWebhookCABundle retrieves the CA bundle from a webhook configuration
	GetWebhookCABundle(ctx context.Context, webhookName string) (string, error)
	// PatchSecret applies a patch to a secret in the specified namespace
	PatchSecret(ctx context.Context, namespace, secretName string, patchData map[string]interface{}) error
	// PatchWebhookConfiguration applies patches to a webhook configuration
	PatchWebhookConfiguration(ctx context.Context, webhookName string, patches []WebhookPatch) error
}
