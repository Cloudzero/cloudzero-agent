// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package handlers_test

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/go-obvious/server/test"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	config "github.com/cloudzero/cloudzero-agent/app/config/gator"
	"github.com/cloudzero/cloudzero-agent/app/domain"
	"github.com/cloudzero/cloudzero-agent/app/domain/testdata"
	"github.com/cloudzero/cloudzero-agent/app/handlers"
	"github.com/cloudzero/cloudzero-agent/app/http/middleware"
	"github.com/cloudzero/cloudzero-agent/app/types/mocks"
)

const MountBase = "/"

func createRequest(method, url string, body io.Reader) *http.Request {
	req, _ := http.NewRequest(method, url, body)
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Content-Encoding", "snappy")
	return req
}

func TestRemoteWriteMethods(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	initialTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	mockClock := mocks.NewMockClock(initialTime)

	storage := mocks.NewMockStore(ctrl)

	cfg := config.Settings{
		CloudAccountID: "123456789012",
		Region:         "us-west-2",
		ClusterName:    "testcluster",
		Cloudzero: config.Cloudzero{
			Host:           "api.cloudzero.com",
			RotateInterval: 10 * time.Minute,
		},
	}

	d, err := domain.NewMetricCollector(&cfg, mockClock, storage, nil)
	assert.NoError(t, err)
	defer d.Close()

	handler := handlers.NewRemoteWriteAPI(MountBase, d)

	storage.EXPECT().Put(gomock.Any(), gomock.Any()).Return(nil)
	storage.EXPECT().Flush().Return(nil)

	payload, _, _, err := testdata.BuildWriteRequest(testdata.WriteRequestFixture.Timeseries, nil, nil, nil, nil, "snappy")
	assert.NoError(t, err)

	req := createRequest("POST", "/", bytes.NewReader(payload))

	q := req.URL.Query()
	q.Add("region", "us-west-2")
	q.Add("cloud_account_id", "123456789012")
	q.Add("cluster_name", "testcluster")
	req.URL.RawQuery = q.Encode()

	resp, err := test.InvokeService(handler.Service, "/", *req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

// TestRemoteWrite_ConnectionCloseOn500 verifies that when the handler returns
// a 5xx response, it sets the Connection: close header on HTTP/1.x responses.
// This forces the client to tear down the TCP connection, so that any
// subsequent request will be freshly load-balanced by kube-proxy (instead of
// staying pinned to a pod that is persistently returning errors).
func TestRemoteWrite_ConnectionCloseOn500(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	initialTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	mockClock := mocks.NewMockClock(initialTime)

	storage := mocks.NewMockStore(ctrl)

	cfg := config.Settings{
		CloudAccountID: "123456789012",
		Region:         "us-west-2",
		ClusterName:    "testcluster",
		Cloudzero: config.Cloudzero{
			Host:           "api.cloudzero.com",
			RotateInterval: 10 * time.Minute,
		},
	}

	d, err := domain.NewMetricCollector(&cfg, mockClock, storage, nil)
	assert.NoError(t, err)
	defer d.Close()

	handler := handlers.NewRemoteWriteAPI(MountBase, d)

	// Force the store to error so PutMetrics returns an error, which makes the
	// handler respond 500.
	storage.EXPECT().Put(gomock.Any(), gomock.Any()).Return(errors.New("boom"))

	payload, _, _, err := testdata.BuildWriteRequest(testdata.WriteRequestFixture.Timeseries, nil, nil, nil, nil, "snappy")
	assert.NoError(t, err)

	req := createRequest("POST", "/", bytes.NewReader(payload))

	q := req.URL.Query()
	q.Add("region", "us-west-2")
	q.Add("cloud_account_id", "123456789012")
	q.Add("cluster_name", "testcluster")
	req.URL.RawQuery = q.Encode()

	resp, err := test.InvokeService(handler.Service, "/", *req)
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	// Go's net/http strips Connection: close from resp.Header and surfaces it
	// as resp.Close. On a 5xx, we want the server to have told the client to
	// close the TCP connection so the next request is re-balanced by
	// kube-proxy instead of staying pinned to a failing pod.
	assert.True(t, resp.Close,
		"5xx responses on HTTP/1.x should set Connection: close so the client reconnects and is re-balanced")
}

// TestRemoteWrite_ErrorRateTrackerWiring verifies that when a RemoteWriteAPI
// is constructed with an ErrorRateTracker, responses flowing through the
// handler are recorded into the tracker. This is the wiring a health check
// relies on.
func TestRemoteWrite_ErrorRateTrackerWiring(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	initialTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	mockClock := mocks.NewMockClock(initialTime)

	storage := mocks.NewMockStore(ctrl)

	cfg := config.Settings{
		CloudAccountID: "123456789012",
		Region:         "us-west-2",
		ClusterName:    "testcluster",
		Cloudzero: config.Cloudzero{
			Host:           "api.cloudzero.com",
			RotateInterval: 10 * time.Minute,
		},
	}

	d, err := domain.NewMetricCollector(&cfg, mockClock, storage, nil)
	assert.NoError(t, err)
	defer d.Close()

	tracker := middleware.NewErrorRateTracker(60 * time.Second)
	handler := handlers.NewRemoteWriteAPI(MountBase, d, handlers.WithErrorRateTracker(tracker))

	// 3 failing requests so we clear the minFailures=3 floor.
	storage.EXPECT().Put(gomock.Any(), gomock.Any()).Return(errors.New("boom")).Times(3)

	payload, _, _, err := testdata.BuildWriteRequest(testdata.WriteRequestFixture.Timeseries, nil, nil, nil, nil, "snappy")
	assert.NoError(t, err)

	for i := 0; i < 3; i++ {
		req := createRequest("POST", "/", bytes.NewReader(payload))
		resp, err := test.InvokeService(handler.Service, "/", *req)
		assert.NoError(t, err)
		_ = resp.Body.Close()
		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	}

	assert.False(t, tracker.Healthy(0.20, 3),
		"after 3 × 500 through the RemoteWriteAPI, the tracker should report unhealthy")
}
