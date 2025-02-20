// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

import (
	"errors"
	"fmt"

	"github.com/cloudzero/cloudzero-insights-controller/app/types"
	"github.com/rs/zerolog/log"
)

func (m *MetricShipper) HandleDisk() error {
	// get the disk usage
	usage, err := m.lister.GetUsage()
	if err != nil {
		return fmt.Errorf("failed to get the usage: %w", err)
	}

	// report all of the metrics
	metricDiskTotalSizeBytes.WithLabelValues().Set(float64(usage.Total))
	metricCurrentDiskUsageBytes.WithLabelValues().Set(float64(usage.Used))
	metricCurrentDiskUsagePercentage.WithLabelValues().Set(usage.PercentUsed)

	// read file counts
	unsent, err := m.lister.GetFiles()
	if err != nil {
		return fmt.Errorf("failed to get the unsent files: %w", err)
	}
	sent, err := m.lister.GetMatching(m.setting.Database.StorageUploadSubpath, nil)
	if err != nil {
		return fmt.Errorf("failed to get the uploaded files; %w", err)
	}
	rr, err := m.lister.GetMatching(replaySubdirName, nil)
	if err != nil {
		return fmt.Errorf("failed to get the replay request files: %w", err)
	}

	// set the file metrics
	metricCurrentDiskUnsentFileCount.WithLabelValues().Set(float64(len(unsent)))
	metricCurrentDiskSentFileCount.WithLabelValues().Set(float64(len(sent)))
	metricCurrentDiskReplayRequestCount.WithLabelValues().Set(float64(len(rr)))

	// get the storage warning level
	warn := usage.GetStorageWarning()

	switch warn {
	case types.StorageWarningNone:
		log.Ctx(m.ctx).Debug().
			Uint64("total", usage.Total).
			Uint64("used", usage.Used).
			Float64("percentUsed", usage.PercentUsed).
			Uint64("total", usage.Total).
			Msg("No storage level warning")
	case types.StorageWarningLow:
		log.Ctx(m.ctx).Debug().
			Uint64("total", usage.Total).
			Uint64("used", usage.Used).
			Float64("percentUsed", usage.PercentUsed).
			Uint64("total", usage.Total).
			Msg("Low storage level warning")
	case types.StorageWarningMed:
		log.Ctx(m.ctx).Info().
			Uint64("total", usage.Total).
			Uint64("used", usage.Used).
			Float64("percentUsed", usage.PercentUsed).
			Uint64("total", usage.Total).
			Msg("Medium storage level warning")
		if err := m.HandleStorageWarningMedium(); err != nil {
			return err
		}
	case types.StorageWarningHigh:
		log.Ctx(m.ctx).Warn().
			Uint64("total", usage.Total).
			Uint64("used", usage.Used).
			Float64("percentUsed", usage.PercentUsed).
			Uint64("total", usage.Total).
			Msg("High storage level warning")
		if err := m.HandleStorageWarningHigh(); err != nil {
			return err
		}
	case types.StorageWarningCrit:
		log.Ctx(m.ctx).Error().
			Uint64("total", usage.Total).
			Uint64("used", usage.Used).
			Float64("percentUsed", usage.PercentUsed).
			Uint64("total", usage.Total).
			Msg("Critical storage level warning")
		if err := m.HandleStorageWarningCritical(); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown storage warning level: %d", warn)
	}

	return nil
}

func (m *MetricShipper) HandleStorageWarningMedium() error {
	return errors.ErrUnsupported
}

func (m *MetricShipper) HandleStorageWarningHigh() error {
	return errors.ErrUnsupported
}

func (m *MetricShipper) HandleStorageWarningCritical() error {
	return errors.ErrUnsupported
}
