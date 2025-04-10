// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudzero/cloudzero-agent/app/domain/shipper"
	"github.com/stretchr/testify/require"
)

func TestUnit_Shipper_ShipperID_Normal(t *testing.T) {
	tmpDir := getTmpDir(t)
	mockLister := &MockAppendableFiles{baseDir: tmpDir}
	settings := getMockSettings("", tmpDir)
	metricShipper, err := shipper.NewMetricShipper(context.Background(), settings, mockLister)
	require.NoError(t, err, "failed to create the metric shipper")

	id, err := metricShipper.GetShipperID()
	require.NoError(t, err, "failed to get the shipperId")
	require.NotEmpty(t, id, "invalid id")

	id2, err := metricShipper.GetShipperID()
	require.NoError(t, err, "failed to get the shipperId the second time")
	require.Equal(t, id, id2, "the second call to GetShipperId returned a different value")
}

func TestUnit_Shipper_ShipperID_FromFile(t *testing.T) {
	tmpDir := getTmpDir(t)
	mockLister := &MockAppendableFiles{baseDir: tmpDir}
	settings := getMockSettings("", tmpDir)
	metricShipper, err := shipper.NewMetricShipper(context.Background(), settings, mockLister)
	require.NoError(t, err, "failed to create the metric shipper")

	expected := "shipper-id"

	// write a file
	err = os.WriteFile(filepath.Join(metricShipper.GetBaseDir(), ".shipperid"), []byte(expected), 0o755)
	require.NoError(t, err, "failed to create the shipperid file")

	// get the shipperid
	id, err := metricShipper.GetShipperID()
	require.NoError(t, err, "failed to get the shipper id")

	require.Equal(t, expected, id)
}

func TestUnit_Shipper_ShipperID_FromEnv(t *testing.T) {
	tmpDir := getTmpDir(t)
	mockLister := &MockAppendableFiles{baseDir: tmpDir}
	settings := getMockSettings("", tmpDir)
	metricShipper, err := shipper.NewMetricShipper(context.Background(), settings, mockLister)
	require.NoError(t, err, "failed to create the metric shipper")

	expected := "shipper-id"

	// set the env variable
	os.Setenv("HOSTNAME", expected)

	// get the shipper id
	id, err := metricShipper.GetShipperID()
	require.NoError(t, err, "failed to get the shipper id")

	require.Equal(t, expected, id)
}
