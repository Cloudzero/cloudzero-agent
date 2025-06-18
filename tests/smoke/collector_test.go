// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package smoke

import (
	"testing"
	"time"

	config "github.com/cloudzero/cloudzero-agent/app/config/gator"
	"github.com/cloudzero/cloudzero-agent/tests/utils"
	"github.com/stretchr/testify/require"
)

func TestSmoke_Collector_Runs(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	runTest(t, func(t *testContext) {
		// start the collector
		collector := t.StartCollector()
		require.NotNil(t, collector, "collector is null")

		// wait for the log message
		err := utils.ContainerWaitForLog(t.ctx, &utils.WaitForLogInput{
			Container: collector,
			Log:       "Starting service",
		})
		require.NoError(t, err, "failed to find log message")
	})
}

func TestSmoke_Controller_FileRotate(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	runTest(t, func(t *testContext) {
		// start the collector
		collector := t.StartCollector()
		require.NotNil(t, collector, "collector is null")

		// start the controller
		controller := t.StartController(controllerArgs{
			hours:   4,
			nodes:   3,
			pods:    5,
			cpu:     96,
			mem:     (1 << 30) * 128,
			batches: 3,
			chunks:  20_000,
		})
		require.NotNil(t, controller, "controller is null")

		// wait for the log message
		err := utils.ContainerWaitForLog(t.ctx, &utils.WaitForLogInput{
			Container: controller,
			Log:       "Successfully ran the mock controller",
		})
		require.NoError(t, err, "failed to find log message")

		// shutdown the collector to force flush logs
		duration := time.Second * 10
		err = (*collector).Stop(t.Context(), &duration)
		require.NoError(t, err, "failed to stop the collector")
	}, withConfigOverride(func(settings *config.Settings) {
		settings.Cloudzero.SendInterval = time.Second * 10
		settings.Cloudzero.UseHTTP = true
		settings.Cloudzero.SendTimeout = time.Second * 30
		settings.Database.CostMaxInterval = time.Second * 10
		settings.Database.ObservabilityMaxInterval = time.Second * 10
		settings.Database.MaxRecords = 10_000 // small record count to encourage file rotation
	}))
}
