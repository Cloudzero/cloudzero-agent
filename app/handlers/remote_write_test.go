//go:build unit
// +build unit

package handlers_test

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/go-obvious/server/test"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/cloudzero/cirrus-remote-write/app/domain"
	"github.com/cloudzero/cirrus-remote-write/app/domain/testdata"
	"github.com/cloudzero/cirrus-remote-write/app/handlers"
	"github.com/cloudzero/cirrus-remote-write/app/types/mocks"
)

const MountBase = "/"

func setup(t *testing.T) (*gomock.Controller, *mocks.MockStore, *handlers.RemoteWriteAPI) {
	ctrl := gomock.NewController(t)
	storage := mocks.NewMockStore(ctrl)
	d := domain.NewMetricCollector(storage)
	handler := handlers.NewRemoteWriteAPI(MountBase, d)
	return ctrl, storage, handler
}

func createRequest(method, url string, body io.Reader) *http.Request {
	req, _ := http.NewRequest(method, url, body)
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Content-Encoding", "snappy")
	req.Header.Set("organization_id", "testorg")
	return req
}

func TestRemoteWriteMethods(t *testing.T) {
	t.Run("post v1 return 204", func(t *testing.T) {
		ctrl, store, handler := setup(t)
		defer ctrl.Finish()
		store.EXPECT().Put(gomock.Any(), gomock.Any()).Return(nil)

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
	})
}
