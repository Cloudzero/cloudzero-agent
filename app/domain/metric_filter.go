// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"fmt"

	"github.com/cloudzero/cloudzero-agent-validator/app/config/gator"
	"github.com/cloudzero/cloudzero-agent-validator/app/domain/filter"
	"github.com/cloudzero/cloudzero-agent-validator/app/types"
)

// MetricFilter is a filter that can be used to filter metrics, including
// labels, according to the configuration provided.
type MetricFilter struct {
	cost                *filter.FilterChecker
	observability       *filter.FilterChecker
	costLabels          *filter.FilterChecker
	observabilityLabels *filter.FilterChecker
}

// NewMetricFilter creates a new MetricFilter for the given configuration.
func NewMetricFilter(cfg *config.Metrics) (*MetricFilter, error) {
	var err error
	mf := &MetricFilter{}
	haveFilter := false

	if len(cfg.Cost) != 0 {
		mf.cost, err = filter.NewFilterChecker(cfg.Cost)
		if err != nil {
			return nil, fmt.Errorf("failed to compile cost filter: %w", err)
		}
		haveFilter = true
	}

	if len(cfg.CostLabels) != 0 {
		mf.costLabels, err = filter.NewFilterChecker(cfg.CostLabels)
		if err != nil {
			return nil, fmt.Errorf("failed to compile cost labels filter: %w", err)
		}
		haveFilter = true
	}

	if len(cfg.Observability) != 0 {
		mf.observability, err = filter.NewFilterChecker(cfg.Observability)
		if err != nil {
			return nil, fmt.Errorf("failed to compile observability filter: %w", err)
		}
		haveFilter = true
	}

	if len(cfg.ObservabilityLabels) != 0 {
		mf.observabilityLabels, err = filter.NewFilterChecker(cfg.ObservabilityLabels)
		if err != nil {
			return nil, fmt.Errorf("failed to compile observability labels filter: %w", err)
		}
		haveFilter = true
	}

	if !haveFilter {
		return nil, nil //nolint:nilnil // methods handle nil properly, returning nil allows us to elide code
	}

	return mf, nil
}

// Filter processes the supplied metrics through the filter. It returns two
// slices, the first being the list of cost metrics, the second being the list
// of observability metrics, both of which have also had the labels filtered
// to only include those that match the filter.
func (mf *MetricFilter) Filter(metrics []types.Metric) (costMetrics []types.Metric, observabilityMetrics []types.Metric) {
	if mf == nil {
		return metrics, metrics
	}

	for _, metric := range metrics {
		if mf.cost == nil || mf.cost.Test(metric.MetricName) {
			costMetric := metric

			if mf.costLabels != nil {
				costMetric.Labels = map[string]string{}
				for k, v := range metric.Labels {
					if mf.costLabels.Test(k) {
						costMetric.Labels[k] = v
					}
				}
			}

			costMetrics = append(costMetrics, costMetric)
		}

		if mf.observability == nil || mf.observability.Test(metric.MetricName) {
			observabilityMetric := metric

			if mf.observabilityLabels != nil {
				observabilityMetric.Labels = map[string]string{}
				for k, v := range metric.Labels {
					if mf.observabilityLabels.Test(k) {
						observabilityMetric.Labels[k] = v
					}
				}
			}

			observabilityMetrics = append(observabilityMetrics, observabilityMetric)
		}
	}

	return costMetrics, observabilityMetrics
}
