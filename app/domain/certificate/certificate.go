// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package certificate provides certificate generation and management functionality
// for the CloudZero Agent, including TLS certificate creation and Kubernetes
// webhook configuration management.
package certificate

//go:generate mockgen -destination=mocks/kubernetes_client_mock.go -package=mocks . KubernetesClient

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"time"
)

const (
	// ECDSA key size constants
	ECDSAKeySize256 = 256
	ECDSAKeySize384 = 384
	ECDSAKeySize521 = 521
)

// CertificateData represents the generated certificate information
type CertificateData struct {
	CABundle string `json:"caBundle"`
	TLSCrt   string `json:"tlsCrt"`
	TLSKey   string `json:"tlsKey"`
}

// WebhookPatch represents a patch operation for webhook configuration
type WebhookPatch struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value string `json:"value"`
}

// CertificateService implements the core certificate management logic
type CertificateService struct {
	k8sClient KubernetesClient
}

// NewCertificateService creates a new certificate service
func NewCertificateService(k8sClient KubernetesClient) *CertificateService {
	return &CertificateService{
		k8sClient: k8sClient,
	}
}

// GenerateCertificate generates a new certificate with the specified parameters
func (s *CertificateService) GenerateCertificate(ctx context.Context, serviceName, namespace string, keySize int, validityDuration time.Duration, algorithm string) (*CertificateData, error) {
	// Validate inputs
	if serviceName == "" {
		return nil, errors.New("service name cannot be empty")
	}
	if namespace == "" {
		return nil, errors.New("namespace cannot be empty")
	}
	if validityDuration <= 0 {
		return nil, errors.New("validity duration must be positive")
	}

	// Generate private key based on algorithm
	var privateKey crypto.PrivateKey
	var publicKey crypto.PublicKey
	var err error

	switch algorithm {
	case "RSA":
		if keySize < 2048 {
			return nil, errors.New("RSA key size must be at least 2048 bits")
		}
		privateKey, err = rsa.GenerateKey(rand.Reader, keySize)
		if err != nil {
			return nil, fmt.Errorf("failed to generate RSA key: %w", err)
		}
		rsaKey, ok := privateKey.(*rsa.PrivateKey)
		if !ok {
			return nil, errors.New("failed to cast private key to RSA key")
		}
		publicKey = rsaKey.Public()
	case "ECDSA":
		if keySize < 256 {
			return nil, errors.New("ECDSA key size must be at least 256 bits")
		}
		var curve elliptic.Curve
		switch keySize {
		case ECDSAKeySize256:
			curve = elliptic.P256()
		case ECDSAKeySize384:
			curve = elliptic.P384()
		case ECDSAKeySize521:
			curve = elliptic.P521()
		default:
			return nil, fmt.Errorf("unsupported ECDSA key size: %d (use %d, %d, or %d)", keySize, ECDSAKeySize256, ECDSAKeySize384, ECDSAKeySize521)
		}
		privateKey, err = ecdsa.GenerateKey(curve, rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("failed to generate ECDSA key: %w", err)
		}
		ecdsaKey, ok := privateKey.(*ecdsa.PrivateKey)
		if !ok {
			return nil, errors.New("failed to cast private key to ECDSA key")
		}
		publicKey = ecdsaKey.Public()
	case "Ed25519":
		if keySize != 0 {
			return nil, errors.New("Ed25519 key size is fixed at 256 bits, cannot be specified")
		}
		publicKey, privateKey, err = ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, fmt.Errorf("failed to generate Ed25519 key: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported algorithm: %s (use RSA, ECDSA, or Ed25519)", algorithm)
	}

	// Create certificate template
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	now := time.Now()
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   serviceName,
			Organization: []string{"CloudZero"},
		},
		NotBefore:             now,
		NotAfter:              now.Add(validityDuration),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{serviceName, serviceName + "." + namespace, serviceName + "." + namespace + ".svc"},
	}

	// Create CA certificate
	caTemplate := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   serviceName + "-ca",
			Organization: []string{"CloudZero"},
		},
		NotBefore:             now,
		NotAfter:              now.Add(validityDuration),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	// Create CA certificate
	caBytes, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, publicKey, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create CA certificate: %w", err)
	}

	// Create server certificate
	certBytes, err := x509.CreateCertificate(rand.Reader, template, caTemplate, publicKey, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create server certificate: %w", err)
	}

	// Encode certificates and key to PEM format
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caBytes})
	if caPEM == nil {
		return nil, errors.New("failed to encode CA certificate to PEM")
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certBytes})
	if certPEM == nil {
		return nil, errors.New("failed to encode server certificate to PEM")
	}

	var keyPEM []byte
	switch k := privateKey.(type) {
	case *rsa.PrivateKey:
		keyPEM = pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)})
		if keyPEM == nil {
			return nil, errors.New("failed to encode RSA private key to PEM")
		}
	case *ecdsa.PrivateKey:
		keyBytes, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal ECDSA private key: %w", err)
		}
		keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})
		if keyPEM == nil {
			return nil, errors.New("failed to encode ECDSA private key to PEM")
		}
	case ed25519.PrivateKey:
		keyPEM = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: k})
		if keyPEM == nil {
			return nil, errors.New("failed to encode Ed25519 private key to PEM")
		}
	default:
		return nil, errors.New("unsupported private key type")
	}

	// Base64 encode for Kubernetes secret storage
	caBundle := base64.StdEncoding.EncodeToString(caPEM)
	tlsCrt := base64.StdEncoding.EncodeToString(certPEM)
	tlsKey := base64.StdEncoding.EncodeToString(keyPEM)

	return &CertificateData{
		CABundle: caBundle,
		TLSCrt:   tlsCrt,
		TLSKey:   tlsKey,
	}, nil
}

// UpdateResources updates the TLS secret and webhook configuration with new certificates
func (s *CertificateService) UpdateResources(ctx context.Context, namespace, secretName, webhookName string, certData *CertificateData) error {
	// Update TLS secret
	patchData := map[string]interface{}{
		"data": map[string]interface{}{
			"ca.crt":  certData.CABundle,
			"tls.crt": certData.TLSCrt,
			"tls.key": certData.TLSKey,
		},
	}

	err := s.k8sClient.PatchSecret(ctx, namespace, secretName, patchData)
	if err != nil {
		return fmt.Errorf("failed to patch secret: %w", err)
	}

	// Update webhook configuration
	patches := []WebhookPatch{
		{
			Op:    "replace",
			Path:  "/webhooks/0/clientConfig/caBundle",
			Value: certData.CABundle,
		},
	}

	err = s.k8sClient.PatchWebhookConfiguration(ctx, webhookName, patches)
	if err != nil {
		return fmt.Errorf("failed to patch webhook configuration: %w", err)
	}

	return nil
}

// ValidateExistingCertificate checks if the existing certificate is valid
func (s *CertificateService) ValidateExistingCertificate(ctx context.Context, namespace, secretName string) (bool, error) {
	secret, err := s.k8sClient.GetTLSSecret(ctx, namespace, secretName)
	if err != nil {
		return false, fmt.Errorf("failed to get TLS secret: %w", err)
	}

	// Check if secret has required fields
	data, ok := secret["data"].(map[string]interface{})
	if !ok {
		return false, errors.New("secret data is not a map")
	}

	requiredFields := []string{"ca.crt", "tls.crt", "tls.key"}
	for _, field := range requiredFields {
		if _, exists := data[field]; !exists {
			return false, nil
		}
	}

	return true, nil
}
