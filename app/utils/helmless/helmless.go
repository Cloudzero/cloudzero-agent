// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package helmless provides utilities for comparing configured values against
// default values from a Helm chart. It produces a minimal YAML file containing
// only the differences, which is useful for understanding what values have been
// customized in a Helm deployment of the CloudZero Agent for Kubernetes.
package helmless

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"github.com/cloudzero/cloudzero-agent/app/domain/helmless"
	"github.com/cloudzero/cloudzero-agent/app/utils/helmless/overrides"
	"gopkg.in/yaml.v3"
)

// Config specifies the configured and default Helm values to compare.
//
// Configured values (required) can be provided in two ways:
//   - ConfiguredValuesPath: Path to a YAML file containing configured values
//   - ConfiguredValuesBytes: Raw bytes of configured values YAML (for in-memory processing)
//
// Default values (optional) can be provided as:
//   - DefaultValuesPath: Path to a YAML file containing default values
//   - If omitted, embedded chart defaults are used automatically
//
// At least one configured values source must be provided. If both path and bytes are
// provided for configured values, bytes takes precedence.
type Config struct {
	ConfiguredValuesPath  string // Path to configured values YAML file
	ConfiguredValuesBytes []byte // Configured values as bytes (alternative to path)
	DefaultValuesPath     string // Path to default values YAML file (optional, uses embedded defaults if empty)
}

// Extract compares configured values against default values and returns YAML containing only the differences.
//
// This function identifies which values have been customized from their defaults, producing
// a minimal override file. This is useful for understanding what values have been modified
// in a Helm deployment of the CloudZero Agent for Kubernetes.
//
// The function:
//   - Loads configured values from the provided path or bytes
//   - Loads default values from the provided path, or uses embedded chart defaults
//   - Compares the two value sets to identify differences
//   - Returns YAML bytes containing only the overridden values
//
// The kubeStateMetrics key is automatically excluded from comparison as it's a subchart alias.
//
// Returns an error if:
//   - No configured values source is provided
//   - YAML parsing fails for either input
//   - File reading fails for provided paths
//   - YAML encoding of the result fails
func Extract(config Config) ([]byte, error) {
	// Load configured values
	var configuredValues map[string]interface{}
	var err error

	switch {
	case len(config.ConfiguredValuesBytes) > 0:
		configuredValues, err = readYAMLFromBytes(config.ConfiguredValuesBytes)
		if err != nil {
			return nil, fmt.Errorf("reading configured values from bytes: %w", err)
		}
	case config.ConfiguredValuesPath != "":
		configuredValues, err = readYAML(config.ConfiguredValuesPath)
		if err != nil {
			return nil, fmt.Errorf("reading configured values from path: %w", err)
		}
	default:
		return nil, errors.New("no configured values provided (need path or bytes)")
	}

	// Load default values
	var defaultValues map[string]interface{}

	switch {
	case config.DefaultValuesPath != "":
		// Use provided file path
		defaultValues, err = readYAML(config.DefaultValuesPath)
		if err != nil {
			return nil, fmt.Errorf("reading default values from path: %w", err)
		}
	default:
		// Use embedded defaults
		defaultValues, err = readYAMLFromBytes(helmless.DefaultValues)
		if err != nil {
			return nil, fmt.Errorf("reading embedded default values: %w", err)
		}
	}

	// Extract overrides (exclude kubeStateMetrics - it's a subchart alias)
	extractor := overrides.NewExtractor("kubeStateMetrics")
	overridesMap := extractor.Extract(configuredValues, defaultValues)

	// Encode to YAML
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(overridesMap); err != nil {
		return nil, fmt.Errorf("encoding overrides to YAML: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("closing YAML encoder: %w", err)
	}

	return buf.Bytes(), nil
}

func readYAML(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return readYAMLFromBytes(data)
}

func readYAMLFromBytes(data []byte) (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := yaml.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}
