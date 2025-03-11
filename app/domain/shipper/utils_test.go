// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/cloudzero/cloudzero-insights-controller/app/config"
	"github.com/cloudzero/cloudzero-insights-controller/app/domain/shipper"
	"github.com/cloudzero/cloudzero-insights-controller/app/store"
	"github.com/cloudzero/cloudzero-insights-controller/app/types"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockAppendableFiles struct {
	mock.Mock
	baseDir string
}

func (m *MockAppendableFiles) GetFiles(paths ...string) ([]string, error) {
	args := m.Called(paths)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockAppendableFiles) ListFiles(paths ...string) ([]os.DirEntry, error) {
	args := m.Called(paths)
	return args.Get(0).([]os.DirEntry), args.Error(1)
}

func (m *MockAppendableFiles) Walk(loc string, process filepath.WalkFunc) error {
	args := m.Called(loc, process)

	// walk the specific location in the store
	if err := filepath.Walk(filepath.Join(m.baseDir, loc), process); err != nil {
		return fmt.Errorf("failed to walk the store: %w", err)
	}

	return args.Error(0)
}

func (m *MockAppendableFiles) GetUsage(paths ...string) (*types.StoreUsage, error) {
	args := m.Called()
	return args.Get(0).(*types.StoreUsage), args.Error(1)
}

func (m *MockAppendableFiles) Raw() (any, error) {
	return nil, nil
}

// MockRoundTripper is a mock implementation of http.RoundTripper
type MockRoundTripper struct {
	status                 int
	mockResponseBody       any
	mockResponseBodyString string
	mockError              error
}

func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.mockResponseBodyString != "" {
		return &http.Response{
			StatusCode: m.status,
			Body:       io.NopCloser(bytes.NewBuffer([]byte(m.mockResponseBodyString))),
		}, m.mockError
	} else {
		enc, err := json.Marshal(m.mockResponseBody)
		if err != nil {
			return nil, err
		}
		return &http.Response{
			StatusCode: m.status,
			Body:       io.NopCloser(bytes.NewBuffer(enc)),
		}, m.mockError
	}
}

func getTmpDir(t *testing.T) string {
	// get a tmp dir
	tmpDir := t.TempDir()
	err := os.Mkdir(filepath.Join(tmpDir, shipper.UploadedSubDirectory), 0o777)
	require.NoError(t, err)
	err = os.Mkdir(filepath.Join(tmpDir, shipper.ReplaySubDirectory), 0o777)
	require.NoError(t, err)
	return tmpDir
}

func getMockSettings(mockURL string) *config.Settings {
	cfg := &config.Settings{
		ClusterName:    "test-cluster",
		CloudAccountID: "test-account",
		Region:         "us-east-1",
		Cloudzero: config.Cloudzero{
			Host:        mockURL,
			SendTimeout: time.Millisecond * 1000,
		},
		Database: config.Database{
			StoragePath: "/tmp/storage",
			PurgeRules: config.PurgeRules{
				MetricsOlderThan: time.Duration(90) * time.Hour * 24,
				Lazy:             true,
				Percent:          20,
			},
		},
	}

	return cfg
}

func getMockSettingsIntegration(t *testing.T, dir, apiKey string) *config.Settings {
	// tmp file to write api key
	filePath := filepath.Join(dir, ".cz-api-key")
	err := os.WriteFile(filePath, []byte(apiKey), 0o644)
	require.NoError(t, err)

	// get the endpoint
	apiHost, exists := os.LookupEnv("CLOUDZERO_HOST")
	require.True(t, exists)

	// create the config
	cfg := &config.Settings{
		ClusterName:    "test-cluster",
		CloudAccountID: "test-account",
		Region:         "us-east-1",
		Logging: config.Logging{
			Level: "debug",
		},
		Cloudzero: config.Cloudzero{
			Host:        apiHost,
			SendTimeout: time.Second * 30,
			APIKeyPath:  filePath,
		},
		Database: config.Database{
			StoragePath: dir,
			PurgeRules: config.PurgeRules{
				MetricsOlderThan: time.Duration(90) * time.Hour * 24,
				Lazy:             true,
				Percent:          20,
			},
		},
	}

	var logger zerolog.Logger
	{
		logLevel, parseErr := zerolog.ParseLevel(cfg.Logging.Level)
		require.NoError(t, parseErr)
		logger = zerolog.New(os.Stdout).Level(logLevel).With().Timestamp().Logger()
		zerolog.DefaultContextLogger = &logger
	}

	// validate the config
	err = cfg.SetAPIKey()
	require.NoError(t, err)
	err = cfg.SetRemoteUploadAPI()
	require.NoError(t, err)

	return cfg
}

func captureOutput(f func()) (string, string) {
	// save original
	oldOut := os.Stdout
	oldErr := os.Stderr

	// create out pipes
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()

	// redirect stdout and stderr
	os.Stdout = wOut
	os.Stderr = wErr

	// fun the passed test func
	f()

	// restore
	os.Stdout = oldOut
	os.Stderr = oldErr

	// read output
	wOut.Close()
	wErr.Close()

	// write into buf
	var outBuf, errBuf bytes.Buffer
	io.Copy(&outBuf, rOut)
	io.Copy(&errBuf, rErr)

	return outBuf.String(), errBuf.String()
}

func createTestFiles(t *testing.T, dir string, n int) []types.File {
	files := make([]types.File, 0)

	for i := range n {
		now := time.Now()

		// create a file location
		path := filepath.Join(dir, fmt.Sprintf("metrics_%d_%05d.json.br", now.UnixMilli(), i))
		file, err := os.Create(path)
		require.NoError(t, err, "failed to create file: %s", err)

		// create the metrics array
		metrics := make([]*types.Metric, 5)
		for j := range 5 {
			metrics[j] = &types.Metric{
				ClusterName:    "test-cluster",
				CloudAccountID: "1234567890",
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

		f, err := store.NewMetricFile(path)
		require.NoError(t, err, "failed to create metric file")
		files = append(files, f)
	}

	return files
}
