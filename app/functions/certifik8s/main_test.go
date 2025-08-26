package main

import (
	"testing"
)

// Simple test to verify the main package compiles and basic functions work
func TestMain(t *testing.T) {
	// Test that the root command is properly configured
	if rootCmd.Use != "cloudzero-certifik8s" {
		t.Errorf("expected root command use to be 'cloudzero-certifik8s', got '%s'", rootCmd.Use)
	}

	// Test that subcommands are properly added
	if len(rootCmd.Commands()) != 3 {
		t.Errorf("expected 3 subcommands, got %d", len(rootCmd.Commands()))
	}

	// Test that generate command exists
	generateCmd := rootCmd.Commands()[0]
	if generateCmd.Use != "generate" {
		t.Errorf("expected first subcommand to be 'generate', got '%s'", generateCmd.Use)
	}

	// Test that validate command exists
	validateCmd := rootCmd.Commands()[1]
	if validateCmd.Use != "patch" {
		t.Errorf("expected second subcommand to be 'patch', got '%s'", validateCmd.Use)
	}

	// Test that patch command exists
	patchCmd := rootCmd.Commands()[2]
	if patchCmd.Use != "validate" {
		t.Errorf("expected third subcommand to be 'validate', got '%s'", patchCmd.Use)
	}
}
