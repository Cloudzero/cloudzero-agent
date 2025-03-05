// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package smoke

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/cloudzero/cloudzero-insights-controller/app/config"
	"github.com/cloudzero/cloudzero-insights-controller/app/types"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/network"
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

func withUploadDelayMs(delayMs string) testContextOption {
	return func(t *testContext) {
		t.uploadDelayMs = delayMs
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
	apiKey       string // actual api key since the validate function is not run on the config
	apiKeyFile   string // location of the api key file
	configFile   string // location of the config file
	dataLocation string // location of actively running data for the collector/shipper

	// container names for docker networking
	collectorName   string
	shipperName     string
	s3instanceName  string
	remotewriteName string

	// internal docker state
	network     *testcontainers.DockerNetwork
	collector   *testcontainers.Container
	shipper     *testcontainers.Container
	s3instance  *testcontainers.Container
	remotewrite *testcontainers.Container

	// mock paramters
	uploadDelayMs string
}

func newTestContext(t *testing.T, opts ...testContextOption) *testContext {
	// create the temp dir structure
	tmpDir := t.TempDir()

	// create an api key file
	apiKey, exists := os.LookupEnv("CLOUDZERO_DEV_API_KEY")
	if !exists {
		apiKey = "ak-test"
	}

	remoteWritePort := "8081"
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

	// create the config
	cfg := config.Settings{
		CloudAccountID: "test-account-id",
		Region:         "us-east-1",
		ClusterName:    "smoke-test-cluster",
		Logging:        config.Logging{Level: "debug"},
		Database:       config.Database{StoragePath: dataLocation},
		Cloudzero: config.Cloudzero{
			APIKeyPath:   apiKeyFile,
			Host:         remoteWriteEndpoint,
			SendTimeout:  time.Second * 30,
			SendInterval: time.Duration(10) * time.Minute,
		},
	}

	// marshal into yaml
	modifiedConfig, err := yaml.Marshal(&cfg)
	require.NoError(t, err, "failed to marshal the config file")

	// write the config file
	configFile := filepath.Join(tmpDir, "config.yaml")
	err = os.WriteFile(configFile, modifiedConfig, 0o777)
	require.NoError(t, err, "failed to write the modified config file")
	require.NoError(t, err, "failed to read copied config file")

	// create the testing object
	tx := &testContext{
		T:               t,
		ctx:             context.Background(), // in go 1.24 use t.Context()
		cfg:             &cfg,
		configFile:      configFile,
		remoteWritePort: remoteWritePort,
		tmpDir:          tmpDir,
		apiKey:          apiKey,
		apiKeyFile:      apiKeyFile,
		dataLocation:    dataLocation,
		collectorName:   "cz-insights-controller-mock-collector",
		shipperName:     "cz-insights-controller-mock-shipper",
		s3instanceName:  "cz-insights-controller-mock-s3instance",
		remotewriteName: "cz-insights-controller-mock-remotewrite",
	}

	// run the options
	for _, opt := range opts {
		opt(tx)
	}

	if tx.uploadDelayMs == "" {
		tx.uploadDelayMs = "0"
	}

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
	if t.remotewrite != nil {
		(*t.remotewrite).Terminate(t.ctx)
	}
	if t.s3instance != nil {
		(*t.s3instance).Terminate(t.ctx)
	}
	if t.network != nil {
		t.network.Remove(t.ctx)
	}
}

// writes valid metric files to the shared data path `t.dataLocation`
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
