//go:build unit
// +build unit

package handlers_test

import (
	"bytes"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/go-obvious/server/test"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/cloudzero/cirrus-remote-write/app/domain"
	"github.com/cloudzero/cirrus-remote-write/app/domain/testdata"
	"github.com/cloudzero/cirrus-remote-write/app/handlers"
	"github.com/cloudzero/cirrus-remote-write/app/types/mocks"
)

const MountBase = "/"

func createRequest(method, url string, body io.Reader) *http.Request {
	req, _ := http.NewRequest(method, url, body)
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Content-Encoding", "snappy")
	req.Header.Set("organization_id", "testorg")
	return req
}

func TestRemoteWriteMethods(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	storage := mocks.NewMockStore(ctrl)

	d := domain.NewMetricCollector(storage, 1000*time.Second)
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
