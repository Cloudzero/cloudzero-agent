// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package smoke

import (
	"testing"

	"github.com/cloudzero/cloudzero-agent-validator/tests/utils"
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
