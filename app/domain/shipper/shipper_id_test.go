// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper_test

import (
	"context"
	"testing"

	"github.com/cloudzero/cloudzero-insights-controller/app/domain/shipper"
	"github.com/stretchr/testify/require"
)

func TestUnit_Shipper_ShipperID(t *testing.T) {
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
