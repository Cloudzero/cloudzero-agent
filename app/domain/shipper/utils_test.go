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
	"github.com/cloudzero/cloudzero-insights-controller/app/types"
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

func (m *MockAppendableFiles) Walk(loc string, process filepath.WalkFunc) error {
	args := m.Called(loc, process)

	// walk the specific location in the store
	if err := filepath.Walk(filepath.Join(m.baseDir, loc), process); err != nil {
		return fmt.Errorf("failed to walk the store: %w", err)
	}

	return args.Error(0)
}

func (m *MockAppendableFiles) GetUsage() (*types.StoreUsage, error) {
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
	err := os.Mkdir(filepath.Join(tmpDir, "uploaded"), 0o777)
	require.NoError(t, err)
	err = os.Mkdir(filepath.Join(tmpDir, "replay"), 0o777)
	require.NoError(t, err)
	return tmpDir
}

func getMockSettings(mockURL string) *config.Settings {
	return &config.Settings{
		ClusterName:    "test-cluster",
		CloudAccountID: "test-account",
		Region:         "us-east-1",
		Cloudzero: config.Cloudzero{
			Host:        mockURL,
			SendTimeout: time.Millisecond * 100,
		},
		Database: config.Database{
			StoragePath:          "/tmp/storage",
			StorageUploadSubpath: "uploaded",
		},
	}
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
		Cloudzero: config.Cloudzero{
			Host:        apiHost,
			SendTimeout: time.Second * 30,
			APIKeyPath:  filePath,
		},
		Database: config.Database{
			StoragePath: "/tmp/storage",
		},
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

var testMetrics = []types.Metric{
	{
		ClusterName:    "test-cluster",
		CloudAccountID: "1234567890",
		Year:           "2024",
		Month:          "1",
		Day:            "2",
		Hour:           "3",
		MetricName:     "test-metric-1",
		NodeName:       "my-node",
		CreatedAt:      time.Now().UnixMilli(),
		Value:          "I'm a value!",
		TimeStamp:      time.Now().UnixMilli(),
		Labels: map[string]string{
			"foo": "bar",
		},
	},
	{
		ClusterName:    "test-cluster",
		CloudAccountID: "1234567890",
		Year:           "2024",
		Month:          "1",
		Day:            "2",
		Hour:           "3",
		MetricName:     "test-metric-2",
		NodeName:       "my-node",
		CreatedAt:      time.Now().UnixMilli(),
		Value:          "I'm a value!",
		TimeStamp:      time.Now().UnixMilli(),
		Labels: map[string]string{
			"foo": "bar",
		},
	},
	{
		ClusterName:    "test-cluster",
		CloudAccountID: "1234567890",
		Year:           "2024",
		Month:          "1",
		Day:            "2",
		Hour:           "3",
		MetricName:     "test-metric-3",
		NodeName:       "my-node",
		CreatedAt:      time.Now().UnixMilli(),
		Value:          "I'm a value!",
		TimeStamp:      time.Now().UnixMilli(),
		Labels: map[string]string{
			"foo": "bar",
		},
	},
}

func compressedTestMetrics() []byte {
	jsonData, err := json.Marshal(testMetrics)
	if err != nil {
		panic(err)
	}

	var compressedData bytes.Buffer

	func() {
		compressor := brotli.NewWriterLevel(&compressedData, 1)
		defer compressor.Close()

		_, err = compressor.Write(jsonData)
		if err != nil {
			panic(err)
		}
	}()

	return compressedData.Bytes()
}

func createTestFiles(t *testing.T, dir string, n int) []*shipper.MetricFile {
	// create some test files to simulate resource tracking
	files := make([]*shipper.MetricFile, 0)
	for i := range n {
		tempFile, err := os.CreateTemp(dir, fmt.Sprintf("file-%d.parquet", i))
		require.NoError(t, err)
		_, err = tempFile.Write(compressedTestMetrics()) // write valid data in each one
		require.NoError(t, err)
		file, err := shipper.NewMetricFile(tempFile.Name())
		require.NoError(t, err)
		files = append(files, file)
	}
	return files
}
