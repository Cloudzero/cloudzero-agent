// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper_test

import (
	"context"
	"os"
	"testing"

	"github.com/cloudzero/cloudzero-insights-controller/app/domain/shipper"
	"github.com/stretchr/testify/require"
)

func TestShipper_Integration_InvalidApiKey(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// get a tmp dir
	tmpDir := t.TempDir()

	// create the metricShipper
	settings := getMockSettingsIntegration(t, tmpDir, "invalid-api-key")
	metricShipper, err := shipper.NewMetricShipper(context.Background(), settings, nil)
	require.NoError(t, err)

	// create test files
	files := createTestFiles(t, tmpDir, 5)

	_, err = metricShipper.AllocatePresignedURLs(files)
	require.Error(t, err)
	require.Equal(t, shipper.ErrUnauthorized, err)
}

func TestShipper_Integration_AllocatePresignedURL(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// setup env
	apiKey, exists := os.LookupEnv("CLOUDZERO_DEV_API_KEY")
	require.True(t, exists)
	tmpDir := t.TempDir()

	// create the metricShipper
	settings := getMockSettingsIntegration(t, tmpDir, apiKey)
	metricShipper, err := shipper.NewMetricShipper(context.Background(), settings, nil)
	require.NoError(t, err)

	// create some test files to simulate resource tracking
	files := createTestFiles(t, tmpDir, 5)

	// get the presigned URLs
	files2, err := metricShipper.AllocatePresignedURLs(files)
	require.NoError(t, err)

	// validate the pre-signed urls exist
	require.Equal(t, len(files), len(files2))
	for _, file := range files2 {
		require.NotEmpty(t, file.PresignedURL)
	}
}

func TestShipper_Integration_UploadToS3(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// setup env
	apiKey, exists := os.LookupEnv("CLOUDZERO_DEV_API_KEY")
	require.True(t, exists)
	tmpDir := t.TempDir()

	// create the metricShipper
	settings := getMockSettingsIntegration(t, tmpDir, apiKey)
	metricShipper, err := shipper.NewMetricShipper(context.Background(), settings, nil)
	require.NoError(t, err)

	// create some test files to simulate resource tracking
	files := createTestFiles(t, tmpDir, 2)

	// get the presigned URLs
	files2, err := metricShipper.AllocatePresignedURLs(files)
	require.NoError(t, err)

	// upload to s3
	for _, file := range files2 {
		err = metricShipper.Upload(file)
		require.NoError(t, err)
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

	// create the metricShipper
	settings := getMockSettingsIntegration(t, tmpDir, apiKey)
	metricShipper, err := shipper.NewMetricShipper(context.Background(), settings, nil)
	require.NoError(t, err)

	// create some test files to simulate resource tracking
	files := createTestFiles(t, tmpDir, 5)

	// get the presigned URLs
	files2, err := metricShipper.AllocatePresignedURLs(files)
	require.NoError(t, err)

	// get the ref ids
	refIDs := make([]string, len(files2))
	for i, file := range files2 {
		refIDs[i] = file.ReferenceID
	}

	// abandon these files
	err = metricShipper.AbandonFiles(refIDs, "integration-test-abandon")
	require.NoError(t, err)
}
