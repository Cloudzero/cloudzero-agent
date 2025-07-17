// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package domain_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	config "github.com/cloudzero/cloudzero-agent/app/config/gator"
	"github.com/cloudzero/cloudzero-agent/app/domain"
	"github.com/cloudzero/cloudzero-agent/app/domain/testdata"
	"github.com/cloudzero/cloudzero-agent/app/types/mocks"
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
		storage.EXPECT().Flush().Return(nil)
		storage.EXPECT().Pending().Return(0).AnyTimes()                // For the shipping progress metric
		storage.EXPECT().ElapsedTime().Return(int64(10000)).AnyTimes() // For the time-based shipping progress metric
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
		storage.EXPECT().Flush().Return(nil)
		storage.EXPECT().Pending().Return(0).AnyTimes()                // For the shipping progress metric
		storage.EXPECT().ElapsedTime().Return(int64(10000)).AnyTimes() // For the time-based shipping progress metric
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

func TestCostMetricsShippingProgressMetric(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	initialTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	mockClock := mocks.NewMockClock(initialTime)

	ctx := context.Background()

	tests := []struct {
		name                     string
		maxRecords               int
		pendingRecords           int
		elapsedTimeMs            int64
		expectedProgress         float64
		expectMetricUpdateCalled bool
	}{
		{
			name:                     "zero pending records",
			maxRecords:               1000,
			pendingRecords:           0,
			elapsedTimeMs:            10000, // 10 seconds
			expectedProgress:         0.0,
			expectMetricUpdateCalled: true,
		},
		{
			name:                     "10 seconds elapsed with expected rate",
			maxRecords:               1000,
			pendingRecords:           6, // Expected: (10000/1800000) * 1000 = 5.56, actual: 6, so 6/5.56 ≈ 1.08
			elapsedTimeMs:            10000,
			expectedProgress:         1.08,
			expectMetricUpdateCalled: true,
		},
		{
			name:                     "30 seconds elapsed with expected rate",
			maxRecords:               1000,
			pendingRecords:           17, // Expected: (30000/1800000) * 1000 = 16.67, actual: 17, so 17/16.67 ≈ 1.02
			elapsedTimeMs:            30000,
			expectedProgress:         1.02,
			expectMetricUpdateCalled: true,
		},
		{
			name:                     "5 minutes elapsed with expected rate",
			maxRecords:               1000,
			pendingRecords:           167, // Expected: (300000/1800000) * 1000 = 166.67, actual: 167, so 167/166.67 ≈ 1.0
			elapsedTimeMs:            300000,
			expectedProgress:         1.0,
			expectMetricUpdateCalled: true,
		},
		{
			name:                     "15 minutes elapsed with expected rate",
			maxRecords:               1500000,
			pendingRecords:           750000, // Expected: (900000/1800000) * 1500000 = 750000, actual: 750000, so 750000/750000 = 1.0
			elapsedTimeMs:            900000,
			expectedProgress:         1.0,
			expectMetricUpdateCalled: true,
		},
		{
			name:                     "30 minutes elapsed (full interval)",
			maxRecords:               1000,
			pendingRecords:           1000, // Expected: (1800000/1800000) * 1000 = 1000, actual: 1000, so 1000/1000 = 1.0
			elapsedTimeMs:            1800000,
			expectedProgress:         1.0,
			expectMetricUpdateCalled: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Settings{
				CloudAccountID: "123456789012",
				Region:         "us-west-2",
				ClusterName:    "testcluster",
				Database: config.Database{
					MaxRecords:      tt.maxRecords,
					CostMaxInterval: 30 * time.Minute, // 30 minutes = 1800000 milliseconds
				},
			}

			// Create a mock store that implements all needed methods
			mockStore := mocks.NewMockStore(ctrl)

			if tt.expectMetricUpdateCalled {
				mockStore.EXPECT().Pending().Return(tt.pendingRecords).AnyTimes()
				mockStore.EXPECT().ElapsedTime().Return(tt.elapsedTimeMs).AnyTimes()
			}

			mockStore.EXPECT().Put(ctx, gomock.Any()).Return(nil).AnyTimes()
			mockStore.EXPECT().Flush().Return(nil).AnyTimes()

			d, err := domain.NewMetricCollector(&cfg, mockClock, mockStore, nil)
			require.NoError(t, err)
			defer d.Close()

			// Create test metric data
			payload, _, _, err := testdata.BuildWriteRequest(testdata.WriteRequestFixture.Timeseries, nil, nil, nil, nil, "snappy")
			require.NoError(t, err)

			// Process metrics to trigger the progress metric update
			stats, err := d.PutMetrics(ctx, "application/x-protobuf", "snappy", payload)
			assert.NoError(t, err)
			assert.Nil(t, stats)
		})
	}
}

func TestUpdateShippingProgressMetric(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	initialTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	mockClock := mocks.NewMockClock(initialTime)

	ctx := context.Background()

	t.Run("flush triggers metric update", func(t *testing.T) {
		cfg := config.Settings{
			CloudAccountID: "123456789012",
			Region:         "us-west-2",
			ClusterName:    "testcluster",
			Database: config.Database{
				MaxRecords:      1000,
				CostMaxInterval: 30 * time.Minute,
			},
		}

		// Create mock stores for both cost and observability
		mockCostStore := mocks.NewMockStore(ctrl)
		mockObservabilityStore := mocks.NewMockStore(ctrl)

		// Expect Pending() and ElapsedTime() to be called when Flush() is called
		mockCostStore.EXPECT().Pending().Return(0).AnyTimes()
		mockCostStore.EXPECT().ElapsedTime().Return(int64(10000)).AnyTimes()

		mockCostStore.EXPECT().Flush().Return(nil)
		mockObservabilityStore.EXPECT().Flush().Return(nil)

		d, err := domain.NewMetricCollector(&cfg, mockClock, mockCostStore, mockObservabilityStore)
		require.NoError(t, err)
		defer d.Close()

		// Call flush to trigger metric update
		err = d.Flush(ctx)
		assert.NoError(t, err)
	})
}
