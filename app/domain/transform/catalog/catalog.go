// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package catalog provides a catalog-based metric transformer that routes
// metrics to registered specialized transformers.
//
// The catalog transformer orchestrates multiple specialized transformers to
// provide automatic routing based on metric characteristics.
package catalog

import (
	"context"

	"github.com/cloudzero/cloudzero-agent/app/types"
)

// Transformer implements types.MetricTransformer using a catalog of specialized
// transformers.
//
// Each transformer in the catalog processes all metrics sequentially.
// Transformers identify which metrics they can handle and transform those while
// passing through others unchanged.
type Transformer struct {
	transformers []types.MetricTransformer
}

// NewTransformer creates a new catalog transformer with the provided
// specialized transformers.
//
// Transformers are applied sequentially - each transformer receives all metrics
// and decides which ones to transform based on implementation-specific logic
// (e.g., metric name patterns).
func NewTransformer(transformers ...types.MetricTransformer) *Transformer {
	return &Transformer{
		transformers: transformers,
	}
}

// Transform processes metrics by routing them sequentially through specialized
// transformers.
//
// Processing flow:
//  1. Pass metrics through first transformer
//  2. Pass results through second transformer
//  3. Continue until all transformers have processed the metrics
//
// This implements the types.MetricTransformer interface.
func (t *Transformer) Transform(ctx context.Context, metrics []types.Metric) ([]types.Metric, error) {
	if len(t.transformers) == 0 {
		return metrics, nil
	}

	// Process through each transformer in sequence
	result := metrics
	var err error

	for _, transformer := range t.transformers {
		result, err = transformer.Transform(ctx, result)
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}
