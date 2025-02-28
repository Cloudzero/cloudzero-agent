package smoke_test

import (
	"testing"

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
		err := t.WaitForLog(collector, "Starting service")
		require.NoError(t, err, "failed to find log message")
	})
}
