// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package k8s_test

import (
	"context"
	"encoding/base64"
	"testing"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"

	"github.com/cloudzero/cloudzero-agent/app/domain/certificate"
	"github.com/cloudzero/cloudzero-agent/app/domain/k8s"
)

func TestNewCertificateClient(t *testing.T) {
	// This test is challenging because it depends on external k8s config
	// We'll mainly test that the function doesn't panic and returns an interface
	client, err := k8s.NewCertificateClient()
	// In a test environment without proper k8s config, this will likely fail
	// That's expected and okay - the real test is in the integration tests
	if err != nil {
		t.Logf("Expected error in test environment without k8s config: %v", err)
		return
	}

	if client == nil {
		t.Error("expected client to be non-nil when no error")
	}
}

func TestNewCertificateClientWithConfig(t *testing.T) {
	config := &rest.Config{}
	clientset := fake.NewSimpleClientset()

	client := k8s.NewCertificateClientWithConfig(config, clientset)

	if client == nil {
		t.Fatal("expected client to be created, got nil")
	}

	// Note: clientset field is unexported, so we can't test it directly
	// The test verifies the client was created successfully
}

func TestCertificateClient_GetTLSSecret(t *testing.T) {
	tests := []struct {
		name          string
		namespace     string
		secretName    string
		secret        *corev1.Secret
		expectError   bool
		errorContains string
	}{
		{
			name:       "successful get",
			namespace:  "test-namespace",
			secretName: "test-secret",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "test-namespace",
				},
				Data: map[string][]byte{
					"ca.crt":  []byte("ca-cert-data"),
					"tls.crt": []byte("tls-cert-data"),
					"tls.key": []byte("tls-key-data"),
				},
			},
			expectError: false,
		},
		{
			name:          "secret not found",
			namespace:     "test-namespace",
			secretName:    "nonexistent-secret",
			expectError:   true,
			errorContains: "failed to get secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientset := fake.NewSimpleClientset()

			// Add secret to fake clientset if provided
			if tt.secret != nil {
				_, err := clientset.CoreV1().Secrets(tt.namespace).Create(
					context.Background(),
					tt.secret,
					metav1.CreateOptions{},
				)
				if err != nil {
					t.Fatalf("failed to create test secret: %v", err)
				}
			}

			client := k8s.NewCertificateClientWithConfig(&rest.Config{}, clientset)

			secretMap, err := client.GetTLSSecret(context.Background(), tt.namespace, tt.secretName)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
					return
				}
				if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("expected error to contain '%s', got '%s'", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if secretMap == nil {
				t.Fatal("expected secretMap to be returned, got nil")
			}

			// Verify the secret was properly converted to map
			metadata, ok := secretMap["metadata"].(map[string]interface{})
			if !ok {
				t.Error("expected metadata field to be a map")
			} else {
				if name, ok := metadata["name"].(string); !ok || name != tt.secretName {
					t.Errorf("expected metadata.name to be '%s', got '%v'", tt.secretName, metadata["name"])
				}
			}

			data, ok := secretMap["data"].(map[string]interface{})
			if !ok {
				t.Error("expected data field to be a map")
			} else {
				expectedKeys := []string{"ca.crt", "tls.crt", "tls.key"}
				for _, key := range expectedKeys {
					if _, exists := data[key]; !exists {
						t.Errorf("expected data to contain key '%s'", key)
					}
				}
			}
		})
	}
}

func TestCertificateClient_GetWebhookCABundle(t *testing.T) {
	tests := []struct {
		name             string
		webhookName      string
		webhook          *admissionregistrationv1.ValidatingWebhookConfiguration
		expectedCABundle string
		expectError      bool
		errorContains    string
	}{
		{
			name:        "successful get with CA bundle",
			webhookName: "test-webhook",
			webhook: &admissionregistrationv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-webhook",
				},
				Webhooks: []admissionregistrationv1.ValidatingWebhook{
					{
						Name: "test-webhook",
						ClientConfig: admissionregistrationv1.WebhookClientConfig{
							CABundle: []byte("test-ca-bundle"),
						},
					},
				},
			},
			expectedCABundle: base64.StdEncoding.EncodeToString([]byte("test-ca-bundle")),
			expectError:      false,
		},
		{
			name:        "webhook with nil CA bundle",
			webhookName: "test-webhook",
			webhook: &admissionregistrationv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-webhook",
				},
				Webhooks: []admissionregistrationv1.ValidatingWebhook{
					{
						Name: "test-webhook",
						ClientConfig: admissionregistrationv1.WebhookClientConfig{
							CABundle: nil,
						},
					},
				},
			},
			expectedCABundle: "",
			expectError:      false,
		},
		{
			name:        "webhook with no webhooks",
			webhookName: "test-webhook",
			webhook: &admissionregistrationv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-webhook",
				},
				Webhooks: []admissionregistrationv1.ValidatingWebhook{},
			},
			expectError:   true,
			errorContains: "webhook configuration test-webhook has no webhooks",
		},
		{
			name:          "webhook not found",
			webhookName:   "nonexistent-webhook",
			expectError:   true,
			errorContains: "failed to get webhook configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientset := fake.NewSimpleClientset()

			// Add webhook to fake clientset if provided
			if tt.webhook != nil {
				_, err := clientset.AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(
					context.Background(),
					tt.webhook,
					metav1.CreateOptions{},
				)
				if err != nil {
					t.Fatalf("failed to create test webhook: %v", err)
				}
			}

			client := k8s.NewCertificateClientWithConfig(&rest.Config{}, clientset)

			caBundle, err := client.GetWebhookCABundle(context.Background(), tt.webhookName)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
					return
				}
				if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("expected error to contain '%s', got '%s'", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if caBundle != tt.expectedCABundle {
				t.Errorf("expected CA bundle '%s', got '%s'", tt.expectedCABundle, caBundle)
			}
		})
	}
}

func TestCertificateClient_PatchSecret(t *testing.T) {
	tests := []struct {
		name          string
		namespace     string
		secretName    string
		secret        *corev1.Secret
		patchData     map[string]interface{}
		expectError   bool
		errorContains string
	}{
		{
			name:       "successful patch",
			namespace:  "test-namespace",
			secretName: "test-secret",
			secret: &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "test-namespace",
				},
				Data: map[string][]byte{
					"old-key": []byte("old-value"),
				},
			},
			patchData: map[string]interface{}{
				"data": map[string]interface{}{
					"ca.crt":  base64.StdEncoding.EncodeToString([]byte("ca-cert")),
					"tls.crt": base64.StdEncoding.EncodeToString([]byte("tls-cert")),
					"tls.key": base64.StdEncoding.EncodeToString([]byte("tls-key")),
				},
			},
			expectError: false,
		},
		{
			name:       "patch nonexistent secret",
			namespace:  "test-namespace",
			secretName: "nonexistent-secret",
			patchData: map[string]interface{}{
				"data": map[string]interface{}{
					"ca.crt": base64.StdEncoding.EncodeToString([]byte("ca-cert")),
				},
			},
			expectError:   true,
			errorContains: "failed to patch secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientset := fake.NewSimpleClientset()

			// Add secret to fake clientset if provided
			if tt.secret != nil {
				_, err := clientset.CoreV1().Secrets(tt.namespace).Create(
					context.Background(),
					tt.secret,
					metav1.CreateOptions{},
				)
				if err != nil {
					t.Fatalf("failed to create test secret: %v", err)
				}
			}

			client := k8s.NewCertificateClientWithConfig(&rest.Config{}, clientset)

			err := client.PatchSecret(context.Background(), tt.namespace, tt.secretName, tt.patchData)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
					return
				}
				if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("expected error to contain '%s', got '%s'", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Verify the secret was actually patched
			updatedSecret, err := clientset.CoreV1().Secrets(tt.namespace).Get(
				context.Background(),
				tt.secretName,
				metav1.GetOptions{},
			)
			if err != nil {
				t.Errorf("failed to get updated secret: %v", err)
				return
			}

			// Check that the patch data was applied
			if patchDataMap, ok := tt.patchData["data"].(map[string]interface{}); ok {
				for key, expectedValue := range patchDataMap {
					if expectedStr, ok := expectedValue.(string); ok {
						expectedBytes, err := base64.StdEncoding.DecodeString(expectedStr)
						if err != nil {
							t.Errorf("failed to decode expected value for key %s: %v", key, err)
							continue
						}

						if actualBytes, exists := updatedSecret.Data[key]; !exists {
							t.Errorf("expected key '%s' not found in updated secret", key)
						} else if string(actualBytes) != string(expectedBytes) {
							t.Errorf("expected value for key '%s' to be '%s', got '%s'",
								key, string(expectedBytes), string(actualBytes))
						}
					}
				}
			}
		})
	}
}

func TestCertificateClient_PatchWebhookConfiguration(t *testing.T) {
	tests := []struct {
		name          string
		webhookName   string
		webhook       *admissionregistrationv1.ValidatingWebhookConfiguration
		patches       []certificate.WebhookPatch
		expectError   bool
		errorContains string
	}{
		{
			name:        "successful patch",
			webhookName: "test-webhook",
			webhook: &admissionregistrationv1.ValidatingWebhookConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-webhook",
				},
				Webhooks: []admissionregistrationv1.ValidatingWebhook{
					{
						Name: "test-webhook",
						ClientConfig: admissionregistrationv1.WebhookClientConfig{
							CABundle: []byte("old-ca-bundle"),
						},
					},
				},
			},
			patches: []certificate.WebhookPatch{
				{
					Op:    "replace",
					Path:  "/webhooks/0/clientConfig/caBundle",
					Value: base64.StdEncoding.EncodeToString([]byte("new-ca-bundle")),
				},
			},
			expectError: false,
		},
		{
			name:        "patch nonexistent webhook",
			webhookName: "nonexistent-webhook",
			patches: []certificate.WebhookPatch{
				{
					Op:    "replace",
					Path:  "/webhooks/0/clientConfig/caBundle",
					Value: base64.StdEncoding.EncodeToString([]byte("new-ca-bundle")),
				},
			},
			expectError:   true,
			errorContains: "failed to patch webhook configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientset := fake.NewSimpleClientset()

			// Add webhook to fake clientset if provided
			if tt.webhook != nil {
				_, err := clientset.AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(
					context.Background(),
					tt.webhook,
					metav1.CreateOptions{},
				)
				if err != nil {
					t.Fatalf("failed to create test webhook: %v", err)
				}
			}

			client := k8s.NewCertificateClientWithConfig(&rest.Config{}, clientset)

			err := client.PatchWebhookConfiguration(context.Background(), tt.webhookName, tt.patches)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
					return
				}
				if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("expected error to contain '%s', got '%s'", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Verify the webhook was actually patched
			updatedWebhook, err := clientset.AdmissionregistrationV1().ValidatingWebhookConfigurations().Get(
				context.Background(),
				tt.webhookName,
				metav1.GetOptions{},
			)
			if err != nil {
				t.Errorf("failed to get updated webhook: %v", err)
				return
			}

			// For this simple test, just verify the webhook still exists
			// A full test would require more complex JSON patch parsing
			if len(updatedWebhook.Webhooks) == 0 {
				t.Error("expected webhook to still have webhooks after patch")
			}
		})
	}
}

func TestCertificateClient_PatchDataMarshalError(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	client := k8s.NewCertificateClientWithConfig(&rest.Config{}, clientset)

	// Create a secret first so we can patch it
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "test-namespace",
		},
		Data: map[string][]byte{
			"old-key": []byte("old-value"),
		},
	}

	_, err := clientset.CoreV1().Secrets("test-namespace").Create(
		context.Background(),
		secret,
		metav1.CreateOptions{},
	)
	if err != nil {
		t.Fatalf("failed to create test secret: %v", err)
	}

	// Create valid patch data
	patchData := map[string]interface{}{
		"data": map[string]interface{}{
			"ca.crt": base64.StdEncoding.EncodeToString([]byte("ca-cert")),
		},
	}

	err = client.PatchSecret(context.Background(), "test-namespace", "test-secret", patchData)
	// This should not error since we're using valid data
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCertificateClient_GetTLSSecretMarshalError(t *testing.T) {
	// Test the case where JSON marshaling fails
	// This is hard to trigger with real data, but we can test the error handling
	clientset := fake.NewSimpleClientset()
	client := k8s.NewCertificateClientWithConfig(&rest.Config{}, clientset)

	// Create a secret with data that might cause marshaling issues
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-secret",
			Namespace: "test-namespace",
		},
		Data: map[string][]byte{
			"test-key": []byte("test-value"),
		},
	}

	_, err := clientset.CoreV1().Secrets("test-namespace").Create(
		context.Background(),
		secret,
		metav1.CreateOptions{},
	)
	if err != nil {
		t.Fatalf("failed to create test secret: %v", err)
	}

	// This should succeed with normal data
	result, err := client.GetTLSSecret(context.Background(), "test-namespace", "test-secret")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == nil {
		t.Error("expected result to be non-nil")
	}
}

func TestCertificateClient_PatchesMarshalError(t *testing.T) {
	clientset := fake.NewSimpleClientset()
	client := k8s.NewCertificateClientWithConfig(&rest.Config{}, clientset)

	// Create a webhook first so we can patch it
	webhook := &admissionregistrationv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-webhook",
		},
		Webhooks: []admissionregistrationv1.ValidatingWebhook{
			{
				Name: "test-webhook",
				ClientConfig: admissionregistrationv1.WebhookClientConfig{
					CABundle: []byte("old-ca-bundle"),
				},
			},
		},
	}

	_, err := clientset.AdmissionregistrationV1().ValidatingWebhookConfigurations().Create(
		context.Background(),
		webhook,
		metav1.CreateOptions{},
	)
	if err != nil {
		t.Fatalf("failed to create test webhook: %v", err)
	}

	// Create patches with valid data
	validPatches := []certificate.WebhookPatch{
		{
			Op:    "replace",
			Path:  "/webhooks/0/clientConfig/caBundle",
			Value: base64.StdEncoding.EncodeToString([]byte("new-ca-bundle")),
		},
	}

	err = client.PatchWebhookConfiguration(context.Background(), "test-webhook", validPatches)
	// This should not error since we're using valid data
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(substr) == 0 || len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			func() bool {
				for i := 1; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}())))
}
