// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package runner

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/cloudzero/cloudzero-agent-validator/pkg/config"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/diagnostic"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/diagnostic/catalog"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/diagnostic/kms"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/status"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/kubernetes"
)

type mockProvider struct {
	Test func(ctx context.Context, client *http.Client, recorder status.Accessor) error
}

func (m *mockProvider) Check(ctx context.Context, client *http.Client, recorder status.Accessor) error {
	if m.Test != nil {
		return m.Test(ctx, client, recorder)
	}
	return nil
}

func NewMockKMSProvider(ctx context.Context, cfg *config.Settings, clientset ...kubernetes.Interface) diagnostic.Provider {
	return &mockProvider{
		Test: func(ctx context.Context, client *http.Client, recorder status.Accessor) error {
			// Simulate a successful check
			return nil
		},
	}
}

func TestRunner_Run_Error(t *testing.T) {
	cfg := &config.Settings{
		Deployment: config.Deployment{
			AccountID:   "test-account",
			Region:      "test-region",
			ClusterName: "test-cluster",
		},
	}

	// Use the mock provider for KMS
	originalNewProvider := kms.NewProvider
	kms.NewProvider = NewMockKMSProvider
	defer func() { kms.NewProvider = originalNewProvider }()

	reg := catalog.NewCatalog(context.Background(), cfg)
	stage := config.ContextStageInit

	r := NewRunner(cfg, reg, stage)
	engine := r.(*runner)

	// Add mock providers
	mockProvider1 := &mockProvider{}
	mockProvider2 := &mockProvider{}
	engine.AddPreStep(mockProvider1)
	engine.AddStep(mockProvider2)
	engine.AddPostStep(mockProvider1)

	// Simulate an error in one of the providers
	mockProvider2.Test = func(ctx context.Context, client *http.Client, recorder status.Accessor) error {
		return errors.New("provider error")
	}

	recorder, err := r.Run(context.Background())

	assert.Error(t, err)
	assert.NotNil(t, recorder)
}

func TestRunner_Run(t *testing.T) {
	cfg := &config.Settings{
		Deployment: config.Deployment{
			AccountID:   "test-account",
			Region:      "test-region",
			ClusterName: "test-cluster",
		},
	}

	// Use the mock provider for KMS
	originalNewProvider := kms.NewProvider
	kms.NewProvider = NewMockKMSProvider
	defer func() { kms.NewProvider = originalNewProvider }()

	reg := catalog.NewCatalog(context.Background(), cfg)
	stage := config.ContextStageInit

	r := NewRunner(cfg, reg, stage)
	engine := r.(*runner)

	// Add mock providers
	mockProvider1 := &mockProvider{}
	mockProvider2 := &mockProvider{}
	engine.AddPreStep(mockProvider1)
	engine.AddStep(mockProvider2)
	engine.AddPostStep(mockProvider1)

	recorder, err := r.Run(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, recorder)
}
