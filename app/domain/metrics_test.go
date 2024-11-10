//go:build unit
// +build unit

package domain_test

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/cloudzero/cirrus-remote-write/app/domain"
	"github.com/cloudzero/cirrus-remote-write/app/domain/testdata"
	"github.com/cloudzero/cirrus-remote-write/app/store"
	"github.com/cloudzero/cirrus-remote-write/app/types"
	"github.com/cloudzero/cirrus-remote-write/app/types/mocks"
)

func TestGetMetricNotFound(t *testing.T) {
	storage := store.NewMemoryStore()
	d := domain.NewMetricsDomain(storage, nil)

	metric, err := d.GetMetric(context.Background(), "1")
	assert.NoError(t, err)
	assert.Nil(t, metric)
}

func TestGetExistingMetric(t *testing.T) {
	ctx := context.Background()

	storage := store.NewMemoryStore()
	storage.Put(ctx, types.Metric{Id: "1", Name: "cloudzero_metric"})

	d := domain.NewMetricsDomain(storage, nil)

	metric, err := d.GetMetric(context.Background(), "1")
	assert.NoError(t, err)
	assert.NotNil(t, metric)
	assert.Equal(t, "cloudzero_metric", metric.Name)
}

func TestGetInternalStoreError(t *testing.T) {
	ctrl := gomock.NewController(t)
	ctx := context.Background()

	storage := mocks.NewMockStore(ctrl)
	storage.EXPECT().
		Get(ctx, gomock.Eq("1")).
		Return(nil, errors.New("internal error"))

	d := domain.NewMetricsDomain(storage, nil)

	product, err := d.GetMetric(ctx, "1")
	assert.Error(t, err)
	assert.EqualError(t, err, "internal error")
	assert.Nil(t, product)
}

func TestAllMetricsWithInvalidNext(t *testing.T) {
	ctrl := gomock.NewController(t)
	ctx := context.Background()

	storage := mocks.NewMockStore(ctrl)
	storage.EXPECT().
		All(ctx, gomock.Nil()).
		AnyTimes()

	d := domain.NewMetricsDomain(storage, nil)

	t.Parallel()

	t.Run("with nil 'next'", func(t *testing.T) {
		d.AllMetrics(ctx, nil)
	})

	t.Run("with empty 'next'", func(t *testing.T) {
		next := ""
		d.AllMetrics(ctx, &next)
	})

	t.Run("with empty spaces 'next'", func(t *testing.T) {
		next := "  "
		d.AllMetrics(ctx, &next)
	})
}

func TestAllAllMetricsInternalStoreError(t *testing.T) {
	ctrl := gomock.NewController(t)
	ctx := context.Background()

	storage := mocks.NewMockStore(ctrl)
	storage.EXPECT().
		All(ctx, gomock.All()).
		Return(types.MetricRange{}, errors.New("internal error"))

	d := domain.NewMetricsDomain(storage, nil)

	_, err := d.AllMetrics(ctx, nil)
	assert.Error(t, err)
	assert.EqualError(t, err, "internal error")
}

func TestAllMetrics(t *testing.T) {
	storage := store.NewMemoryStore()
	d := domain.NewMetricsDomain(storage, nil)
	ctx := context.Background()

	t.Run("with an empty store", func(t *testing.T) {
		metricRange, err := d.AllMetrics(ctx, nil)
		assert.NoError(t, err)
		assert.NotNil(t, metricRange)
		assert.Empty(t, metricRange.Metrics)
	})

	t.Run("with products on the store", func(t *testing.T) {
		storage.Put(ctx, types.Metric{Id: "2", Name: "cloudzero_metric"})

		metricRange, err := d.AllMetrics(ctx, nil)
		assert.NoError(t, err)
		assert.NotNil(t, metricRange)
		assert.NotEmpty(t, metricRange.Metrics)
		assert.Len(t, metricRange.Metrics, 1)
	})
}

func TestGetMetricNames(t *testing.T) {
	ctx := context.Background()

	t.Run("with an empty store", func(t *testing.T) {
		storage := store.NewMemoryStore()
		d := domain.NewMetricsDomain(storage, nil)

		names, err := d.GetMetricNames(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, names)
		assert.Empty(t, names)
	})

	t.Run("with metrics in the store", func(t *testing.T) {
		storage := store.NewMemoryStore()
		storage.Put(ctx, types.Metric{Id: "1", Name: "metric1"})
		storage.Put(ctx, types.Metric{Id: "2", Name: "metric2"})
		storage.Put(ctx, types.Metric{Id: "3", Name: "metric1"}) // Duplicate name

		d := domain.NewMetricsDomain(storage, nil)

		names, err := d.GetMetricNames(ctx)
		assert.NoError(t, err)
		assert.NotNil(t, names)
		assert.Len(t, names, 2)
		assert.Contains(t, names, "metric1")
		assert.Contains(t, names, "metric2")
	})

	t.Run("with internal store error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		store := mocks.NewMockStore(ctrl)
		store.EXPECT().
			All(ctx, gomock.Nil()).
			Return(types.MetricRange{}, errors.New("internal error"))

		d := domain.NewMetricsDomain(store, nil)

		names, err := d.GetMetricNames(ctx)
		assert.Error(t, err)
		assert.EqualError(t, err, "internal error")
		assert.Nil(t, names)
	})
}

func TestPutMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	ctx := context.Background()

	t.Run("V1 Decode with Compression", func(t *testing.T) {
		storage := mocks.NewMockStore(ctrl)
		storage.EXPECT().Put(ctx, gomock.Any()).Return(nil)
		d := domain.NewMetricsDomain(storage, nil)

		payload, _, _, err := testdata.BuildWriteRequest(testdata.WriteRequestFixture.Timeseries, nil, nil, nil, nil, "snappy")
		stats, err := d.PutMetrics(ctx, "application/x-protobuf", "snappy", payload)
		assert.NoError(t, err)
		assert.Nil(t, stats)
	})

	t.Run("V2 Decode Path", func(t *testing.T) {
		storage := mocks.NewMockStore(ctrl)
		storage.EXPECT().Put(ctx, gomock.Any()).Return(nil)
		d := domain.NewMetricsDomain(storage, nil)

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
