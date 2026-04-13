// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package certmanager

import (
	"context"
	"time"

	"github.com/cloudzero/cloudzero-agent/app/domain/certificate"
)

// Adapter wraps the root module's CertificateService to implement the CertManager interface.
// This keeps all certificate business logic in the existing domain layer while allowing
// the reconciler to depend only on the CertManager interface.
type Adapter struct {
	svc *certificate.CertificateService
}

// NewAdapter creates a new Adapter wrapping the given CertificateService.
func NewAdapter(svc *certificate.CertificateService) *Adapter {
	return &Adapter{svc: svc}
}

func (a *Adapter) ValidateExisting(ctx context.Context, namespace, secretName string) (bool, error) {
	return a.svc.ValidateExistingCertificate(ctx, namespace, secretName)
}

func (a *Adapter) ValidateExpiry(ctx context.Context, namespace, secretName string, threshold time.Duration) (bool, error) {
	return a.svc.ValidateCertificateExpiry(ctx, namespace, secretName, threshold)
}

func (a *Adapter) Generate(ctx context.Context, serviceName, namespace string, keySize int, validity time.Duration, algorithm string) (*certificate.CertificateData, error) {
	return a.svc.GenerateCertificate(ctx, serviceName, namespace, keySize, validity, algorithm)
}

func (a *Adapter) UpdateResources(ctx context.Context, namespace, secretName, webhookName string, certData *certificate.CertificateData) error {
	return a.svc.UpdateResources(ctx, namespace, secretName, webhookName, certData)
}
