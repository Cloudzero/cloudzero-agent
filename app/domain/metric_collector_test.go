//go:build unit
// +build unit

package domain_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/cloudzero/cloudzero-insights-controller/app/config"
	"github.com/cloudzero/cloudzero-insights-controller/app/domain"
	"github.com/cloudzero/cloudzero-insights-controller/app/domain/testdata"
	"github.com/cloudzero/cloudzero-insights-controller/app/types/mocks"
)

func TestPutMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()
	cfg := config.Settings{
		OrganizationID: "testorg",
		CloudAccountID: "123456789012",
		Region:         "us-west-2",
		ClusterName:    "testcluster",
		Cloudzero: config.Cloudzero{
			Host:           "api.cloudzero.com",
			RotateInterval: 10 * time.Second,
		},
	}

	t.Run("V1 Decode with Compression", func(t *testing.T) {
		storage := mocks.NewMockStore(ctrl)
		storage.EXPECT().Put(ctx, gomock.Any()).Return(nil)
		d := domain.NewMetricCollector(&cfg, storage)
		defer d.Close()

		payload, _, _, err := testdata.BuildWriteRequest(testdata.WriteRequestFixture.Timeseries, nil, nil, nil, nil, "snappy")
		stats, err := d.PutMetrics(ctx, "application/x-protobuf", "snappy", payload)
		assert.NoError(t, err)
		assert.Nil(t, stats)
	})

	t.Run("V2 Decode Path", func(t *testing.T) {
		storage := mocks.NewMockStore(ctrl)
		storage.EXPECT().Put(ctx, gomock.Any()).Return(nil)
		d := domain.NewMetricCollector(&cfg, storage)
		defer d.Close()

		payload, _, _, err := testdata.BuildV2WriteRequest(
			testdata.WriteV2RequestFixture.Timeseries,
			testdata.WriteV2RequestFixture.Symbols,
			nil,
			nil,
			nil,
			"snappy",
		)
		assert.NoError(t, err)

		stats, err := d.PutMetrics(ctx, "application/x-protobuf;proto=io.prometheus.write.v2.Request", "snappy", payload)
		assert.NoError(t, err)
		assert.NotNil(t, stats)
	})
}
