// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"testing"

	config "github.com/cloudzero/cloudzero-agent/app/config/validator"
	"github.com/stretchr/testify/assert"
)

func TestDiagnostics_IsValidDiagnostics(t *testing.T) {
	tcases := []struct {
		name       string
		diagnostic string
		expected   bool
	}{
		{
			name:       "DiagnosticAPIKey",
			diagnostic: config.DiagnosticAPIKey,
			expected:   true,
		},
		{
			name:       "DiagnosticK8sVersion",
			diagnostic: config.DiagnosticK8sVersion,
			expected:   true,
		},
		{
			name:       "DiagnosticKMS",
			diagnostic: config.DiagnosticKMS,
			expected:   true,
		},
		{
			name:       "DiagnosticScrapeConfig",
			diagnostic: config.DiagnosticScrapeConfig,
			expected:   true,
		},
		{
			name:       "UnknownDiagnostic",
			diagnostic: "bogus",
			expected:   false,
		},
	}
	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			assert.True(t, config.IsValidDiagnostic(tc.diagnostic) == tc.expected)
		})
	}
}

func TestDiagnostics_IsValidStage(t *testing.T) {
	tcases := []struct {
		name     string
		stage    string
		expected bool
	}{
		{
			name:     "ContextStageInit",
			stage:    config.ContextStageInit,
			expected: true,
		},
		{
			name:     "ContextStageStart",
			stage:    config.ContextStageStart,
			expected: true,
		},
		{
			name:     "ContextStageStop",
			stage:    config.ContextStageStop,
			expected: true,
		},
		{
			name:     "UnknownStage",
			stage:    "bogus",
			expected: false,
		},
	}
	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			assert.True(t, config.IsValidStage(tc.stage) == tc.expected)
		})
	}
}

func TestStage_Validate(t *testing.T) {
	tcases := []struct {
		name     string
		stage    *config.Stage
		expected bool
	}{
		{
			name: "ValidStage",
			stage: &config.Stage{
				Name:    config.ContextStageInit,
				Enforce: false,
				Checks:  []string{config.DiagnosticAPIKey},
			},
			expected: false,
		},
		{
			name: "InvalidStage",
			stage: &config.Stage{
				Name:    "bogus",
				Enforce: false,
				Checks:  []string{config.DiagnosticAPIKey},
			},
			expected: true,
		},
		{
			name: "InvalidStageCheck",
			stage: &config.Stage{
				Name:    config.ContextStageInit,
				Enforce: false,
				Checks:  []string{"bogus"},
			},
			expected: true,
		},
	}
	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.stage.Validate()
			if tc.expected && err == nil {
				t.Errorf("Expected error, got nil")
			}
			if !tc.expected && err != nil {
				t.Errorf("Expected no error, got %v", err)
			}
		})
	}
}
