// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
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
	// ShouldFail returns true only when requiredFailures is true
	tests := []struct {
		name             string
		requiredFailures bool
		expected         bool
	}{
		{
			name:             "no required failures - should not fail",
			requiredFailures: false,
			expected:         false,
		},
		{
			name:             "has required failures - should fail",
			requiredFailures: true,
			expected:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &runner{
				requiredFailures: tt.requiredFailures,
			}
			assert.Equal(t, tt.expected, r.ShouldFail())
		})
	}
}

func TestRunner_CheckTypesFromConfig(t *testing.T) {
	tests := []struct {
		name              string
		stages            []config.Stage
		targetStage       string
		expectedCheckType map[string]config.CheckType
	}{
		{
			name: "required check type from config",
			stages: []config.Stage{
				{
					Name: config.ContextStageStart,
					Checks: []config.CheckConfig{
						{Name: "api_key_valid", Type: config.CheckTypeRequired},
					},
				},
			},
			targetStage: config.ContextStageStart,
			expectedCheckType: map[string]config.CheckType{
				"api_key_valid": config.CheckTypeRequired,
			},
		},
		{
			name: "optional check type from config",
			stages: []config.Stage{
				{
					Name: config.ContextStageStart,
					Checks: []config.CheckConfig{
						{Name: "k8s_version", Type: config.CheckTypeOptional},
					},
				},
			},
			targetStage: config.ContextStageStart,
			expectedCheckType: map[string]config.CheckType{
				"k8s_version": config.CheckTypeOptional,
			},
		},
		{
			name: "informative check type from config",
			stages: []config.Stage{
				{
					Name: config.ContextStageStart,
					Checks: []config.CheckConfig{
						{Name: "k8s_provider", Type: config.CheckTypeInformative},
					},
				},
			},
			targetStage: config.ContextStageStart,
			expectedCheckType: map[string]config.CheckType{
				"k8s_provider": config.CheckTypeInformative,
			},
		},
		{
			name: "multiple check types from config",
			stages: []config.Stage{
				{
					Name: config.ContextStageStart,
					Checks: []config.CheckConfig{
						{Name: "api_key_valid", Type: config.CheckTypeRequired},
						{Name: "k8s_version", Type: config.CheckTypeOptional},
						{Name: "k8s_provider", Type: config.CheckTypeInformative},
					},
				},
			},
			targetStage: config.ContextStageStart,
			expectedCheckType: map[string]config.CheckType{
				"api_key_valid": config.CheckTypeRequired,
				"k8s_version":   config.CheckTypeOptional,
				"k8s_provider":  config.CheckTypeInformative,
			},
		},
		{
			name:              "empty checkTypes when no matching stage",
			stages:            []config.Stage{},
			targetStage:       config.ContextStageStart,
			expectedCheckType: map[string]config.CheckType{},
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

			assert.Equal(t, tt.expectedCheckType, engine.checkTypes)
		})
	}
}

func TestRunner_RequiredFailuresTracking(t *testing.T) {
	tests := []struct {
		name                     string
		checkType                config.CheckType
		checkPassing             bool
		expectedRequiredFailures bool
	}{
		{
			name:                     "required check passing - no required failures",
			checkType:                config.CheckTypeRequired,
			checkPassing:             true,
			expectedRequiredFailures: false,
		},
		{
			name:                     "required check failing - has required failures",
			checkType:                config.CheckTypeRequired,
			checkPassing:             false,
			expectedRequiredFailures: true,
		},
		{
			name:                     "optional check passing - no required failures",
			checkType:                config.CheckTypeOptional,
			checkPassing:             true,
			expectedRequiredFailures: false,
		},
		{
			name:                     "optional check failing - no required failures",
			checkType:                config.CheckTypeOptional,
			checkPassing:             false,
			expectedRequiredFailures: false,
		},
		{
			name:                     "informative check passing - no required failures",
			checkType:                config.CheckTypeInformative,
			checkPassing:             true,
			expectedRequiredFailures: false,
		},
		{
			name:                     "informative check failing - no required failures",
			checkType:                config.CheckTypeInformative,
			checkPassing:             false,
			expectedRequiredFailures: false,
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
			r := NewRunner(cfg, reg, config.ContextStageStart)
			engine := r.(*runner)

			// Clear any default providers
			engine.pre = nil
			engine.plan = nil
			engine.post = nil

			// Set the check type for our test check
			engine.checkTypes["test-check"] = tt.checkType

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
			assert.Equal(t, tt.expectedRequiredFailures, engine.requiredFailures)
		})
	}
}

func TestRunner_ShouldFailIntegration(t *testing.T) {
	// Integration test: verify the full flow from config -> checkTypes -> requiredFailures -> ShouldFail
	tests := []struct {
		name               string
		checkType          config.CheckType
		checkPassing       bool
		expectedShouldFail bool
	}{
		{
			name:               "required check failing = should fail",
			checkType:          config.CheckTypeRequired,
			checkPassing:       false,
			expectedShouldFail: true,
		},
		{
			name:               "required check passing = should not fail",
			checkType:          config.CheckTypeRequired,
			checkPassing:       true,
			expectedShouldFail: false,
		},
		{
			name:               "optional check failing = should not fail",
			checkType:          config.CheckTypeOptional,
			checkPassing:       false,
			expectedShouldFail: false,
		},
		{
			name:               "optional check passing = should not fail",
			checkType:          config.CheckTypeOptional,
			checkPassing:       true,
			expectedShouldFail: false,
		},
		{
			name:               "informative check failing = should not fail",
			checkType:          config.CheckTypeInformative,
			checkPassing:       false,
			expectedShouldFail: false,
		},
		{
			name:               "informative check passing = should not fail",
			checkType:          config.CheckTypeInformative,
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
							Name: config.ContextStageStart,
							Checks: []config.CheckConfig{
								{Name: "test-check", Type: tt.checkType},
							},
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
