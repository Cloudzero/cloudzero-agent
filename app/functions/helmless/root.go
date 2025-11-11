// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package main implements a tool for comparing configured values against
// default values from a Helm chart. It produces a minimal YAML file containing
// only the differences, which is useful for understanding what values have been
// customized in a Helm deployment of the CloudZero Agent for Kubernetes.
package main

import (
	"fmt"
	"os"

	"github.com/cloudzero/cloudzero-agent/app/utils/helmless"
	"github.com/spf13/cobra"
)

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

		// Configure the helmless extraction
		config := helmless.Config{
			ConfiguredValuesPath: configuredPath,
			DefaultValuesPath:    defaultsPath,
		}

		// Extract the overrides
		result, err := helmless.Extract(config)
		if err != nil {
			return fmt.Errorf("extracting overrides: %w", err)
		}

		// Write output to file or stdout
		output := os.Stdout
		if outputPath != "-" {
			file, err := os.Create(outputPath)
			if err != nil {
				return fmt.Errorf("create output file: %w", err)
			}
			defer file.Close()
			output = file
		}

		if _, err := output.Write(result); err != nil {
			return fmt.Errorf("writing output: %w", err)
		}

		return nil
	},
}

// init initializes the command line flags.
func init() {
	rootCmd.Flags().StringP("configured", "c", "configured-values.yaml", "Path to configured values YAML file")
	rootCmd.Flags().StringP("defaults", "d", "", "Path to default values YAML file (uses embedded defaults if not provided)")
	rootCmd.Flags().StringP("output", "o", "-", "Path to output overrides YAML file")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
