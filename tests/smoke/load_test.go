package smoke

import (
	"path/filepath"
	"testing"
	"time"

	config "github.com/cloudzero/cloudzero-agent/app/config/gator"
	"github.com/cloudzero/cloudzero-agent/tests/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_Shipper(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	runTest(t, func(t *testContext) {
		// write files to the data directory
		numMetricFiles := 3000
		t.WriteTestMetrics(numMetricFiles, 100)

		// start the mock remote write
		remotewrite := t.StartMockRemoteWrite()
		require.NotNil(t, remotewrite, "remotewrite is null")

		// start the shipper
		shipper := t.StartShipper()
		require.NotNil(t, shipper, "shipper is null")

		// wait for the log message
		err := utils.ContainerWaitForLog(t.ctx, &utils.WaitForLogInput{
			Container: t.shipper,
			Log:       "Successfully ran the shipper cycle",
			Timeout:   time.Duration(5) * time.Minute,
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
	},
		withConfigOverride(func(settings *config.Settings) {
			settings.Cloudzero.SendInterval = time.Second * 10
			settings.Cloudzero.UseHTTP = true
			settings.Cloudzero.SendTimeout = time.Minute * 5
		}),
		withUploadDelayMs("0"),
	)
}

func TestLoad_ClientApplication(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	runTest(t, func(t *testContext) {
		// start the remote write
		remotewrite := t.StartMockRemoteWrite()
		require.NotNil(t, remotewrite, "remotewrite is null")

		// start the collector
		collector := t.StartCollector()
		require.NotNil(t, collector, "collector is null")

		// start the collector
		controller := t.StartController(controllerArgs{
			hours:   8,
			nodes:   7,
			pods:    20,
			cpu:     96,
			mem:     (1 << 30) * 128,
			batches: 1,
			chunks:  20_000,
		})
		require.NotNil(t, controller, "controller is null")

		// wait for the log message
		err := utils.ContainerWaitForLog(t.ctx, &utils.WaitForLogInput{
			Container: controller,
			Log:       "Successfully ran the mock controller",
			Timeout:   time.Minute * 5,
		})
		require.NoError(t, err, "failed to find log message")

		// shutdown the collector to force flush to disk
		duration := time.Second * 10
		err = (*collector).Stop(t.Context(), &duration)
		require.NoError(t, err, "failed to stop the collector")

		// start the shipper
		shipper := t.StartShipper()
		require.NotNil(t, shipper, "shipper is null")

		// wait for the shipper to send files
		err = utils.ContainerWaitForLog(t.ctx, &utils.WaitForLogInput{
			Container: shipper,
			Log:       "Successfully uploaded new files",
			Timeout:   time.Minute * 5,
		})
		require.NoError(t, err, "failed to find log message waiting for the shipper")

		// ensure there are no new files
		newFiles, err := filepath.Glob(filepath.Join(t.dataLocation, "*_*_*.json.br"))
		require.NoError(t, err, "failed to read the root directory")
		require.Empty(t, newFiles, "found new files")

		uploaded, err := filepath.Glob(filepath.Join(t.dataLocation, "uploaded", "*_*_*.json.br"))
		assert.NoError(t, err, "failed to read the uploaded directory")
		assert.NotEmpty(t, uploaded, "there were no uploaded files")

		// ensure the number of files in the minio client are equal
		response := t.QueryMinio()
		assert.NotEmpty(t, response.Objects)
		assert.Equal(t, len(uploaded), len(response.Objects))
	}, withConfigOverride(func(settings *config.Settings) {
		settings.Cloudzero.SendInterval = time.Second * 10
		settings.Cloudzero.UseHTTP = true
		settings.Cloudzero.SendTimeout = time.Second * 30
		settings.Database.CostMaxInterval = time.Second * 10
		settings.Database.ObservabilityMaxInterval = time.Second * 10
	}))
}
