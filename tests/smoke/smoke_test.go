package smoke_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/cloudzero/cloudzero-insights-controller/app/config"
	"github.com/cloudzero/cloudzero-insights-controller/app/types"
	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
	"gopkg.in/yaml.v2"
)

type stdoutLogConsumer struct{}

// Accept prints the log to stdout
func (lc *stdoutLogConsumer) Accept(l testcontainers.Log) {
	fmt.Print(string(l.Content))
}

type testContextOption func(t *testContext)

// gives access to a pointer of the settings to edit any before they are
// passed into the container as a file on disk
func withConfigOverride(override func(settings *config.Settings)) testContextOption {
	return func(t *testContext) {
		override(t.cfg)
	}
}

type testContext struct {
	*testing.T
	ctx context.Context
	mu  sync.Mutex

	// config
	cfg             *config.Settings
	remoteWritePort string // for mock remote write

	// directory information
	tmpDir       string // the root dir created with t.TempDir()
	apiKeyFile   string // location of the api key file
	configFile   string // location of the config file
	dataLocation string // location of actively running data for the collector/shipper

	// internal docker state
	network     *testcontainers.DockerNetwork
	collector   *testcontainers.Container
	shipper     *testcontainers.Container
	remotewrite *testcontainers.Container
}

func newTestContext(t *testing.T, opts ...testContextOption) *testContext {
	// create the temp dir structure
	tmpDir := t.TempDir()

	// create an api key file
	apiKey, exists := os.LookupEnv("CLOUDZERO_DEV_API_KEY")
	if !exists {
		apiKey = "ak-test"
	}

	remoteWritePort := "8080"
	remoteWriteEndpoint, exists := os.LookupEnv("CLOUDZERO_HOST")
	if !exists {
		remoteWriteEndpoint = "mock-host:8081"
	}

	// write the api key file
	apiKeyFile := filepath.Join(tmpDir, ".api-key")
	err := os.WriteFile(apiKeyFile, []byte(apiKey), 0o777)
	require.NoError(t, err, "failed to write the api key file")

	// create the shared data directory
	dataLocation, err := os.MkdirTemp(tmpDir, "data-*")
	require.NoError(t, err, "failed to create the data location")

	// create the object with a test config
	tx := &testContext{
		T:   t,
		ctx: context.Background(), // in go 1.24 use t.Context()
		cfg: &config.Settings{
			CloudAccountID: "test-account-id",
			Region:         "us-east-1",
			ClusterName:    "smoke-test-cluster",
			Logging:        config.Logging{Level: "debug"},
			Database:       config.Database{StoragePath: dataLocation},
			Cloudzero: config.Cloudzero{
				APIKeyPath:  apiKeyFile,
				Host:        remoteWriteEndpoint,
				SendTimeout: time.Second * 30,
				UseHttp:     true,
			},
		},
		remoteWritePort: remoteWritePort,
		tmpDir:          tmpDir,
		apiKeyFile:      apiKeyFile,
		dataLocation:    dataLocation,
	}

	// run the options
	for _, opt := range opts {
		opt(tx)
	}

	// marshal into yaml
	modifiedConfig, err := yaml.Marshal(&tx.cfg)
	require.NoError(t, err, "failed to marshal the config file")

	// write the config file
	configFile := filepath.Join(tmpDir, "config.yaml")
	err = os.WriteFile(configFile, modifiedConfig, 0o777)
	require.NoError(t, err, "failed to write the modified config file")
	require.NoError(t, err, "failed to read copied config file")

	// set on the context object
	tx.configFile = configFile

	return tx
}

// Sets the setting as modified by the function and writes the config file
func (t *testContext) SetSettings(f func(settings *config.Settings) error) {
	err := f(t.cfg)
	require.NoError(t, err, "failed to write the new config")

	// marshal into yaml
	modifiedConfig, err := yaml.Marshal(t.cfg)
	require.NoError(t, err, "failed to marshal the config file")

	// write the config file
	err = os.WriteFile(t.configFile, modifiedConfig, 0o777)
	require.NoError(t, err, "failed to write the modified config file")
}

// Wrap tests in this to inject `testContext` into them
func runTest(t *testing.T, test func(t *testContext), opts ...testContextOption) {
	tx := newTestContext(t, opts...)
	t.Cleanup(tx.Clean)
	defer tx.Clean()
	test(tx)
}

func (t *testContext) Clean() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.collector != nil {
		(*t.collector).Terminate(t.ctx)
	}
	if t.shipper != nil {
		(*t.shipper).Terminate(t.ctx)
	}
	if t.network != nil {
		t.network.Remove(t.ctx)
	}
}

// writes valid metric files to the shared data path `dataLocation`
func (t *testContext) WriteTestMetrics(numFiles int, numMetrics int) {
	for i := range numFiles {
		now := time.Now()

		// create a file location
		file, err := os.Create(filepath.Join(t.dataLocation, fmt.Sprintf("metrics_%d_%05d.json.br", now.UnixMilli(), i)))
		require.NoError(t, err, "failed to create file: %s", err)

		// create the metrics array
		metrics := make([]*types.Metric, numMetrics)
		for j := range numMetrics {
			metrics[j] = &types.Metric{
				ClusterName:    t.cfg.ClusterName,
				CloudAccountID: t.cfg.CloudAccountID,
				Year:           fmt.Sprintf("%04d", now.Year()),
				Month:          fmt.Sprintf("%02d", int(now.Month())),
				Day:            fmt.Sprintf("%02d", now.Day()),
				Hour:           fmt.Sprintf("%02d", now.Hour()),
				MetricName:     fmt.Sprintf("test-metric-%d", j),
				NodeName:       "test-node",
				CreatedAt:      time.Now().UnixMilli(),
				Value:          "I'm a value!",
				TimeStamp:      time.Now().UnixMilli(),
				Labels: map[string]string{
					"foo": "bar",
				},
			}
		}

		// compress the metrics
		jsonData, err := json.Marshal(metrics)
		require.NoError(t, err, "failed to encode the metrics as json")

		var compressedData bytes.Buffer
		func() {
			compressor := brotli.NewWriterLevel(&compressedData, 1)
			defer compressor.Close()

			_, err = compressor.Write(jsonData)
			require.NoError(t, err, "failed to write the json data through the brotli compressor")
		}()

		// write the data to the file
		_, err = file.Write(compressedData.Bytes())
		require.NoError(t, err, "failed to write the metrics to the file")
	}
}

func (t *testContext) WaitForCondition(timeout int, poll int, condition func() (bool, error)) error {
	ctx, cancel := context.WithTimeout(t.ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	ticker := time.NewTicker(time.Duration(poll) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return errors.New("timeout reached, condition not met")
		case <-ticker.C:
			passed, err := condition()
			if err != nil {
				return err
			}
			if passed {
				fmt.Println("Met condition")
				return nil
			}
		}
	}
}

// Polls the logs of the container to see if a `log` message exists. If the timeout is
// exceeded, an error returns. Polls every 1 second and waits for 30 seconds.
// If an error message is thrown, then this will fail
func (t *testContext) WaitForLog(container *testcontainers.Container, log string) error {
	if container == nil {
		return fmt.Errorf("container is nil")
	}

	return t.WaitForCondition(30, 1, func() (bool, error) {
		// read the logs
		reader, err := (*container).Logs(t.ctx)
		if err != nil {
			return false, err
		}
		data, err := io.ReadAll(reader)
		if err != nil {
			return false, err
		}

		if strings.Contains(strings.ToLower(string(data)), "error") {
			return false, fmt.Errorf("error message found")
		}

		if strings.Contains(strings.ToLower(string(data)), strings.ToLower(log)) {
			return true, nil
		}

		return false, nil
	})
}

func (t *testContext) CreateNetwork() *testcontainers.DockerNetwork {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.network == nil {
		network, err := network.New(
			t.ctx,
			network.WithAttachable(),
		)
		require.NoError(t, err, "failed to create network")
		t.network = network
	}

	return t.network
}

func (t *testContext) StartCollector() *testcontainers.Container {
	t.CreateNetwork()

	if t.collector == nil {
		fmt.Println("Building collector ...")

		collectorReq := testcontainers.ContainerRequest{
			FromDockerfile: testcontainers.FromDockerfile{
				Context:    "../..",
				Dockerfile: "tests/docker/Dockerfile.collector",
				KeepImage:  true,
			},
			Networks: []string{t.network.Name},
			HostConfigModifier: func(hc *container.HostConfig) {
				hc.Binds = append(hc.Binds, fmt.Sprintf("%s:%s", t.tmpDir, t.tmpDir)) // bind the tmp dir to the container
			},
			Entrypoint: []string{"/app/collector", "-config", t.configFile},
			Env:        map[string]string{},
			LogConsumerCfg: &testcontainers.LogConsumerConfig{
				Consumers: []testcontainers.LogConsumer{&stdoutLogConsumer{}},
			},
			WaitingFor: wait.ForLog("Starting service"),
		}

		collector, err := testcontainers.GenericContainer(t.ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: collectorReq,
			Started:          true,
		})
		require.NoError(t, err, "failed to create the collector")

		fmt.Println("Collector built successfully")
		t.collector = &collector
	}

	return t.collector
}

func (t *testContext) StartShipper() *testcontainers.Container {
	t.CreateNetwork()

	if t.shipper == nil {
		fmt.Println("Building shipper ...")

		shipperReq := testcontainers.ContainerRequest{
			FromDockerfile: testcontainers.FromDockerfile{
				Context:    "../..",
				Dockerfile: "tests/docker/Dockerfile.shipper",
				KeepImage:  true,
			},
			Networks: []string{t.network.Name},
			HostConfigModifier: func(hc *container.HostConfig) {
				hc.Binds = append(hc.Binds, fmt.Sprintf("%s:%s", t.tmpDir, t.tmpDir)) // bind the tmp dir to the container
			},
			Entrypoint: []string{"/app/shipper", "-config", t.configFile},
			Env:        map[string]string{},
			LogConsumerCfg: &testcontainers.LogConsumerConfig{
				Consumers: []testcontainers.LogConsumer{&stdoutLogConsumer{}},
			},
			WaitingFor: wait.ForLog("Shipper service starting"),
		}

		shipper, err := testcontainers.GenericContainer(t.ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: shipperReq,
			Started:          true,
		})
		require.NoError(t, err, "failed to create the shipper")

		fmt.Println("Shipper built successfully")
		t.shipper = &shipper
	}

	return t.shipper
}

func (t *testContext) StartMockRemoteWrite() *testcontainers.Container {
	t.CreateNetwork()

	if t.shipper == nil {
		fmt.Println("Building the mock remotewrite ...")

		remotewriteReq := testcontainers.ContainerRequest{
			FromDockerfile: testcontainers.FromDockerfile{
				Context:    "../..",
				Dockerfile: "tests/docker/Dockerfile.remotewrite",
				KeepImage:  true,
			},
			Networks: []string{t.network.Name},
			// HostConfigModifier: func(hc *container.HostConfig) {
			// 	hc.Binds = append(hc.Binds, fmt.Sprintf("%s:%s", t.tmpDir, t.tmpDir)) // bind the tmp dir to the container
			// },
			Entrypoint: []string{"/app/remotewrite"},
			Env: map[string]string{
				"API_KEY": t.cfg.GetAPIKey(),
				"PORT":    t.remoteWritePort,
			},
			LogConsumerCfg: &testcontainers.LogConsumerConfig{
				Consumers: []testcontainers.LogConsumer{&stdoutLogConsumer{}},
			},
			WaitingFor: wait.ForLog("Server is running on :"),
		}

		remotewrite, err := testcontainers.GenericContainer(t.ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: remotewriteReq,
			Started:          true,
		})
		require.NoError(t, err, "failed to create the mock remotewrite")

		fmt.Println("Mock remotewrite built successfully")
		t.remotewrite = &remotewrite

		// get the host
		host, err := remotewrite.Host(t.ctx)
		require.NoError(t, err, "failed to get the mock remotewrite host")

		// set the host as the setting
		t.SetSettings(func(settings *config.Settings) error {
			settings.Cloudzero.Host = fmt.Sprintf("%s:%s", host, t.remoteWritePort)
			return nil
		})
	}

	return t.remotewrite
}
