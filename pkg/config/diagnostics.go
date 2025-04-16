// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"strings"
)

const (
	DiagnosticAPIKey            string = "api_key_valid" //nolint:gosec // false positive for G101: Potential hardcoded credentials
	DiagnosticK8sVersion        string = "k8s_version"
	DiagnosticKMS               string = "kube_state_metrics_reachable"
	DiagnosticPrometheusVersion string = "prometheus_version"
	DiagnosticScrapeConfig      string = "scrape_cfg"
)

const (
	DiagnosticInternalInitStart  string = "init_start"
	DiagnosticInternalInitStop   string = "init_ok"
	DiagnosticInternalInitFailed string = "init_failed"
	DiagnosticInternalPodStart   string = "pod_start"
	DiagnosticInternalPodStop    string = "pod_stop"
)

func IsValidDiagnostic(d string) bool {
	d = strings.ToLower(strings.TrimSpace(d))
	switch d {
	case DiagnosticAPIKey, DiagnosticK8sVersion,
		DiagnosticKMS, DiagnosticScrapeConfig,
		DiagnosticPrometheusVersion:
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

type Stage struct {
	Name    string   `yaml:"name"`
	Enforce bool     `yaml:"enforce" default:"false"`
	Checks  []string `yaml:"checks"`
}

func (s *Stage) Validate() error {
	s.Name = strings.ToLower(strings.TrimSpace(s.Name))
	if !IsValidStage(s.Name) {
		return fmt.Errorf("invalid stage: %s", s.Name)
	}

	for i, check := range s.Checks {
		check = strings.ToLower(strings.TrimSpace(check))
		if !IsValidDiagnostic(check) {
			return fmt.Errorf("unknown diagnostic check: %s", check)
		}
		s.Checks[i] = check
	}
	return nil
}
