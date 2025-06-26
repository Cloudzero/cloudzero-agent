// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/cloudzero/cloudzero-agent/app/utils/scout"
	"github.com/cloudzero/cloudzero-agent/app/utils/scout/types"
	"github.com/spf13/cobra"
)

var (
	// CLI flags
	outputFormat string
	timeout      time.Duration
	verbose      bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "scout",
	Short: "CloudZero Scout - Cloud Environment Information Tool",
	Long: `CloudZero Scout is a CLI tool that automatically detects and retrieves 
information about the current cloud environment including cloud provider, 
region, and account/subscription/project ID.

By default, running 'scout' will retrieve full environment information.
Use 'scout detect' to only identify the cloud provider.

Supported cloud providers:
- Amazon Web Services (AWS)
- Google Cloud`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runScout()
	},
}

// infoCmd represents the info command
var infoCmd = &cobra.Command{
	Use:   "info",
	Short: "Retrieve cloud environment information",
	Long: `Retrieve detailed information about the current cloud environment including:
- Cloud provider (aws, google)
- Region/location
- Account ID/Subscription ID/Project ID`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runScout()
	},
}

// detectCmd represents the detect command
var detectCmd = &cobra.Command{
	Use:   "detect",
	Short: "Detect the current cloud provider",
	Long:  `Detect which cloud provider the current environment is running on without retrieving full metadata.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDetect()
	},
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "json", "Output format (json, yaml, table)")
	rootCmd.PersistentFlags().DurationVarP(&timeout, "timeout", "t", 10*time.Second, "Timeout for metadata retrieval")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

	// Add subcommands
	rootCmd.AddCommand(infoCmd)
	rootCmd.AddCommand(detectCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runScout executes the main scout functionality
func runScout() error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if verbose {
		fmt.Fprintf(os.Stderr, "Initializing CloudZero Scout...\n")
		fmt.Fprintf(os.Stderr, "Timeout: %v\n", timeout)
	}

	// Create scout instance with auto-detection
	s := scout.NewScout()

	if verbose {
		fmt.Fprintf(os.Stderr, "Retrieving environment information...\n")
	}

	// Get environment information
	info, err := s.EnvironmentInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve environment information: %w", err)
	}

	// Output the results
	return outputEnvironmentInfo(info)
}

// runDetect executes cloud provider detection only
func runDetect() error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if verbose {
		fmt.Fprintf(os.Stderr, "Detecting cloud provider...\n")
	}

	// Create scout instance with auto-detection
	s := scout.NewScout()

	// Get environment information (which includes detection)
	info, err := s.EnvironmentInfo(ctx)
	if err != nil {
		// If detection failed, return unknown
		provider := types.CloudProviderUnknown
		return outputCloudProvider(provider)
	}

	return outputCloudProvider(info.CloudProvider)
}

// outputCloudProvider outputs just the cloud provider in the specified format
func outputCloudProvider(provider types.CloudProvider) error {
	switch outputFormat {
	case "json":
		result := map[string]string{"cloudProvider": string(provider)}
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	case "table":
		fmt.Printf("Cloud Provider: %s\n", provider)
	default:
		fmt.Println(provider)
	}

	return nil
}

// outputEnvironmentInfo outputs the environment information in the specified format
func outputEnvironmentInfo(info *types.EnvironmentInfo) error {
	switch outputFormat {
	case "json":
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(info)

	case "yaml":
		fmt.Printf("cloudProvider: %s\n", info.CloudProvider)
		fmt.Printf("region: %s\n", info.Region)
		fmt.Printf("accountId: %s\n", info.AccountID)

	case "table":
		fmt.Printf("%-15s: %s\n", "Cloud Provider", info.CloudProvider)
		fmt.Printf("%-15s: %s\n", "Region", info.Region)
		fmt.Printf("%-15s: %s\n", "Account ID", info.AccountID)

	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}

	return nil
}
