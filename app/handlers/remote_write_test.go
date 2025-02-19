// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package handlers_test

import (
	"bytes"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/go-obvious/server/test"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/cloudzero/cloudzero-insights-controller/app/config"
	"github.com/cloudzero/cloudzero-insights-controller/app/domain"
	"github.com/cloudzero/cloudzero-insights-controller/app/domain/testdata"
	"github.com/cloudzero/cloudzero-insights-controller/app/handlers"
	"github.com/cloudzero/cloudzero-insights-controller/app/types/mocks"
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

	d := domain.NewMetricCollector(&cfg, storage)
	defer d.Close()

	handler := handlers.NewRemoteWriteAPI(MountBase, d)

	storage.EXPECT().Put(gomock.Any(), gomock.Any()).Return(nil)

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
