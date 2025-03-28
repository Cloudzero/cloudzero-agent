// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package housekeeper_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/domain/housekeeper"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/types"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/types/mocks"
)

func TestHouseKeeper_Start(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	initialTime := time.Date(2023, 10, 1, 12, 0, 0, 0, time.UTC)
	mockClock := mocks.NewMockClock(initialTime)

	ctx := context.Background()
	settings := &config.Settings{
		Database: config.Database{
			CleanupInterval: 10 * time.Millisecond,
			RetentionTime:   24 * time.Hour,
		},
	}

	mockStore := mocks.NewMockResourceStore(ctrl)

	hk := housekeeper.New(ctx, mockStore, mockClock, settings)

	t.Run("Start HouseKeeper", func(t *testing.T) {
		mockStore.EXPECT().FindAllBy(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
		err := hk.Run()
		require.NoError(t, err)
		assert.True(t, hk.IsRunning())
		hk.Shutdown()
		assert.False(t, hk.IsRunning())
	})

	t.Run("Start HouseKeeper with expired records", func(t *testing.T) {
		expiredRecords := []*types.ResourceTags{
			{ID: "1"},
			{ID: "2"},
		}

		mockStore.EXPECT().Tx(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
		mockStore.EXPECT().FindAllBy(gomock.Any(), gomock.Any()).Return(expiredRecords, nil).AnyTimes()
		mockStore.EXPECT().Delete(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

		err := hk.Run()
		require.NoError(t, err)
		assert.True(t, hk.IsRunning())

		// Wait for the cleanup interval to pass
		time.Sleep(10 * settings.Database.CleanupInterval)

		hk.Shutdown()
		assert.False(t, hk.IsRunning())
	})

	t.Run("Start HouseKeeper with FindAllBy error", func(t *testing.T) {
		mockStore.EXPECT().FindAllBy(gomock.Any(), gomock.Any()).Return(nil, assert.AnError).AnyTimes()

		err := hk.Run()
		require.NoError(t, err)
		assert.True(t, hk.IsRunning())

		// Wait for the cleanup interval to pass
		time.Sleep(10 * settings.Database.CleanupInterval)

		hk.Shutdown()
		assert.False(t, hk.IsRunning())
	})

	t.Run("Start HouseKeeper with Transaction error", func(t *testing.T) {
		expiredRecords := []*types.ResourceTags{
			{ID: "1"},
			{ID: "2"},
		}

		mockStore.EXPECT().Tx(gomock.Any(), gomock.Any()).Return(assert.AnError).AnyTimes()
		mockStore.EXPECT().FindAllBy(gomock.Any(), gomock.Any()).Return(expiredRecords, nil).AnyTimes()

		err := hk.Run()
		require.NoError(t, err)
		assert.True(t, hk.IsRunning())

		// Wait for the cleanup interval to pass
		time.Sleep(10 * settings.Database.CleanupInterval)

		hk.Shutdown()
		assert.False(t, hk.IsRunning())
	})
}
