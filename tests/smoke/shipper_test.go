package smoke_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/cloudzero/cloudzero-insights-controller/app/config"
	"github.com/stretchr/testify/require"
)

func TestSmoke_Shipper_Runs(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	runTest(t, func(t *testContext) {
		// start the shipper
		shipper := t.StartShipper()
		require.NotNil(t, shipper, "shipper is null")

		// write files to the data directory
		numMetricFiles := 10
		t.WriteTestMetrics(numMetricFiles, 100)

		// wait for the log message
		err := t.WaitForLog(shipper, fmt.Sprintf("\"numNewFiles\":%d", numMetricFiles))
		require.NoError(t, err, "failed to find log message")
	}, withConfigOverride(func(settings *config.Settings) {
		settings.Cloudzero.SendInterval = time.Second * 10
	}))
}
