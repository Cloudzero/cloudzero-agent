// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package catalog_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/cloudzero/cloudzero-agent/pkg/config"
	"github.com/cloudzero/cloudzero-agent/pkg/diagnostic"
	"github.com/cloudzero/cloudzero-agent/pkg/diagnostic/catalog"
	"github.com/cloudzero/cloudzero-agent/pkg/diagnostic/kms"
	"github.com/cloudzero/cloudzero-agent/pkg/status"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"
)

type mockProvider struct{}

func (m *mockProvider) Check(ctx context.Context, client *http.Client, recorder status.Accessor) error {
	return nil
}

func mockKMSProvider(ctx context.Context, cfg *config.Settings, clientset ...kubernetes.Interface) diagnostic.Provider {
	return &mockProvider{}
}

func TestRegistry_Get(t *testing.T) {
	// Override kms.NewProvider
	originalNewProvider := kms.NewProvider
	kms.NewProvider = mockKMSProvider
	defer func() { kms.NewProvider = originalNewProvider }()

	ctx := context.Background()
	c := &config.Settings{}
	r := catalog.NewCatalog(ctx, c)

	// Test getting providers with existing IDs
	providers := r.Get(config.DiagnosticAPIKey, config.DiagnosticK8sVersion)
	assert.Len(t, providers, 2)

	// Test getting providers with non-existing IDs
	providers = r.Get("non-existing-id")
	assert.Empty(t, providers)

	// Test getting providers with empty IDs
	providers = r.Get()
	assert.Empty(t, providers)
}

func TestRegistry_Has(t *testing.T) {
	// Override kms.NewProvider
	originalNewProvider := kms.NewProvider
	kms.NewProvider = mockKMSProvider
	defer func() { kms.NewProvider = originalNewProvider }()

	ctx := context.Background()
	c := &config.Settings{}
	r := catalog.NewCatalog(ctx, c)

	// Test checking for existing ID
	has := r.Has(config.DiagnosticAPIKey)
	assert.True(t, has)

	// Test checking for non-existing ID
	has = r.Has("non-existing-id")
	assert.False(t, has)
}

func TestRegistry_List(t *testing.T) {
	// Override kms.NewProvider
	originalNewProvider := kms.NewProvider
	kms.NewProvider = mockKMSProvider
	defer func() { kms.NewProvider = originalNewProvider }()

	ctx := context.Background()
	c := &config.Settings{}
	r := catalog.NewCatalog(ctx, c)

	// Test listing providers
	providers := r.List()
	assert.Len(t, providers, 5) // Update the expected length to 6
}
