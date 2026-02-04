// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"strings"
)

const (
	DiagnosticAPIKey            string = "api_key_valid" //nolint:gosec // false positive for G101: Potential hardcoded credentials
	DiagnosticK8sVersion        string = "k8s_version"
	DiagnosticK8sNamespace      string = "k8s_namespace"
	DiagnosticK8sProvider       string = "k8s_provider"
	DiagnosticKMS               string = "kube_state_metrics_reachable"
	DiagnosticPrometheusVersion string = "prometheus_version"
	DiagnosticScrapeConfig      string = "scrape_cfg"
	DiagnosticInsightsIngress   string = "webhook_server_reachable"
	DiagnosticAgentSettings     string = "agent_settings"
	DiagnosticIstioXClusterLB   string = "istio_xcluster_lb"
)

const (
	DiagnosticInternalInitStart  string = "init_start"
	DiagnosticInternalInitStop   string = "init_ok"
	DiagnosticInternalInitFailed string = "init_failed"
	DiagnosticInternalPodStart   string = "pod_start"
	DiagnosticInternalPodStop    string = "pod_stop"
	DiagnosticInternalConfigLoad string = "config_load"
)

// CheckType represents the severity/handling of a diagnostic check
type CheckType string

const (
	// CheckTypeRequired causes the validator to exit with error on failure
	CheckTypeRequired CheckType = "required"
	// CheckTypeOptional emits warnings on failure but doesn't affect exit code
	CheckTypeOptional CheckType = "optional"
	// CheckTypeInformative is for information gathering only - always reports passing
	CheckTypeInformative CheckType = "informative"
)

func IsValidCheckType(t string) bool {
	switch CheckType(strings.ToLower(t)) {
	case CheckTypeRequired, CheckTypeOptional, CheckTypeInformative:
		return true
	}
	return false
}

func IsValidDiagnostic(d string) bool {
	d = strings.ToLower(strings.TrimSpace(d))
	switch d {
	case DiagnosticAPIKey, DiagnosticK8sVersion,
		DiagnosticK8sNamespace, DiagnosticK8sProvider,
		DiagnosticKMS, DiagnosticScrapeConfig,
		DiagnosticPrometheusVersion, DiagnosticInsightsIngress,
		DiagnosticAgentSettings, DiagnosticIstioXClusterLB:
		return true
	}
	return false
}

type Diagnostics struct {
	Stages []Stage `yaml:"stages"`
}

func (s *Diagnostics) Validate() error {
	for _, stage := range s.Stages {
		if err := stage.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// CheckConfig defines a single check with its type (as parsed from ConfigMap)
type CheckConfig struct {
	Name string    `yaml:"name"`
	Type CheckType `yaml:"type" default:"optional"`
}

func (c *CheckConfig) Validate() error {
	c.Name = strings.ToLower(strings.TrimSpace(c.Name))
	if !IsValidDiagnostic(c.Name) {
		return fmt.Errorf("unknown diagnostic check: %s", c.Name)
	}
	c.Type = CheckType(strings.ToLower(strings.TrimSpace(string(c.Type))))
	if c.Type == "" {
		c.Type = CheckTypeOptional
	}
	if !IsValidCheckType(string(c.Type)) {
		return fmt.Errorf("invalid check type for %s: %s", c.Name, c.Type)
	}
	return nil
}

type Stage struct {
	Name   string        `yaml:"name"`
	Checks []CheckConfig `yaml:"checks"`
}

func (s *Stage) Validate() error {
	s.Name = strings.ToLower(strings.TrimSpace(s.Name))
	if !IsValidStage(s.Name) {
		return fmt.Errorf("invalid stage: %s", s.Name)
	}

	for i := range s.Checks {
		if err := s.Checks[i].Validate(); err != nil {
			return err
		}
	}
	return nil
}
