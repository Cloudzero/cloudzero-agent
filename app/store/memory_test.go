// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package store_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cloudzero/cloudzero-insights-controller/app/store"
	"github.com/cloudzero/cloudzero-insights-controller/app/types"
)

func TestMemoryStore_All(t *testing.T) {
	ctx := context.Background()
	memoryStore := store.NewMemoryStore()

	t.Run("with an empty store", func(t *testing.T) {
		metricRange, err := memoryStore.All(ctx, nil)
		assert.NoError(t, err)
		assert.NotNil(t, metricRange)
		assert.Empty(t, metricRange.Metrics)
		assert.NotNil(t, metricRange.Next)
	})

	t.Run("with metrics in the store", func(t *testing.T) {
		memoryStore.Put(ctx, types.Metric{Id: "1", Name: "metric1"})
		memoryStore.Put(ctx, types.Metric{Id: "2", Name: "metric2"})

		metricRange, err := memoryStore.All(ctx, nil)
		assert.NoError(t, err)
		assert.NotNil(t, metricRange)
		assert.NotEmpty(t, metricRange.Metrics)
		assert.Len(t, metricRange.Metrics, 2)
		assert.NotNil(t, metricRange.Next)
	})
}

func TestMemoryStore_Get(t *testing.T) {
	ctx := context.Background()
	memoryStore := store.NewMemoryStore()

	t.Run("with a non-existent metric", func(t *testing.T) {
		metric, err := memoryStore.Get(ctx, "non-existent-id")
		assert.NoError(t, err)
		assert.Nil(t, metric)
	})

	t.Run("with an existing metric", func(t *testing.T) {
		expectedMetric := types.Metric{Id: "1", Name: "metric1"}
		memoryStore.Put(ctx, expectedMetric)

		metric, err := memoryStore.Get(ctx, "1")
		assert.NoError(t, err)
		assert.NotNil(t, metric)
		assert.Equal(t, expectedMetric, *metric)
	})
}

func TestMemoryStore_Delete(t *testing.T) {
	ctx := context.Background()
	memoryStore := store.NewMemoryStore()

	t.Run("with a non-existent metric", func(t *testing.T) {
		err := memoryStore.Delete(ctx, "non-existent-id")
		assert.NoError(t, err)
	})

	t.Run("with an existing metric", func(t *testing.T) {
		memoryStore.Put(ctx, types.Metric{Id: "1", Name: "metric1"})

		err := memoryStore.Delete(ctx, "1")
		assert.NoError(t, err)

		metric, err := memoryStore.Get(ctx, "1")
		assert.NoError(t, err)
		assert.Nil(t, metric)
	})
}
