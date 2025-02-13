// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/cloudzero/cloudzero-insights-controller/app/config"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockAppendableFiles struct {
	mock.Mock
}

func (m *MockAppendableFiles) GetFiles() ([]string, error) {
	args := m.Called()
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockAppendableFiles) GetMatching(loc string, requests []string) ([]string, error) {
	args := m.Called(loc, requests)
	return args.Get(0).([]string), args.Error(1)
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

func setupSettings(mockURL string) *config.Settings {
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

func createTestFiles(t *testing.T, dir string, n int) []*File {
	// create some test files to simulate resource tracking
	files := make([]*File, 0)
	for i := range n {
		tempFile, err := os.CreateTemp(dir, fmt.Sprintf("file-%d.parquet", i))
		require.NoError(t, err)
		_, err = tempFile.WriteString(fmt.Sprintf("This is some test data - %d", n))
		require.NoError(t, err)
		file, err := NewFile(tempFile.Name())
		require.NoError(t, err)
		files = append(files, file)
	}
	return files
}
