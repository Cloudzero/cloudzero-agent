// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package domain_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/cloudzero/cloudzero-agent-validator/app/config/gator"
	"github.com/cloudzero/cloudzero-agent-validator/app/domain"
	"github.com/cloudzero/cloudzero-agent-validator/app/domain/testdata"
	"github.com/cloudzero/cloudzero-agent-validator/app/types/mocks"
)

func TestPutMetrics(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	initialTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	mockClock := mocks.NewMockClock(initialTime)

	ctx := context.Background()
	cfg := config.Settings{
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
		d, err := domain.NewMetricCollector(&cfg, mockClock, storage, nil)
		require.NoError(t, err)
		defer d.Close()

		payload, _, _, err := testdata.BuildWriteRequest(testdata.WriteRequestFixture.Timeseries, nil, nil, nil, nil, "snappy")
		require.NoError(t, err)
		stats, err := d.PutMetrics(ctx, "application/x-protobuf", "snappy", payload)
		assert.NoError(t, err)
		assert.Nil(t, stats)
	})

	t.Run("V2 Decode Path", func(t *testing.T) {
		storage := mocks.NewMockStore(ctrl)
		storage.EXPECT().Put(ctx, gomock.Any()).Return(nil)
		d, err := domain.NewMetricCollector(&cfg, mockClock, storage, nil)
		require.NoError(t, err)
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
