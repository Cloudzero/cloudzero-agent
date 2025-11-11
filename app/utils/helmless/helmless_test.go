// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package helmless

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestExtract(t *testing.T) {
	tests := []struct {
		name         string
		config       Config
		wantErr      bool
		validateFunc func(t *testing.T, result []byte)
	}{
		{
			name: "no configured values",
			config: Config{},
			wantErr: true,
		},
		{
			name: "use embedded defaults",
			config: Config{
				ConfiguredValuesBytes: []byte(`
components:
  collector:
    replicas: 5
`),
			},
			wantErr: false,
			validateFunc: func(t *testing.T, result []byte) {
				var overrides map[string]interface{}
				if err := yaml.Unmarshal(result, &overrides); err != nil {
					t.Fatalf("failed to unmarshal result: %v", err)
				}

				// Should have extracted the replicas override
				components, ok := overrides["components"].(map[string]interface{})
				if !ok {
					t.Fatalf("expected components to be present")
				}

				collector, ok := components["collector"].(map[string]interface{})
				if !ok {
					t.Fatalf("expected collector to be present")
				}

				replicas, ok := collector["replicas"].(int)
				if !ok {
					t.Fatalf("expected replicas to be an int")
				}

				if replicas != 5 {
					t.Errorf("expected replicas=5, got %v", replicas)
				}
			},
		},
		{
			name: "invalid configured YAML",
			config: Config{
				ConfiguredValuesBytes: []byte(`invalid: yaml: : :`),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Extract(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Extract() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.validateFunc != nil && err == nil {
				tt.validateFunc(t, result)
			}
		})
	}
}

func TestExtractWithFilePaths(t *testing.T) {
	// Create temporary directory for test files
	tempDir := t.TempDir()

	// Create configured values file
	configuredFile := filepath.Join(tempDir, "configured.yaml")
	configuredContent := []byte(`
replicas: 5
image: nginx:latest
config:
  database:
    host: custom.db.com
    port: 5432
`)
	if err := os.WriteFile(configuredFile, configuredContent, 0o644); err != nil {
		t.Fatalf("failed to write configured file: %v", err)
	}

	// Create default values file
	defaultsFile := filepath.Join(tempDir, "defaults.yaml")
	defaultsContent := []byte(`
replicas: 3
image: nginx:latest
config:
  database:
    host: localhost
    port: 5432
`)
	if err := os.WriteFile(defaultsFile, defaultsContent, 0o644); err != nil {
		t.Fatalf("failed to write defaults file: %v", err)
	}

	tests := []struct {
		name         string
		config       Config
		wantErr      bool
		validateFunc func(t *testing.T, result []byte)
	}{
		{
			name: "extract from file paths",
			config: Config{
				ConfiguredValuesPath: configuredFile,
				DefaultValuesPath:    defaultsFile,
			},
			wantErr: false,
			validateFunc: func(t *testing.T, result []byte) {
				var overrides map[string]interface{}
				if err := yaml.Unmarshal(result, &overrides); err != nil {
					t.Fatalf("failed to unmarshal result: %v", err)
				}

				if overrides["replicas"] != 5 {
					t.Errorf("expected replicas=5, got %v", overrides["replicas"])
				}

				config, ok := overrides["config"].(map[string]interface{})
				if !ok {
					t.Fatalf("expected config to be present")
				}

				database, ok := config["database"].(map[string]interface{})
				if !ok {
					t.Fatalf("expected database to be present")
				}

				if database["host"] != "custom.db.com" {
					t.Errorf("expected host=custom.db.com, got %v", database["host"])
				}
			},
		},
		{
			name: "extract from configured path with embedded defaults",
			config: Config{
				ConfiguredValuesPath: configuredFile,
			},
			wantErr: false,
			validateFunc: func(t *testing.T, result []byte) {
				var overrides map[string]interface{}
				if err := yaml.Unmarshal(result, &overrides); err != nil {
					t.Fatalf("failed to unmarshal result: %v", err)
				}
				// Should have some overrides
				if len(overrides) == 0 {
					t.Errorf("expected some overrides")
				}
			},
		},
		{
			name: "invalid configured file path",
			config: Config{
				ConfiguredValuesPath: "/nonexistent/file.yaml",
				DefaultValuesPath:    defaultsFile,
			},
			wantErr: true,
		},
		{
			name: "invalid defaults file path",
			config: Config{
				ConfiguredValuesPath: configuredFile,
				DefaultValuesPath:    "/nonexistent/defaults.yaml",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Extract(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Extract() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.validateFunc != nil && err == nil {
				tt.validateFunc(t, result)
			}
		})
	}
}
