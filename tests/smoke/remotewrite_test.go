package smoke_test

import (
	"fmt"
	"testing"

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
		err := t.WaitForLog(remotewrite, fmt.Sprintf("Server is running on :%s", t.remoteWritePort))
		require.NoError(t, err, "failed to find log message")
	})
}
