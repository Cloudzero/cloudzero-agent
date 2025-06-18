// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package smoke

import (
	"fmt"
	"testing"

	"github.com/cloudzero/cloudzero-agent/tests/utils"
	"github.com/stretchr/testify/require"
)

func TestSmoke_RemoteWrite_Runs(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	runTest(t, func(t *testContext) {
		// start the collector
		remotewrite := t.StartMockRemoteWrite()
		require.NotNil(t, remotewrite, "remotewrite is null")

		// wait for the log message
		err := utils.ContainerWaitForLog(t.ctx, &utils.WaitForLogInput{
			Container: remotewrite,
			Log:       fmt.Sprintf("Mock remotewrite is listening on: 'localhost:%s", t.remoteWritePort),
		})
		require.NoError(t, err, "failed to find log message")
	})
}
