// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package transform provides metric transformation capabilities for
// standardizing vendor-specific metrics into common formats for cost
// allocation.
package transform

import (
	"github.com/cloudzero/cloudzero-agent/app/domain/transform/catalog"
	"github.com/cloudzero/cloudzero-agent/app/domain/transform/dcgm"
	"github.com/cloudzero/cloudzero-agent/app/types"
)

// NewMetricTransformer creates a new MetricTransformer with all registered
// specialized transformers.
//
// This is the primary entry point for metric transformation, following the
// Scout pattern. Add new specialized transformers here as peer implementations
// (Intel XPU, AMD ROCm, network, etc.).
func NewMetricTransformer() types.MetricTransformer {
	return catalog.NewTransformer(
		dcgm.NewTransformer(),
	)
}
