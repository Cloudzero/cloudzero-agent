//go:build unit
// +build unit

package handlers_test

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/go-obvious/server/test"
	"github.com/go-obvious/timestamp"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/cloudzero/cirrus-remote-write/app/domain"
	"github.com/cloudzero/cirrus-remote-write/app/domain/testdata"
	"github.com/cloudzero/cirrus-remote-write/app/handlers"
	"github.com/cloudzero/cirrus-remote-write/app/types"
	"github.com/cloudzero/cirrus-remote-write/app/types/mocks"
)

const MountBase = "/"

func setup(t *testing.T) (*gomock.Controller, *mocks.MockStore, *handlers.RemoteWrite) {
	ctrl := gomock.NewController(t)
	storage := mocks.NewMockStore(ctrl)
	d := domain.NewMetricsDomain(storage, nil)
	handler := handlers.NewRemoteWrite(MountBase, d)
	return ctrl, storage, handler
}

func createRequest(method, url string, body io.Reader) *http.Request {
	req, _ := http.NewRequest(method, url, body)
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Content-Encoding", "snappy")
	req.Header.Set("organization_id", "testorg")
	return req
}

func TestRemoteWriteGetAll(t *testing.T) {
	t.Run("get all returns 200", func(t *testing.T) {
		ctrl, store, handler := setup(t)
		defer ctrl.Finish()
		store.EXPECT().All(gomock.Any(), gomock.Any()).Return(types.MetricRange{Metrics: []types.Metric{}}, nil)

		req := createRequest(http.MethodGet, MountBase, strings.NewReader(""))

		resp, err := test.InvokeService(handler.Service, MountBase, *req)
		assert.NoError(t, err)

		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		bodyStr := strings.TrimSpace(string(body))
		assert.JSONEq(t, `{"metrics":[]}`, bodyStr)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("get counts returns 200", func(t *testing.T) {
		ctrl, store, handler := setup(t)
		defer ctrl.Finish()
		store.EXPECT().All(gomock.Any(), gomock.Any()).Return(types.MetricRange{Metrics: []types.Metric{}}, nil)

		req := createRequest(http.MethodGet, "/count", strings.NewReader(""))

		resp, err := test.InvokeService(handler.Service, "/count", *req)
		assert.NoError(t, err)

		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		bodyStr := strings.TrimSpace(string(body))
		assert.Equal(t, "0", bodyStr)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("get names returns 200", func(t *testing.T) {
		ctrl, store, handler := setup(t)
		defer ctrl.Finish()
		store.EXPECT().All(gomock.Any(), gomock.Any()).Return(types.MetricRange{Metrics: []types.Metric{
			types.NewMetric("test_metric_1", timestamp.Milli(), map[string]string{"test": "test1"}, "1.0"),
			types.NewMetric("test_metric_2", timestamp.Milli(), map[string]string{"test": "test2"}, "2.0"),
			types.NewMetric("test_metric_3", timestamp.Milli(), map[string]string{"test": "test3"}, "3.0"),
		}}, nil)

		req := createRequest(http.MethodGet, "/names", strings.NewReader(""))

		resp, err := test.InvokeService(handler.Service, "/names", *req)
		assert.NoError(t, err)

		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		bodyStr := strings.TrimSpace(string(body))
		assert.JSONEq(t, `["test_metric_1","test_metric_2","test_metric_3"]`, bodyStr)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("get name returns 200", func(t *testing.T) {
		ctrl, store, handler := setup(t)
		defer ctrl.Finish()
		store.EXPECT().All(gomock.Any(), gomock.Any()).Return(types.MetricRange{Metrics: []types.Metric{
			types.NewMetric("test_metric_1", timestamp.Milli(), map[string]string{"test": "test1"}, "1.0"),
			types.NewMetric("test_metric_2", timestamp.Milli(), map[string]string{"test": "test2"}, "2.0"),
			types.NewMetric("test_metric_3", timestamp.Milli(), map[string]string{"test": "test3"}, "3.0"),
		}}, nil)

		req := createRequest(http.MethodGet, "/names/test_metric_1", strings.NewReader(""))

		resp, err := test.InvokeService(handler.Service, "/names/test_metric_1", *req)
		assert.NoError(t, err)

		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		bodyStr := strings.TrimSpace(string(body))
		assert.Equal(t, `"test_metric_1"`, bodyStr)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

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
