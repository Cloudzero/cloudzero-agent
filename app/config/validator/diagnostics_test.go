// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
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
				Name: config.ContextStageInit,
				Checks: []config.CheckConfig{
					{Name: config.DiagnosticAPIKey, Type: config.CheckTypeRequired},
				},
			},
			expected: false,
		},
		{
			name: "InvalidStage",
			stage: &config.Stage{
				Name: "bogus",
				Checks: []config.CheckConfig{
					{Name: config.DiagnosticAPIKey, Type: config.CheckTypeRequired},
				},
			},
			expected: true,
		},
		{
			name: "InvalidStageCheck",
			stage: &config.Stage{
				Name: config.ContextStageInit,
				Checks: []config.CheckConfig{
					{Name: "bogus", Type: config.CheckTypeRequired},
				},
			},
			expected: true,
		},
		{
			name: "InvalidCheckType",
			stage: &config.Stage{
				Name: config.ContextStageInit,
				Checks: []config.CheckConfig{
					{Name: config.DiagnosticAPIKey, Type: "invalid"},
				},
			},
			expected: true,
		},
		{
			name: "EmptyCheckTypeDefaultsToOptional",
			stage: &config.Stage{
				Name: config.ContextStageInit,
				Checks: []config.CheckConfig{
					{Name: config.DiagnosticAPIKey, Type: ""},
				},
			},
			expected: false, // Empty type defaults to optional, which is valid
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

func TestIsValidCheckType(t *testing.T) {
	tcases := []struct {
		name     string
		input    string
		expected bool
	}{
		{name: "required", input: "required", expected: true},
		{name: "optional", input: "optional", expected: true},
		{name: "informative", input: "informative", expected: true},
		{name: "Required_uppercase", input: "Required", expected: true},
		{name: "OPTIONAL_uppercase", input: "OPTIONAL", expected: true},
		{name: "invalid", input: "invalid", expected: false},
		{name: "empty", input: "", expected: false},
	}
	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, config.IsValidCheckType(tc.input))
		})
	}
}

func TestCheckConfig_DefaultType(t *testing.T) {
	check := &config.CheckConfig{Name: config.DiagnosticAPIKey, Type: ""}
	err := check.Validate()
	assert.NoError(t, err)
	assert.Equal(t, config.CheckTypeOptional, check.Type, "empty type should default to optional")
}

func TestCheckConfig_Validate(t *testing.T) {
	tcases := []struct {
		name     string
		check    *config.CheckConfig
		hasError bool
	}{
		{
			name:     "valid required check",
			check:    &config.CheckConfig{Name: config.DiagnosticAPIKey, Type: config.CheckTypeRequired},
			hasError: false,
		},
		{
			name:     "valid optional check",
			check:    &config.CheckConfig{Name: config.DiagnosticK8sVersion, Type: config.CheckTypeOptional},
			hasError: false,
		},
		{
			name:     "valid informative check",
			check:    &config.CheckConfig{Name: config.DiagnosticK8sProvider, Type: config.CheckTypeInformative},
			hasError: false,
		},
		{
			name:     "empty type defaults to optional",
			check:    &config.CheckConfig{Name: config.DiagnosticAPIKey, Type: ""},
			hasError: false,
		},
		{
			name:     "invalid check name",
			check:    &config.CheckConfig{Name: "bogus", Type: config.CheckTypeRequired},
			hasError: true,
		},
		{
			name:     "invalid check type",
			check:    &config.CheckConfig{Name: config.DiagnosticAPIKey, Type: "bogus"},
			hasError: true,
		},
	}
	for _, tc := range tcases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.check.Validate()
			if tc.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
