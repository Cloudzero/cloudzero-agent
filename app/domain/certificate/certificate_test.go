package certificate_test

import (
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"github.com/cloudzero/cloudzero-agent/app/domain/certificate"
	"github.com/cloudzero/cloudzero-agent/app/domain/certificate/mocks"
)

func TestNewCertificateService(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockClient := mocks.NewMockKubernetesClient(ctrl)

	service := certificate.NewCertificateService(mockClient)

	if service == nil {
		t.Fatal("expected service to be created, got nil")
	}

	// Note: k8sClient field is unexported, so we can't test it directly
	// The test verifies the service was created successfully
}

func TestCertificateService_GenerateCertificate(t *testing.T) {
	tests := []struct {
		name             string
		serviceName      string
		namespace        string
		keySize          int
		validityDuration time.Duration
		algorithm        string
		expectError      bool
		errorContains    string
	}{
		{
			name:             "valid RSA certificate",
			serviceName:      "test-service",
			namespace:        "test-namespace",
			keySize:          2048,
			validityDuration: time.Hour * 24 * 365,
			algorithm:        "RSA",
			expectError:      false,
		},
		{
			name:             "valid ECDSA certificate",
			serviceName:      "test-service",
			namespace:        "test-namespace",
			keySize:          256,
			validityDuration: time.Hour * 24 * 30,
			algorithm:        "ECDSA",
			expectError:      false,
		},
		{
			name:             "valid Ed25519 certificate",
			serviceName:      "test-service",
			namespace:        "test-namespace",
			keySize:          0, // Ignored for Ed25519
			validityDuration: time.Hour * 24 * 365,
			algorithm:        "Ed25519",
			expectError:      false,
		},
		{
			name:             "empty service name",
			serviceName:      "",
			namespace:        "test-namespace",
			keySize:          2048,
			validityDuration: time.Hour * 24 * 365,
			algorithm:        "RSA",
			expectError:      true,
			errorContains:    "service name cannot be empty",
		},
		{
			name:             "empty namespace",
			serviceName:      "test-service",
			namespace:        "",
			keySize:          2048,
			validityDuration: time.Hour * 24 * 365,
			algorithm:        "RSA",
			expectError:      true,
			errorContains:    "namespace cannot be empty",
		},
		{
			name:             "zero validity duration",
			serviceName:      "test-service",
			namespace:        "test-namespace",
			keySize:          2048,
			validityDuration: 0,
			algorithm:        "RSA",
			expectError:      true,
			errorContains:    "validity duration must be positive",
		},
		{
			name:             "negative validity duration",
			serviceName:      "test-service",
			namespace:        "test-namespace",
			keySize:          2048,
			validityDuration: -time.Hour,
			algorithm:        "RSA",
			expectError:      true,
			errorContains:    "validity duration must be positive",
		},
		{
			name:             "RSA key size too small",
			serviceName:      "test-service",
			namespace:        "test-namespace",
			keySize:          1024,
			validityDuration: time.Hour * 24 * 365,
			algorithm:        "RSA",
			expectError:      true,
			errorContains:    "RSA key size must be at least 2048 bits",
		},
		{
			name:             "ECDSA invalid key size",
			serviceName:      "test-service",
			namespace:        "test-namespace",
			keySize:          512,
			validityDuration: time.Hour * 24 * 365,
			algorithm:        "ECDSA",
			expectError:      true,
			errorContains:    "unsupported ECDSA key size",
		},
		{
			name:             "Ed25519 with specified key size",
			serviceName:      "test-service",
			namespace:        "test-namespace",
			keySize:          256,
			validityDuration: time.Hour * 24 * 365,
			algorithm:        "Ed25519",
			expectError:      true,
			errorContains:    "Ed25519 key size is fixed at 256 bits, cannot be specified",
		},
		{
			name:             "unsupported algorithm",
			serviceName:      "test-service",
			namespace:        "test-namespace",
			keySize:          2048,
			validityDuration: time.Hour * 24 * 365,
			algorithm:        "DES", // Unsupported
			expectError:      true,
			errorContains:    "unsupported algorithm: DES",
		},
		{
			name:             "ECDSA P384",
			serviceName:      "test-service",
			namespace:        "test-namespace",
			keySize:          384,
			validityDuration: time.Hour * 24 * 365,
			algorithm:        "ECDSA",
			expectError:      false,
		},
		{
			name:             "ECDSA P521",
			serviceName:      "test-service",
			namespace:        "test-namespace",
			keySize:          521,
			validityDuration: time.Hour * 24 * 365,
			algorithm:        "ECDSA",
			expectError:      false,
		},
		{
			name:             "RSA with large key size",
			serviceName:      "test-service",
			namespace:        "test-namespace",
			keySize:          4096,
			validityDuration: time.Hour * 24 * 365,
			algorithm:        "RSA",
			expectError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := mocks.NewMockKubernetesClient(ctrl)
			service := certificate.NewCertificateService(mockClient)

			certData, err := service.GenerateCertificate(
				context.Background(),
				tt.serviceName,
				tt.namespace,
				tt.keySize,
				tt.validityDuration,
				tt.algorithm,
			)

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

			if certData == nil {
				t.Fatal("expected certData to be returned, got nil")
			}

			// Verify certificate data is properly formatted
			if certData.CABundle == "" {
				t.Error("expected CABundle to be non-empty")
			}
			if certData.TLSCrt == "" {
				t.Error("expected TLSCrt to be non-empty")
			}
			if certData.TLSKey == "" {
				t.Error("expected TLSKey to be non-empty")
			}

			// Verify certificates are valid base64
			caBundleBytes, err := base64.StdEncoding.DecodeString(certData.CABundle)
			if err != nil {
				t.Errorf("CABundle is not valid base64: %v", err)
			}

			tlsCrtBytes, err := base64.StdEncoding.DecodeString(certData.TLSCrt)
			if err != nil {
				t.Errorf("TLSCrt is not valid base64: %v", err)
			}

			tlsKeyBytes, err := base64.StdEncoding.DecodeString(certData.TLSKey)
			if err != nil {
				t.Errorf("TLSKey is not valid base64: %v", err)
			}

			// Verify certificates are valid PEM
			caBlock, _ := pem.Decode(caBundleBytes)
			if caBlock == nil {
				t.Error("CABundle is not valid PEM")
			} else if caBlock.Type != "CERTIFICATE" {
				t.Errorf("expected CA certificate PEM type 'CERTIFICATE', got '%s'", caBlock.Type)
			}

			certBlock, _ := pem.Decode(tlsCrtBytes)
			if certBlock == nil {
				t.Error("TLSCrt is not valid PEM")
			} else if certBlock.Type != "CERTIFICATE" {
				t.Errorf("expected TLS certificate PEM type 'CERTIFICATE', got '%s'", certBlock.Type)
			}

			keyBlock, _ := pem.Decode(tlsKeyBytes)
			if keyBlock == nil {
				t.Error("TLSKey is not valid PEM")
			}

			// Verify certificate can be parsed
			if caBlock != nil {
				_, err := x509.ParseCertificate(caBlock.Bytes)
				if err != nil {
					t.Errorf("failed to parse CA certificate: %v", err)
				}
			}

			if certBlock != nil {
				cert, err := x509.ParseCertificate(certBlock.Bytes)
				if err != nil {
					t.Errorf("failed to parse TLS certificate: %v", err)
				} else {
					// Verify certificate has expected SAN entries
					expectedDNSNames := []string{
						tt.serviceName,
						fmt.Sprintf("%s.%s", tt.serviceName, tt.namespace),
						fmt.Sprintf("%s.%s.svc", tt.serviceName, tt.namespace),
					}

					for _, expectedDNS := range expectedDNSNames {
						found := false
						for _, dnsName := range cert.DNSNames {
							if dnsName == expectedDNS {
								found = true
								break
							}
						}
						if !found {
							t.Errorf("expected DNS name '%s' not found in certificate", expectedDNS)
						}
					}
				}
			}
		})
	}
}

func TestCertificateService_UpdateResources(t *testing.T) {
	tests := []struct {
		name                  string
		namespace             string
		secretName            string
		webhookName           string
		certData              *certificate.CertificateData
		mockSecretPatchError  error
		mockWebhookPatchError error
		expectError           bool
		errorContains         string
	}{
		{
			name:        "successful update",
			namespace:   "test-namespace",
			secretName:  "test-secret",
			webhookName: "test-webhook",
			certData: &certificate.CertificateData{
				CABundle: "Y2EtYnVuZGxl", // base64 "ca-bundle"
				TLSCrt:   "dGxzLWNydA==", // base64 "tls-crt"
				TLSKey:   "dGxzLWtleQ==", // base64 "tls-key"
			},
			expectError: false,
		},
		{
			name:        "secret patch error",
			namespace:   "test-namespace",
			secretName:  "test-secret",
			webhookName: "test-webhook",
			certData: &certificate.CertificateData{
				CABundle: "Y2EtYnVuZGxl",
				TLSCrt:   "dGxzLWNydA==",
				TLSKey:   "dGxzLWtleQ==",
			},
			mockSecretPatchError: fmt.Errorf("secret patch failed"),
			expectError:          true,
			errorContains:        "failed to patch secret",
		},
		{
			name:        "webhook patch error",
			namespace:   "test-namespace",
			secretName:  "test-secret",
			webhookName: "test-webhook",
			certData: &certificate.CertificateData{
				CABundle: "Y2EtYnVuZGxl",
				TLSCrt:   "dGxzLWNydA==",
				TLSKey:   "dGxzLWtleQ==",
			},
			mockWebhookPatchError: fmt.Errorf("webhook patch failed"),
			expectError:           true,
			errorContains:         "failed to patch webhook configuration",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := mocks.NewMockKubernetesClient(ctrl)
			service := certificate.NewCertificateService(mockClient)

			// Set up expectations
			expectedPatchData := map[string]interface{}{
				"data": map[string]interface{}{
					"ca.crt":  tt.certData.CABundle,
					"tls.crt": tt.certData.TLSCrt,
					"tls.key": tt.certData.TLSKey,
				},
			}

			expectedPatches := []certificate.WebhookPatch{
				{
					Op:    "replace",
					Path:  "/webhooks/0/clientConfig/caBundle",
					Value: tt.certData.CABundle,
				},
			}

			mockClient.EXPECT().
				PatchSecret(gomock.Any(), tt.namespace, tt.secretName, expectedPatchData).
				Return(tt.mockSecretPatchError)

			if tt.mockSecretPatchError == nil {
				mockClient.EXPECT().
					PatchWebhookConfiguration(gomock.Any(), tt.webhookName, expectedPatches).
					Return(tt.mockWebhookPatchError)
			}

			err := service.UpdateResources(
				context.Background(),
				tt.namespace,
				tt.secretName,
				tt.webhookName,
				tt.certData,
			)

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
			}
		})
	}
}

func TestCertificateService_ValidateExistingCertificate(t *testing.T) {
	tests := []struct {
		name          string
		namespace     string
		secretName    string
		mockSecret    map[string]interface{}
		mockError     error
		expectValid   bool
		expectError   bool
		errorContains string
	}{
		{
			name:       "valid certificate",
			namespace:  "test-namespace",
			secretName: "test-secret",
			mockSecret: map[string]interface{}{
				"data": map[string]interface{}{
					"ca.crt":  "Y2EtY3J0",
					"tls.crt": "dGxzLWNydA==",
					"tls.key": "dGxzLWtleQ==",
				},
			},
			expectValid: true,
			expectError: false,
		},
		{
			name:       "missing ca.crt",
			namespace:  "test-namespace",
			secretName: "test-secret",
			mockSecret: map[string]interface{}{
				"data": map[string]interface{}{
					"tls.crt": "dGxzLWNydA==",
					"tls.key": "dGxzLWtleQ==",
				},
			},
			expectValid: false,
			expectError: false,
		},
		{
			name:       "missing tls.crt",
			namespace:  "test-namespace",
			secretName: "test-secret",
			mockSecret: map[string]interface{}{
				"data": map[string]interface{}{
					"ca.crt":  "Y2EtY3J0",
					"tls.key": "dGxzLWtleQ==",
				},
			},
			expectValid: false,
			expectError: false,
		},
		{
			name:       "missing tls.key",
			namespace:  "test-namespace",
			secretName: "test-secret",
			mockSecret: map[string]interface{}{
				"data": map[string]interface{}{
					"ca.crt":  "Y2EtY3J0",
					"tls.crt": "dGxzLWNydA==",
				},
			},
			expectValid: false,
			expectError: false,
		},
		{
			name:       "no data field",
			namespace:  "test-namespace",
			secretName: "test-secret",
			mockSecret: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "test-secret",
				},
			},
			expectValid:   false,
			expectError:   true,
			errorContains: "secret data is not a map",
		},
		{
			name:          "get secret error",
			namespace:     "test-namespace",
			secretName:    "test-secret",
			mockError:     fmt.Errorf("secret not found"),
			expectValid:   false,
			expectError:   true,
			errorContains: "failed to get TLS secret",
		},
		{
			name:       "data field not a map",
			namespace:  "test-namespace",
			secretName: "test-secret",
			mockSecret: map[string]interface{}{
				"data": "not-a-map",
			},
			expectValid:   false,
			expectError:   true,
			errorContains: "secret data is not a map",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := mocks.NewMockKubernetesClient(ctrl)
			service := certificate.NewCertificateService(mockClient)

			mockClient.EXPECT().
				GetTLSSecret(gomock.Any(), tt.namespace, tt.secretName).
				Return(tt.mockSecret, tt.mockError)

			isValid, err := service.ValidateExistingCertificate(
				context.Background(),
				tt.namespace,
				tt.secretName,
			)

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

			if isValid != tt.expectValid {
				t.Errorf("expected isValid=%v, got %v", tt.expectValid, isValid)
			}
		})
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
