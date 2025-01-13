// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cloudzero/cloudzero-agent-validator/pkg/config"
)

const (
	chartVersion = "0.1.0"
	agentVersion = "0.2.0"

	apiHost = "https://api.cloudzero.com"
	apiKey  = "my-cloudzero-token"

	kmsServiceEndpoint = "http://kube-state-metrics:8080"
)

func TestSettings_NewSettings(t *testing.T) {
	cwd, err := os.Getwd()
	assert.NoError(t, err)

	configFilePath := cwd + "/testdata/cloudzero-agent-validator.yml"
	envFilePath := cwd + "/testdata/cloudzero-agent-validator.yml"

	settings, err := config.NewSettings(envFilePath, configFilePath)
	assert.NoError(t, err)
	assert.NotNil(t, settings)

	// verify Logging
	assert.Equal(t, "debug", settings.Logging.Level)
	assert.Equal(t, "./cloudzero-agent-validator.log", settings.Logging.Location)

	// verify Deployment
	assert.Equal(t, accountID, settings.Deployment.AccountID)
	assert.Equal(t, clusterID, settings.Deployment.ClusterName)
	assert.Equal(t, region, settings.Deployment.Region)

	// verify Versions
	assert.Equal(t, chartVersion, settings.Versions.ChartVersion)
	assert.Equal(t, agentVersion, settings.Versions.AgentVersion)

	// verify Cloudzero
	assert.Equal(t, apiHost, settings.Cloudzero.Host)
	assert.Equal(t, "./api_key_file", settings.Cloudzero.CredentialsFile)

	// verify Prometheus
	assert.Equal(t, kmsServiceEndpoint, settings.Prometheus.KubeStateMetricsServiceEndpoint)
	assert.Equal(t, []string{"prometheus.yml"}, settings.Prometheus.Configurations)

	// verify Diagnostics
	assert.Len(t, settings.Diagnostics.Stages, 3)
	for _, stage := range []string{config.ContextStageInit, config.ContextStageStart, config.ContextStageStop} {
		var stageInfo *config.Stage
		for i := range settings.Diagnostics.Stages {
			if stage == settings.Diagnostics.Stages[i].Name {
				stageInfo = &settings.Diagnostics.Stages[i]
				break
			}
		}
		assert.NotNil(t, stageInfo, "stage %s not found", stage)

		assert.Equal(t, stage, stageInfo.Name)
		assert.NotEmpty(t, stageInfo.Checks)
		for _, check := range stageInfo.Checks {
			assert.True(t, config.IsValidDiagnostic(check))
		}
	}
}

func TestSettings_Validate(t *testing.T) {
	wd, err := os.Getwd()
	assert.NoError(t, err)

	// Helper values for path checking
	filesPath := wd + "/testdata"
	configFilePath := filesPath + "/cloudzero-agent-validator.yml"
	envFilePath := filesPath + "/cloudzero-agent-validator.yml"
	logfileLocation := filesPath + "/cloudzero-agent-validator.log"
	secretFilePath := filesPath + "/api_key_file"
	scrapeConfigFile := filesPath + "/prometheus.yml"

	settings, err := config.NewSettings(envFilePath, configFilePath)
	assert.NoError(t, err)
	assert.NotNil(t, settings)

	err = os.Chdir(filesPath)
	assert.NoError(t, err)
	defer func() { _ = os.Chdir(wd) }()

	// make a duplicate with a full path
	settings.Prometheus.Configurations = append(settings.Prometheus.Configurations, scrapeConfigFile)
	// Before running validation - change to the testdata directory
	assert.NoError(t, settings.Validate())

	// Now make sure the paths for logging, secrets, and scrape config are valid
	assert.Equal(t, logfileLocation, settings.Logging.Location)
	assert.Equal(t, secretFilePath, settings.Cloudzero.CredentialsFile)
	assert.Equal(t, []string{scrapeConfigFile}, settings.Prometheus.Configurations)

	// ALso make sure we have the secret
	assert.NotEmpty(t, settings.Cloudzero.Credential)
	assert.Equal(t, apiKey, settings.Cloudzero.Credential)
}
