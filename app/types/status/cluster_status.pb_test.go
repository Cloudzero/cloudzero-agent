// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package status_test

import (
	"testing"

	"github.com/cloudzero/cloudzero-agent/app/types/status"
	"github.com/stretchr/testify/assert"
)

const (
	Name             = "test-name"
	AccountID        = "test-account"
	Region           = "test-region"
	ChartVersion     = "test-chart-version"
	AgentVersion     = "test-agent-version"
	ScrapeConfig     = "test-scrape-config"
	ValidatorVersion = "test-validator-version"
	K8SVersion       = "test-k8s-version"
)

func TestClusterStatus(t *testing.T) {
	// Create a new ClusterStatus instance
	cs := &status.ClusterStatus{
		Account:          AccountID,
		Region:           Region,
		Name:             Name,
		State:            status.StatusType_STATUS_TYPE_INIT_STARTED,
		ChartVersion:     ChartVersion,
		AgentVersion:     AgentVersion,
		ScrapeConfig:     ScrapeConfig,
		ValidatorVersion: ValidatorVersion,
		K8SVersion:       K8SVersion,
		Checks: []*status.StatusCheck{
			{
				Name:    "test-check-1",
				Passing: true,
				Error:   "",
			},
			{
				Name:    "test-check-2",
				Passing: false,
				Error:   "test-error",
			},
		},
	}

	// Test the getters
	assert.Equal(t, AccountID, cs.GetAccount())
	assert.Equal(t, Region, cs.GetRegion())
	assert.Equal(t, Name, cs.GetName())
	assert.Equal(t, status.StatusType_STATUS_TYPE_INIT_STARTED, cs.GetState())
	assert.Equal(t, ChartVersion, cs.GetChartVersion())
	assert.Equal(t, AgentVersion, cs.GetAgentVersion())
	assert.Equal(t, ScrapeConfig, cs.GetScrapeConfig())
	assert.Equal(t, ValidatorVersion, cs.GetValidatorVersion())
	assert.Equal(t, K8SVersion, cs.GetK8SVersion())
	assert.Equal(t, 2, len(cs.GetChecks()))

	// Test the first check
	check1 := cs.GetChecks()[0]
	assert.Equal(t, "test-check-1", check1.GetName())
	assert.True(t, check1.GetPassing())
	assert.Equal(t, "", check1.GetError())

	// Test the second check
	check2 := cs.GetChecks()[1]
	assert.Equal(t, "test-check-2", check2.GetName())
	assert.False(t, check2.GetPassing())
	assert.Equal(t, "test-error", check2.GetError())
}

func TestStatusCheck_Reset(t *testing.T) {
	statusCheck := &status.StatusCheck{
		Name:    Name,
		Passing: true,
		Error:   "error",
	}

	statusCheck.Reset()

	if statusCheck.Name != "" {
		t.Errorf("Expected Name to be empty, got %s", statusCheck.Name)
	}

	if statusCheck.Passing != false {
		t.Errorf("Expected Passing to be false, got %t", statusCheck.Passing)
	}

	if statusCheck.Error != "" {
		t.Errorf("Expected Error to be empty, got %s", statusCheck.Error)
	}
}

func TestClusterStatus_Reset(t *testing.T) {
	clusterStatus := &status.ClusterStatus{
		Account:          Name,
		Region:           Name,
		Name:             Name,
		State:            status.StatusType_STATUS_TYPE_INIT_STARTED,
		ChartVersion:     Name,
		AgentVersion:     Name,
		ScrapeConfig:     Name,
		ValidatorVersion: Name,
		K8SVersion:       Name,
		Checks: []*status.StatusCheck{
			{
				Name:    Name,
				Passing: true,
				Error:   "error",
			},
		},
	}

	clusterStatus.Reset()

	if clusterStatus.Account != "" {
		t.Errorf("Expected Account to be empty, got %s", clusterStatus.Account)
	}

	if clusterStatus.Region != "" {
		t.Errorf("Expected Region to be empty, got %s", clusterStatus.Region)
	}

	if clusterStatus.Name != "" {
		t.Errorf("Expected Name to be empty, got %s", clusterStatus.Name)
	}

	if clusterStatus.State != status.StatusType_STATUS_TYPE_UNSPECIFIED {
		t.Errorf("Expected State to be STATUS_TYPE_UNSPECIFIED, got %v", clusterStatus.State)
	}
}
