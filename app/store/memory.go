// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package store

// import (
// 	"context"

// 	"github.com/aws/aws-sdk-go-v2/aws"

// 	"github.com/cloudzero/cloudzero-insights-controller/app/types"
// )

// type MemoryStore struct {
// 	storage map[string]types.Metric
// }

// // Flush implements types.Store.
// func (m *MemoryStore) Flush() error {
// 	m.storage = make(map[string]types.Metric)
// 	return nil
// }

// // Pending implements types.Store.
// func (m *MemoryStore) Pending() int {
// 	return len(m.storage)
// }

// // Just to make sure MemoryStore implements the Store interface
// var _ types.Store = (*MemoryStore)(nil)

// func NewMemoryStore() *MemoryStore {
// 	return &MemoryStore{
// 		storage: make(map[string]types.Metric),
// 	}
// }

// func (m *MemoryStore) All(ctx context.Context, next *string) (types.MetricRange, error) {
// 	metricsRange := types.MetricRange{
// 		Metrics: []types.Metric{},
// 		Next:    aws.String("random next string"),
// 	}

// 	for _, v := range m.storage {
// 		metricsRange.Metrics = append(metricsRange.Metrics, v)
// 	}

// 	return metricsRange, nil
// }

// func (m *MemoryStore) Get(ctx context.Context, id string) (*types.Metric, error) {
// 	p, ok := m.storage[id]
// 	if !ok {
// 		return nil, nil //nolint:nilnil // we should probably revisit this API at some point, but not today
// 	}

// 	return &p, nil
// }

// func (m *MemoryStore) Put(ctx context.Context, metrics ...types.Metric) error {
// 	for _, metric := range metrics {
// 		m.storage[metric.ID.String()] = metric
// 	}
// 	return nil
// }

// func (m *MemoryStore) Delete(ctx context.Context, id string) error {
// 	delete(m.storage, id)

// 	return nil
// }
