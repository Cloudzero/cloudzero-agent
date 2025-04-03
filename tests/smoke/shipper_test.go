// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package smoke

import (
	"path/filepath"
	"testing"
	"time"

	config "github.com/cloudzero/cloudzero-agent-validator/app/config/gator"
	"github.com/cloudzero/cloudzero-agent-validator/tests/utils"
	"github.com/stretchr/testify/require"
)

func TestSmoke_Shipper_WithRemoteLambdaAPI(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	runTest(t, func(t *testContext) {
		// write files to the data directory
		numMetricFiles := 10
		t.WriteTestMetrics(numMetricFiles, 100)

		// start the shipper
		shipper := t.StartShipper()
		require.NotNil(t, shipper, "shipper is null")

		// wait for the log message
		err := utils.ContainerWaitForLog(t.ctx, &utils.WaitForLogInput{
			Container: shipper,
			Log:       "Successfully ran the shipper cycle",
		})
		require.NoError(t, err, "failed to find log message")
	}, withConfigOverride(func(settings *config.Settings) {
		settings.Cloudzero.SendInterval = time.Second * 10
	}))
}

func TestSmoke_Shipper_WithMockRemoteWrite(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	runTest(t, func(t *testContext) {
		// write files to the data directory
		numMetricFiles := 10
		t.WriteTestMetrics(numMetricFiles, 100)

		// start the mock remote write
		remotewrite := t.StartMockRemoteWrite()
		require.NotNil(t, remotewrite, "remotewrite is null")

		// start the shipper
		shipper := t.StartShipper()
		require.NotNil(t, shipper, "shipper is null")

		// wait for the log message
		err := utils.ContainerWaitForLog(t.ctx, &utils.WaitForLogInput{
			Container: shipper,
			Log:       "Successfully ran the shipper cycle",
		})
		require.NoError(t, err, "failed to find log message")

		// ensure that the minio client has the correct files
		response := t.QueryMinio()
		require.NotEmpty(t, response.Objects)
		require.Equal(t, numMetricFiles, response.Length)

		// validate the filesystem has the correct files as well
		newFiles, err := filepath.Glob(filepath.Join(t.dataLocation, "*_*_*.json.br"))
		require.NoError(t, err, "failed to read the root directory")
		require.Empty(t, newFiles, "root directory is not empty") // ensure all files were uploaded

		uploaded, err := filepath.Glob(filepath.Join(t.dataLocation, "uploaded", "*_*_*.json.br"))
		require.NoError(t, err, "failed to read the uploaded directory")
		// ensure all files were uploaded, but account for the shipper purging up to 20% of the files
		require.GreaterOrEqual(t, len(uploaded), int(float64(numMetricFiles)*0.8))
	}, withConfigOverride(func(settings *config.Settings) {
		settings.Cloudzero.SendInterval = time.Second * 10
		settings.Cloudzero.UseHTTP = true
	}))
}
