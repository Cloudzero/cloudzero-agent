// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package main implements a tool for comparing configured values against
// default values from a Helm chart. It produces a minimal YAML file containing
// only the differences, which is useful for understanding what values have been
// customized in a Helm deployment of the CloudZero Agent for Kubernetes.
package main

//go:generate make -C ../../.. app/functions/helmless/default-values.yaml

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/cloudzero/cloudzero-agent/app/functions/helmless/overrides"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

//go:embed default-values.yaml
var embeddedDefaultValues []byte

// Config holds the configuration for the Helm overrides extractor.
type Config struct {
	// ConfiguredValuesPath is the path to the YAML file containing the
	// configured values.
	ConfiguredValuesPath string
	// DefaultValuesPath is the path to the YAML file containing the default
	// values from the Helm chart. If empty, embedded defaults will be used.
	DefaultValuesPath string
	// OutputPath is the file to write the output to. If nil, output will be
	// written to stdout.
	OutputPath *os.File
}

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "helmless",
	Short: "Compare configured values against Helm chart defaults",
	Long: `A tool to compare configured values from a YAML file against default values from a Helm chart,
identifying differences and creating a minimal overrides file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configuredPath, err := cmd.Flags().GetString("configured")
		if err != nil {
			return err
		}
		defaultsPath, err := cmd.Flags().GetString("defaults")
		if err != nil {
			return err
		}
		outputPath, err := cmd.Flags().GetString("output")
		if err != nil {
			return err
		}
		config := Config{
			ConfiguredValuesPath: configuredPath,
			DefaultValuesPath:    defaultsPath,
			OutputPath:           os.Stdout,
		}
		if outputPath != "-" {
			output, err := os.Create(outputPath)
			if err != nil {
				return fmt.Errorf("create output file: %w", err)
			}
			config.OutputPath = output
		}
		return run(config)
	},
}

// init initializes the command line flags.
func init() {
	rootCmd.Flags().StringP("configured", "c", "configured-values.yaml", "Path to configured values YAML file")
	rootCmd.Flags().StringP("defaults", "d", "", "Path to default values YAML file (uses embedded defaults if not provided)")
	rootCmd.Flags().StringP("output", "o", "-", "Path to output overrides YAML file")
}

// run executes the main logic of the program.
func run(config Config) error {
	configuredValues, err := readYAML(config.ConfiguredValuesPath)
	if err != nil {
		return fmt.Errorf("reading configured values: %w", err)
	}

	var defaultValues map[string]interface{}
	if config.DefaultValuesPath == "" {
		// Use embedded defaults
		defaultValues, err = readYAMLFromBytes(embeddedDefaultValues)
		if err != nil {
			return fmt.Errorf("reading embedded default values: %w", err)
		}
	} else {
		// Use provided defaults file
		defaultValues, err = readYAML(config.DefaultValuesPath)
		if err != nil {
			return fmt.Errorf("reading default values: %w", err)
		}
	}

	// Create extractor with kubeStateMetrics excluded (it's an alias for a
	// subchart, and subchart values aren't included in the output of
	// `helm show values ./helm`)
	extractor := overrides.NewExtractor("kubeStateMetrics")
	overridesMap := extractor.Extract(configuredValues, defaultValues)

	if err := writeYAML(config.OutputPath, overridesMap); err != nil {
		return fmt.Errorf("writing overrides: %w", err)
	}

	return nil
}

// readYAML reads and parses a YAML file into a map.
func readYAML(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return readYAMLFromBytes(data)
}

// readYAMLFromBytes parses YAML data from bytes into a map.
func readYAMLFromBytes(data []byte) (map[string]interface{}, error) {
	var result map[string]interface{}
	if err := yaml.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// writeYAML writes a value to a file in YAML format.
func writeYAML(output *os.File, data interface{}) error {
	encoder := yaml.NewEncoder(output)
	defer encoder.Close()

	encoder.SetIndent(2)
	if err := encoder.Encode(data); err != nil {
		return err
	}

	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
