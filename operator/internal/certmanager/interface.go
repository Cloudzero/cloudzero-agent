// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package certmanager provides the interface and adapter for TLS certificate management
// used by the CloudZeroAgent reconciler.
package certmanager

//go:generate go run go.uber.org/mock/mockgen@v0.6.0 -destination=mocks/cert_manager_mock.go -package=mocks -source=interface.go

import (
	"context"
	"time"

	"github.com/cloudzero/cloudzero-agent/app/domain/certificate"
)

// CertManager abstracts certificate operations for use by the reconciler.
// This interface allows the reconciler to be tested independently of the real
// Kubernetes API and certificate generation logic.
type CertManager interface {
	// ValidateExisting checks whether the TLS secret contains the required
	// ca.crt, tls.crt, and tls.key fields (field presence only, not expiry).
	ValidateExisting(ctx context.Context, namespace, secretName string) (bool, error)

	// ValidateExpiry checks whether the certificate in the TLS secret is valid
	// and will not expire within the given threshold duration.
	// Returns true if the certificate is present, parseable, and not near expiry.
	ValidateExpiry(ctx context.Context, namespace, secretName string, threshold time.Duration) (bool, error)

	// Generate creates a new self-signed CA and server certificate.
	Generate(ctx context.Context, serviceName, namespace string, keySize int, validity time.Duration, algorithm string) (*certificate.CertificateData, error)

	// UpdateResources patches the TLS Secret and ValidatingWebhookConfiguration
	// with the provided certificate data.
	UpdateResources(ctx context.Context, namespace, secretName, webhookName string, certData *certificate.CertificateData) error
}
