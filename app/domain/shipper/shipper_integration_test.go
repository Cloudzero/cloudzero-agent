//go:build integration
// +build integration

// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudzero/cloudzero-insights-controller/app/config"
	"github.com/stretchr/testify/require"
)

func TestShipper_Integration_InvalidApiKey(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// get a tmp dir
	tmpDir := t.TempDir()

	// create the shipper
	settings := setupSettingsIntegration(t, tmpDir, "invalid-api-key")
	shipper, err := NewMetricShipper(context.Background(), settings, nil)
	require.NoError(t, err)

	// create test files
	files := createTestFiles(t, tmpDir, 5)

	_, err = shipper.AllocatePresignedURLs(files)
	require.Error(t, err)
	require.Equal(t, ErrUnauthorized, err)
}

func TestShipper_Integration_AllocatePresignedURL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// setup env
	apiKey, exists := os.LookupEnv("CLOUDZERO_DEV_API_KEY")
	require.True(t, exists)
	tmpDir := t.TempDir()

	// create the shipper
	settings := setupSettingsIntegration(t, tmpDir, apiKey)
	shipper, err := NewMetricShipper(context.Background(), settings, nil)
	require.NoError(t, err)

	// create some test files to simulate resource tracking
	files := createTestFiles(t, tmpDir, 5)

	// get the presigned URLs
	files2, err := shipper.AllocatePresignedURLs(files)
	require.NoError(t, err)

	// validate the pre-signed urls exist
	require.Equal(t, len(files), len(files2))
	for _, file := range files2 {
		require.NotEmpty(t, file.PresignedURL)
	}
}

func TestShipper_Integration_ExpiredPresignedURL(t *testing.T) {}

func TestShipper_Integration_ReplayRequest(t *testing.T) {}

func TestShipper_Integration_AbandonFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// setup env
	apiKey, exists := os.LookupEnv("CLOUDZERO_DEV_API_KEY")
	require.True(t, exists)
	tmpDir := t.TempDir()

	// create the shipper
	settings := setupSettingsIntegration(t, tmpDir, apiKey)
	shipper, err := NewMetricShipper(context.Background(), settings, nil)
	require.NoError(t, err)

	// create some test files to simulate resource tracking
	files := createTestFiles(t, tmpDir, 5)

	// get the presigned URLs
	files2, err := shipper.AllocatePresignedURLs(files)
	require.NoError(t, err)

	// get the ref ids
	refIDs := make([]string, len(files2))
	for i, file := range files2 {
		refIDs[i] = file.ReferenceID
	}

	// abandon these files
	err = shipper.AbandonFiles(refIDs, "integration-test-abandon")
	require.NoError(t, err)
}

func setupSettingsIntegration(t *testing.T, dir, apiKey string) *config.Settings {
	// tmp file to write api key
	filePath := filepath.Join(dir, ".cz-api-key")
	err := os.WriteFile(filePath, []byte(apiKey), 0o644)
	require.NoError(t, err)

	// get the endpoint
	apiHost, exists := os.LookupEnv("CLOUDZERO_HOST")
	require.True(t, exists)

	// create the config
	cfg := &config.Settings{
		ClusterName:    "test-cluster",
		CloudAccountID: "test-account",
		Region:         "us-east-1",
		Cloudzero: config.Cloudzero{
			Host:        apiHost,
			SendTimeout: time.Second * 30,
			APIKeyPath:  filePath,
		},
		Database: config.Database{
			StoragePath: "/tmp/storage",
		},
	}

	// validate the config
	err = cfg.SetAPIKey()
	require.NoError(t, err)
	err = cfg.SetRemoteUploadAPI()
	require.NoError(t, err)

	return cfg
}
