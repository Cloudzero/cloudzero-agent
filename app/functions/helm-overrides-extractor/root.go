// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.

// Package main implements a tool for comparing configured values against
// default values from a Helm chart. It produces a minimal YAML file containing
// only the differences, which is useful for understanding what values have been
// customized in a Helm deployment of the CloudZero Agent for Kubernetes.
package main

import (
	"fmt"
	"os"

	"github.com/google/go-cmp/cmp"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// Config holds the configuration for the Helm overrides extractor.
type Config struct {
	// ConfiguredValuesPath is the path to the YAML file containing the
	// configured values.
	ConfiguredValuesPath string
	// DefaultValuesPath is the path to the YAML file containing the default
	// values from the Helm chart.
	DefaultValuesPath string
	// OutputPath is the file to write the output to. If nil, output will be
	// written to stdout.
	OutputPath *os.File
}

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:   "helm-overrides-extractor",
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

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// init initializes the command line flags.
func init() {
	rootCmd.Flags().StringP("configured", "c", "configured-values.yaml", "Path to configured values YAML file")
	rootCmd.Flags().StringP("defaults", "d", "default-values.yaml", "Path to default values YAML file")
	rootCmd.Flags().StringP("output", "o", "-", "Path to output overrides YAML file")
}

// run executes the main logic of the program.
func run(config Config) error {
	configuredValues, err := readYAML(config.ConfiguredValuesPath)
	if err != nil {
		return fmt.Errorf("reading configured values: %w", err)
	}

	defaultValues, err := readYAML(config.DefaultValuesPath)
	if err != nil {
		return fmt.Errorf("reading default values: %w", err)
	}

	// Remove kubeStateMetrics; it's special.
	delete(configuredValues, "kubeStateMetrics")

	overrides := diffMaps(configuredValues, defaultValues)

	if err := writeYAML(config.OutputPath, overrides); err != nil {
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

// diffMaps returns a map containing only the keys whose values differ from the
// defaults. It recursively compares maps and arrays, only including values that
// are significant (non-empty strings, non-zero numbers, non-empty maps/arrays,
// or boolean values).
func diffMaps(configured, defaults map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for key, confVal := range configured {
		defVal, exists := defaults[key]
		if !exists {
			// If key doesn't exist in defaults and is significant, it's an
			// override
			if isSignificantValue(confVal) {
				result[key] = confVal
			}
			continue
		}

		// Both exist, compare values
		if !cmp.Equal(confVal, defVal) {
			// If they're both maps, recursively compare them
			confMap, confIsMap := confVal.(map[string]interface{})
			defMap, defIsMap := defVal.(map[string]interface{})
			if confIsMap && defIsMap {
				if len(confMap) > 0 {
					diff := diffMaps(confMap, defMap)
					if len(diff) > 0 {
						result[key] = diff
					}
				}
				continue
			}

			// For non-map values, include if significant
			if isSignificantValue(confVal) {
				result[key] = confVal
			}
		}
	}
	return result
}

// isSignificantValue determines if a value is significant enough to be included
// in the output. A value is considered significant if it is:
//
//   - A non-empty string
//   - A non-zero number
//   - A non-empty map or array
//   - A boolean value (true/false)
func isSignificantValue(value interface{}) bool {
	switch v := value.(type) {
	case map[string]interface{}:
		// For maps, check if any of their values are significant
		for _, val := range v {
			if isSignificantValue(val) {
				return true
			}
		}
		return false
	case []interface{}:
		// For slices, check if any of their values are significant
		for _, val := range v {
			if isSignificantValue(val) {
				return true
			}
		}
		return false
	case string:
		return v != ""
	case bool:
		return true
	case int, int64, float64:
		return true
	case nil:
		return false
	default:
		return fmt.Sprintf("%v", v) != ""
	}
}

func main() {
	Execute()
}
