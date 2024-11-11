//go:build unit
// +build unit

package domain_test

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/cloudzero/cirrus-remote-write/app/domain"
	"github.com/cloudzero/cirrus-remote-write/app/domain/testdata"
	"github.com/cloudzero/cirrus-remote-write/app/types/mocks"
)

func TestPutMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	t.Run("V1 Decode with Compression", func(t *testing.T) {
		storage := mocks.NewMockStore(ctrl)
		storage.EXPECT().Put(ctx, gomock.Any()).Return(nil)
		d := domain.NewMetricCollector(storage)

		payload, _, _, err := testdata.BuildWriteRequest(testdata.WriteRequestFixture.Timeseries, nil, nil, nil, nil, "snappy")
		stats, err := d.PutMetrics(ctx, "application/x-protobuf", "snappy", payload)
		assert.NoError(t, err)
		assert.Nil(t, stats)
	})

	t.Run("V2 Decode Path", func(t *testing.T) {
		storage := mocks.NewMockStore(ctrl)
		storage.EXPECT().Put(ctx, gomock.Any()).Return(nil)
		d := domain.NewMetricCollector(storage)

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
