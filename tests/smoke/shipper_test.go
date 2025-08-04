// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package smoke

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	config "github.com/cloudzero/cloudzero-agent/app/config/gator"
	"github.com/cloudzero/cloudzero-agent/tests/utils"
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
		// (divide by 2 since t.WriteTestMetrics creates half metric files half observability files)
		require.GreaterOrEqual(t, len(uploaded), int(float64(numMetricFiles/2)*0.8))
	}, withConfigOverride(func(settings *config.Settings) {
		settings.Cloudzero.SendInterval = time.Second * 10
		settings.Cloudzero.UseHTTP = true
	}))
}

// When an upload error occurs, the shipper should not exit out of the loop
// The correct behavior is to loop infinitely
func TestSmoke_Shipper_Upload_Error(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	runTest(t, func(t *testContext) {
		// write files to the data directory
		numMetricFiles := 1
		t.WriteTestMetrics(numMetricFiles, 100)

		// start the mock remote write
		t.errorOnUpload = "true"
		remotewrite := t.StartMockRemoteWrite()
		require.NotNil(t, remotewrite, "remotewrite is null")

		// start the shipper
		shipper := t.StartShipper()
		require.NotNil(t, shipper, "shipper is null")

		// wait for the log message
		// with a timeout of 120 seconds and an interval of 30, we should see at least 3 instances of this message
		// 3 instances of this message means that the shipper loop successfully stays active with upload errors
		err := utils.ContainerWaitForLog(t.ctx, &utils.WaitForLogInput{
			Container:  shipper,
			Log:        "failed to ship the metrics",
			Timeout:    time.Second * 120,
			AllowError: true,
			N:          3,
		})
		require.NoError(t, err, "failed to find log message")
	}, withConfigOverride(func(settings *config.Settings) {
		settings.Cloudzero.SendInterval = time.Second * 30
		settings.Cloudzero.UseHTTP = true
	}))
}

func TestSmoke_Shipper_ReplayRequest_OK(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	runTest(t, func(t *testContext) {
		// write files to the data directory
		numMetricFiles := 1
		filenames := t.WriteTestMetrics(numMetricFiles, 100)

		// create a mock payload for the remotewrite to send
		payload := make([]map[string]string, 0)
		payload = append(payload, map[string]string{
			"ref_id": filenames[0],
			"url":    "",
		})
		enc, err := json.Marshal(payload)
		require.NoError(t, err, "failed to encode mock payload")

		// start the mock remote write
		t.replayRequestPayload = string(enc)
		remotewrite := t.StartMockRemoteWrite()
		require.NotNil(t, remotewrite, "remotewrite is null")

		// start the shipper
		shipper := t.StartShipper()
		require.NotNil(t, shipper, "shipper is null")

		// wait for shipper to finish
		err = utils.ContainerWaitForLog(t.ctx, &utils.WaitForLogInput{
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

// Shippers should be able to upload files while NOT getting a valid replay request payload
func TestSmoke_Shipper_ReplayRequest_Invalid_Payload(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	runTest(t, func(t *testContext) {
		// write files to the data directory
		numMetricFiles := 10
		t.WriteTestMetrics(numMetricFiles, 100)

		// start the mock remote write
		t.replayRequestPayload = "This is not a valid payload for a remote write request"
		remotewrite := t.StartMockRemoteWrite()
		require.NotNil(t, remotewrite, "remotewrite is null")

		// start the shipper
		shipper := t.StartShipper()
		require.NotNil(t, shipper, "shipper is null")

		// wait for the log message
		err := utils.ContainerWaitForLog(t.ctx, &utils.WaitForLogInput{
			Container:  shipper,
			Log:        "Successfully ran the shipper cycle",
			AllowError: true,
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
		// (divide by 2 since t.WriteTestMetrics creates half metric files half observability files)
		require.GreaterOrEqual(t, len(uploaded), int(float64(numMetricFiles/2)*0.8))
	}, withConfigOverride(func(settings *config.Settings) {
		settings.Cloudzero.SendInterval = time.Second * 10
		settings.Cloudzero.UseHTTP = true
	}))
}

// Ensure the shipper can run multiple cycles with no locks
func TestSmoke_Shipper_NoDeadLock(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	runTest(t, func(t *testContext) {
		// start the shipper
		shipper := t.StartShipper()
		require.NotNil(t, shipper, "shipper is null")

		// wait for the log message
		err := utils.ContainerWaitForLog(t.ctx, &utils.WaitForLogInput{
			Container:  shipper,
			Log:        "Successfully ran the shipper cycle",
			AllowError: true,
			N:          3,
			Timeout:    time.Minute * 2,
		})
		require.NoError(t, err, "failed to find log message")
	}, withConfigOverride(func(settings *config.Settings) {
		settings.Cloudzero.SendInterval = time.Second * 10
		settings.Cloudzero.UseHTTP = true
	}))
}

// The shipper should be able to handle context deadline timeouts
func TestSmoke_Shipper_RemoteWrite_HTTPTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	runTest(t, func(t *testContext) {
		// write files to the data directory
		numMetricFiles := 10
		t.WriteTestMetrics(numMetricFiles, 100)

		// start the mock remote write
		t.uploadDelayMs = "5000" // 5seconds
		remotewrite := t.StartMockRemoteWrite()
		require.NotNil(t, remotewrite, "remotewrite is null")

		// start the shipper
		shipper := t.StartShipper()
		require.NotNil(t, shipper, "shipper is null")

		// wait for the log message
		err := utils.ContainerWaitForLog(t.ctx, &utils.WaitForLogInput{
			Container:  shipper,
			Log:        "context deadline exceeded",
			AllowError: true,
			N:          3,
			Timeout:    time.Minute * 2,
		})
		require.NoError(t, err, "failed to find log message")
	}, withConfigOverride(func(settings *config.Settings) {
		settings.Cloudzero.SendInterval = time.Second * 10
		settings.Cloudzero.SendTimeout = time.Second // set duration very low to timeout http requests
		settings.Cloudzero.UseHTTP = true
	}))
}

// This test attempts to simulate delay in requesting presigned urls, then delay
// in uploading files. But, with an http timeout context deadline that is MORE
// than a single reuquest operation, but LESS than the 2. Such that, if each timeout / context
// deadline is truly independent of each other, then the context will be reset. But if there
// are instances in which contexts are unknowingly shared, then this will cause a context deadline
// exceeded which we DO NOT WANT.
func TestSmoke_Shipper_RemoteWrite_MutliTimeout(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
}
