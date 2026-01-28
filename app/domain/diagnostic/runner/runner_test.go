// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package runner

import (
	"context"
	"errors"
	"net/http"
	"testing"

	config "github.com/cloudzero/cloudzero-agent/app/config/validator"
	"github.com/cloudzero/cloudzero-agent/app/domain/diagnostic"
	"github.com/cloudzero/cloudzero-agent/app/domain/diagnostic/catalog"
	"github.com/cloudzero/cloudzero-agent/app/domain/diagnostic/kms"
	"github.com/cloudzero/cloudzero-agent/app/types/status"
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

func TestRunner_ShouldFail(t *testing.T) {
	tests := []struct {
		name        string
		enforce     bool
		hasFailures bool
		expected    bool
	}{
		{
			name:        "enforce false, no failures - should not fail",
			enforce:     false,
			hasFailures: false,
			expected:    false,
		},
		{
			name:        "enforce false, has failures - should not fail",
			enforce:     false,
			hasFailures: true,
			expected:    false,
		},
		{
			name:        "enforce true, no failures - should not fail",
			enforce:     true,
			hasFailures: false,
			expected:    false,
		},
		{
			name:        "enforce true, has failures - should fail",
			enforce:     true,
			hasFailures: true,
			expected:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &runner{
				enforce:     tt.enforce,
				hasFailures: tt.hasFailures,
			}
			assert.Equal(t, tt.expected, r.ShouldFail())
		})
	}
}

func TestRunner_EnforceSetFromStageConfig(t *testing.T) {
	tests := []struct {
		name            string
		stages          []config.Stage
		targetStage     string
		expectedEnforce bool
	}{
		{
			name: "enforce true from matching stage",
			stages: []config.Stage{
				{Name: config.ContextStageInit, Enforce: true, Checks: []string{}},
			},
			targetStage:     config.ContextStageInit,
			expectedEnforce: true,
		},
		{
			name: "enforce false from matching stage",
			stages: []config.Stage{
				{Name: config.ContextStageInit, Enforce: false, Checks: []string{}},
			},
			targetStage:     config.ContextStageInit,
			expectedEnforce: false,
		},
		{
			name: "enforce from correct stage when multiple stages exist",
			stages: []config.Stage{
				{Name: config.ContextStageInit, Enforce: true, Checks: []string{}},
				{Name: config.ContextStageStart, Enforce: false, Checks: []string{}},
			},
			targetStage:     config.ContextStageStart,
			expectedEnforce: false,
		},
		{
			name:            "enforce defaults to false when no matching stage",
			stages:          []config.Stage{},
			targetStage:     config.ContextStageInit,
			expectedEnforce: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Settings{
				Deployment: config.Deployment{
					AccountID:   "test-account",
					Region:      "test-region",
					ClusterName: "test-cluster",
				},
				Diagnostics: config.Diagnostics{
					Stages: tt.stages,
				},
			}

			// Use the mock provider for KMS
			originalNewProvider := kms.NewProvider
			kms.NewProvider = NewMockKMSProvider
			defer func() { kms.NewProvider = originalNewProvider }()

			reg := catalog.NewCatalog(context.Background(), cfg)
			r := NewRunner(cfg, reg, tt.targetStage)
			engine := r.(*runner)

			assert.Equal(t, tt.expectedEnforce, engine.enforce)
		})
	}
}

func TestRunner_HasFailuresTracking(t *testing.T) {
	tests := []struct {
		name                string
		checkPassing        bool
		expectedHasFailures bool
	}{
		{
			name:                "hasFailures false when check passes",
			checkPassing:        true,
			expectedHasFailures: false,
		},
		{
			name:                "hasFailures true when check fails",
			checkPassing:        false,
			expectedHasFailures: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
			r := NewRunner(cfg, reg, config.ContextStageStart) // Use post-start to avoid init-specific logic
			engine := r.(*runner)

			// Clear any default providers
			engine.pre = nil
			engine.plan = nil
			engine.post = nil

			// Add a mock provider that records a check result
			mockProv := &mockProvider{
				Test: func(ctx context.Context, client *http.Client, recorder status.Accessor) error {
					recorder.WriteToReport(func(cs *status.ClusterStatus) {
						cs.Checks = append(cs.Checks, &status.StatusCheck{
							Name:    "test-check",
							Passing: tt.checkPassing,
						})
					})
					return nil
				},
			}
			engine.AddStep(mockProv)

			_, err := r.Run(context.Background())
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedHasFailures, engine.hasFailures)
		})
	}
}

func TestRunner_ShouldFailIntegration(t *testing.T) {
	// Integration test: verify the full flow from config -> enforce -> hasFailures -> ShouldFail
	tests := []struct {
		name               string
		enforceInConfig    bool
		checkPassing       bool
		expectedShouldFail bool
	}{
		{
			name:               "enforce true + failing check = should fail",
			enforceInConfig:    true,
			checkPassing:       false,
			expectedShouldFail: true,
		},
		{
			name:               "enforce true + passing check = should not fail",
			enforceInConfig:    true,
			checkPassing:       true,
			expectedShouldFail: false,
		},
		{
			name:               "enforce false + failing check = should not fail",
			enforceInConfig:    false,
			checkPassing:       false,
			expectedShouldFail: false,
		},
		{
			name:               "enforce false + passing check = should not fail",
			enforceInConfig:    false,
			checkPassing:       true,
			expectedShouldFail: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Settings{
				Deployment: config.Deployment{
					AccountID:   "test-account",
					Region:      "test-region",
					ClusterName: "test-cluster",
				},
				Diagnostics: config.Diagnostics{
					Stages: []config.Stage{
						{
							Name:    config.ContextStageStart,
							Enforce: tt.enforceInConfig,
							Checks:  []string{},
						},
					},
				},
			}

			// Use the mock provider for KMS
			originalNewProvider := kms.NewProvider
			kms.NewProvider = NewMockKMSProvider
			defer func() { kms.NewProvider = originalNewProvider }()

			reg := catalog.NewCatalog(context.Background(), cfg)
			r := NewRunner(cfg, reg, config.ContextStageStart)
			engine := r.(*runner)

			// Clear default providers and add our mock
			engine.pre = nil
			engine.plan = nil
			engine.post = nil

			mockProv := &mockProvider{
				Test: func(ctx context.Context, client *http.Client, recorder status.Accessor) error {
					recorder.WriteToReport(func(cs *status.ClusterStatus) {
						cs.Checks = append(cs.Checks, &status.StatusCheck{
							Name:    "test-check",
							Passing: tt.checkPassing,
						})
					})
					return nil
				},
			}
			engine.AddStep(mockProv)

			_, err := r.Run(context.Background())
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedShouldFail, r.ShouldFail())
		})
	}
}

func TestRunner_IsEnforced(t *testing.T) {
	tests := []struct {
		name     string
		enforce  bool
		expected bool
	}{
		{
			name:     "IsEnforced returns true when enforce is true",
			enforce:  true,
			expected: true,
		},
		{
			name:     "IsEnforced returns false when enforce is false",
			enforce:  false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &runner{
				enforce: tt.enforce,
			}
			assert.Equal(t, tt.expected, r.IsEnforced())
		})
	}
}
