// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package telemetry_test

import (
	"bytes"
	"compress/gzip"
	"context"
	_ "embed"
	"io"
	net "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"google.golang.org/protobuf/proto"

	"github.com/cloudzero/cloudzero-agent-validator/pkg/config"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/http"
	pb "github.com/cloudzero/cloudzero-agent-validator/pkg/status"
	"github.com/cloudzero/cloudzero-agent-validator/pkg/telemetry"
	"github.com/stretchr/testify/assert"
)

//go:embed testdata/prometheus.yml
var prometheusScrapeConfig []byte

const (
	// Test message constants
	Account          = "my-account"
	Region           = "us-east-1"
	Name             = "my-cluster"
	State            = pb.StatusType_STATUS_TYPE_INIT_STARTED
	ChartVersion     = "0.0.1"
	AgentVersion     = "0.0.2"
	ValidatorVersion = "0.0.3"
	K8SVersion       = "0.0.4"
)

var (
	// Test message constants
	ScrapeConfig = string(prometheusScrapeConfig)

	// Test status checks
	Check1 = &pb.StatusCheck{Name: "check1", Passing: true}
	Check2 = &pb.StatusCheck{Name: "check2", Passing: false}

	// Test header
	Headers = map[string]string{
		"Authorization": "Bearer myToken",
	}
)

// global for easier testing
var serverDecodedStatus pb.ClusterStatus

// TestPostStatus tests the PostStatus function.
func TestPostStatus(t *testing.T) {
	tcase := []struct {
		name       string
		badRequest bool
		doTimeout  bool
		wantErr    bool
	}{
		{
			name: "good",
		},
		// {
		// 	name:       "bad request error",
		// 	badRequest: true,
		// 	wantErr:    true,
		// },
		// {
		// 	name:      "timeout error",
		// 	doTimeout: true,
		// 	wantErr:   true,
		// },
	}

	// Create a mock HTTP server
	for _, tc := range tcase {
		t.Run(tc.name, func(t *testing.T) {
			handler := NewTestServerHandler(t).Handler
			if tc.badRequest {
				handler = func(w net.ResponseWriter, r *net.Request) {
					w.WriteHeader(net.StatusBadRequest)
				}
			} else if tc.doTimeout {
				handler = func(w net.ResponseWriter, r *net.Request) {
					time.Sleep(6 * time.Second)
					w.WriteHeader(net.StatusOK)
				}
			}

			server := httptest.NewServer(net.HandlerFunc(handler))
			defer server.Close()

			cfg := &config.Settings{
				Cloudzero: config.Cloudzero{
					Host:       server.URL,
					Credential: "myToken",
				},
			}

			// Create a test message
			builder := pb.NewAccessor(createTestClusterStatus(t))
			accessor := builder.(pb.Accessor)

			// Create a test context
			ctx := context.Background()

			// Create a test HTTP client
			client := &net.Client{
				Timeout: 5 * time.Second,
			}

			// Call the function being tested
			err := telemetry.Post(ctx, client, cfg, accessor)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}

			// Now verify the serverDecodedStatus
			assert.Equal(t, Account, serverDecodedStatus.Account)
			assert.Equal(t, Region, serverDecodedStatus.Region)
			assert.Equal(t, Name, serverDecodedStatus.Name)
			assert.Equal(t, State, serverDecodedStatus.State)
			assert.Equal(t, ChartVersion, serverDecodedStatus.ChartVersion)
			assert.Equal(t, AgentVersion, serverDecodedStatus.AgentVersion)
			assert.Equal(t, ScrapeConfig, serverDecodedStatus.ScrapeConfig)
			assert.Equal(t, ValidatorVersion, serverDecodedStatus.ValidatorVersion)
			assert.Equal(t, K8SVersion, serverDecodedStatus.K8SVersion)
			assert.Len(t, serverDecodedStatus.Checks, 2)
		})
	}

}

// Testing Helpers

type testHandler struct {
	t *testing.T
}

func NewTestServerHandler(t *testing.T) *testHandler {
	t.Helper()
	return &testHandler{
		t: t,
	}
}

// handlePostStatusRequest is a helper function that handles the HTTP request in the mock server.
func (ts *testHandler) Handler(w net.ResponseWriter, r *net.Request) {
	ts.t.Helper()
	// Verify the request headers
	contentEncoding := r.Header.Get(http.HeaderContentEncoding)
	contentType := r.Header.Get(http.HeaderContentType)
	if contentEncoding != http.ContentTypeGzip {
		net.Error(w, "unexpected content encoding", net.StatusBadRequest)
		return
	}
	if contentType != http.ContentTypeProtobuf {
		net.Error(w, "unexpected content type", net.StatusBadRequest)
		return
	}

	// Verify the request body
	// Read the compressed request body
	compressedData, err := io.ReadAll(r.Body)
	if err != nil {
		net.Error(w, "Failed to read request body", net.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	// Decompress the gzip data
	gzipReader, err := gzip.NewReader(bytes.NewBuffer(compressedData))
	if err != nil {
		net.Error(w, "Failed to create gzip reader", net.StatusInternalServerError)
		return
	}
	defer gzipReader.Close()

	// Read the decompressed data
	decompressedData, err := io.ReadAll(gzipReader)
	if err != nil {
		net.Error(w, "Failed to read decompressed data", net.StatusInternalServerError)
		return
	}

	// Unmarshal the Protobuf message
	err = proto.Unmarshal(decompressedData, &serverDecodedStatus)
	if err != nil {
		net.Error(w, "Failed to unmarshal protobuf message", net.StatusBadRequest)
		return
	}

	// Write a response
	w.WriteHeader(net.StatusOK)
}

// createTestClusterStatus creates a test ClusterStatus message.
func createTestClusterStatus(t *testing.T) *pb.ClusterStatus {
	t.Helper()
	return &pb.ClusterStatus{
		Account:          Account,
		Region:           Region,
		Name:             Name,
		State:            State,
		ChartVersion:     ChartVersion,
		AgentVersion:     AgentVersion,
		ScrapeConfig:     ScrapeConfig,
		ValidatorVersion: ValidatorVersion,
		K8SVersion:       K8SVersion,
		Checks:           []*pb.StatusCheck{Check1, Check2},
	}
}
