// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/cloudzero/cloudzero-agent/app/domain/certificate"
	"github.com/cloudzero/cloudzero-agent/app/domain/k8s"
)

var (
	serviceName       string
	namespace         string
	secretName        string
	webhookName       string
	keySize           int
	validityDuration  string
	algorithm         string
	enableLabels      bool
	enableAnnotations bool
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "cloudzero-certifik8s",
	Short: "Generate and manage TLS certificates for CloudZero Agent",
	Long: `CloudZero Certifik8s is a tool for generating and managing TLS certificates
for CloudZero Agent components, particularly webhook servers.

The tool can generate certificates with configurable algorithms (RSA, ECDSA, Ed25519),
key sizes, and validity periods. It automatically updates Kubernetes secrets and
webhook configurations with the new certificates.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Default to generate command for backward compatibility
		return generateCmd.RunE(cmd, args)
	},
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate a new TLS certificate",
	Long: `Generate a new TLS certificate with the specified parameters and update
Kubernetes resources accordingly.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Parse validity duration
		duration, err := time.ParseDuration(validityDuration)
		if err != nil {
			return fmt.Errorf("invalid validity duration '%s': %w", validityDuration, err)
		}

		// Create Kubernetes client
		k8sClient, err := k8s.NewCertificateClient()
		if err != nil {
			return fmt.Errorf("failed to create Kubernetes client: %w", err)
		}

		// Create certificate service
		certService := certificate.NewCertificateService(k8sClient)

		// Generate certificate
		certData, err := certService.GenerateCertificate(
			context.Background(),
			serviceName,
			namespace,
			keySize,
			duration,
			algorithm,
		)
		if err != nil {
			return fmt.Errorf("failed to generate certificate: %w", err)
		}

		// Update Kubernetes resources
		err = certService.UpdateResources(
			context.Background(),
			namespace,
			secretName,
			webhookName,
			certData,
		)
		if err != nil {
			return fmt.Errorf("failed to update resources: %w", err)
		}

		fmt.Println("Certificate generated and resources updated successfully")
		return nil
	},
}

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate existing certificate",
	Long: `Validate that an existing TLS certificate is properly configured
in the Kubernetes secret.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Create Kubernetes client
		k8sClient, err := k8s.NewCertificateClient()
		if err != nil {
			return fmt.Errorf("failed to create Kubernetes client: %w", err)
		}

		// Create certificate service
		certService := certificate.NewCertificateService(k8sClient)

		// Validate certificate
		isValid, err := certService.ValidateExistingCertificate(
			context.Background(),
			namespace,
			secretName,
		)
		if err != nil {
			return fmt.Errorf("failed to validate certificate: %w", err)
		}

		if isValid {
			fmt.Println("Certificate is valid and properly configured")
		} else {
			fmt.Println("Certificate is missing or improperly configured")
		}
		return nil
	},
}

var patchCmd = &cobra.Command{
	Use:   "patch",
	Short: "Patch existing certificate",
	Long: `Patch an existing TLS certificate with new data. This is useful
for updating certificates without regenerating them.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// For patch command, we need to read certificate data from somewhere
		// This is a placeholder - in practice, you might read from files or stdin
		fmt.Println("Patch command not yet implemented - use generate instead")
		return nil
	},
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&serviceName, "service-name", "", "Service name for the certificate")
	rootCmd.PersistentFlags().StringVar(&namespace, "namespace", "", "Kubernetes namespace")
	rootCmd.PersistentFlags().StringVar(&secretName, "secret-name", "", "Name of the TLS secret")
	rootCmd.PersistentFlags().StringVar(&webhookName, "webhook-name", "", "Name of the webhook configuration")

	// Generate command flags
	generateCmd.Flags().IntVar(&keySize, "key-size", 2048, "Key size in bits (RSA: 2048+, ECDSA: 256/384/521, Ed25519: ignored)")
	generateCmd.Flags().StringVar(&validityDuration, "validity-duration", "876000h", "Certificate validity period (e.g., '1h', '365d', '876000h')")
	generateCmd.Flags().StringVar(&algorithm, "algorithm", "RSA", "Certificate algorithm (RSA, ECDSA, Ed25519)")
	generateCmd.Flags().BoolVar(&enableLabels, "enable-labels", false, "Enable label collection")
	generateCmd.Flags().BoolVar(&enableAnnotations, "enable-annotations", false, "Enable annotation collection")

	// Mark persistent flags as required on the generate command
	if err := rootCmd.MarkPersistentFlagRequired("service-name"); err != nil {
		panic(fmt.Sprintf("failed to mark service-name as required: %v", err))
	}
	if err := rootCmd.MarkPersistentFlagRequired("namespace"); err != nil {
		panic(fmt.Sprintf("failed to mark namespace as required: %v", err))
	}
	if err := rootCmd.MarkPersistentFlagRequired("secret-name"); err != nil {
		panic(fmt.Sprintf("failed to mark secret-name as required: %v", err))
	}
	if err := rootCmd.MarkPersistentFlagRequired("webhook-name"); err != nil {
		panic(fmt.Sprintf("failed to mark webhook-name as required: %v", err))
	}

	// Add subcommands
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(patchCmd)
}
