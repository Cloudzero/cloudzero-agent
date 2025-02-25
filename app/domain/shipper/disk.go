// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/cloudzero/cloudzero-insights-controller/app/types"
	"github.com/rs/zerolog/log"
)

// var metricCutoffTime = time.Now().AddDate(0, 0, -90) // date where metrics will start to be purged based on normal cleanup operations

func (m *MetricShipper) HandleDisk(metricCutoff time.Time) error {
	// get the disk usage
	usage, err := m.GetDiskUsage()
	if err != nil {
		return err
	}

	// get the storage warning level
	warn := usage.GetStorageWarning()

	switch warn {
	case types.StorageWarningNone:
		fallthrough
	case types.StorageWarningLow:
		log.Ctx(m.ctx).Debug().
			Uint64("total", usage.Total).
			Uint64("used", usage.Used).
			Float64("percentUsed", usage.PercentUsed).
			Uint64("total", usage.Total).
			Any("warningLevel", warn).
			Msg("storage level warning")

		// do nothing to the disk at this point
		return nil
	case types.StorageWarningMed:
		log.Ctx(m.ctx).Info().
			Uint64("total", usage.Total).
			Uint64("used", usage.Used).
			Float64("percentUsed", usage.PercentUsed).
			Uint64("total", usage.Total).
			Msg("Medium storage level warning")
		if err := m.handleStorageWarningMedium(metricCutoff); err != nil {
			return err
		}
	case types.StorageWarningHigh:
		log.Ctx(m.ctx).Warn().
			Uint64("total", usage.Total).
			Uint64("used", usage.Used).
			Float64("percentUsed", usage.PercentUsed).
			Uint64("total", usage.Total).
			Msg("High storage level warning")
		if err := m.handleStorageWarningHigh(metricCutoff); err != nil {
			return err
		}
	case types.StorageWarningCrit:
		log.Ctx(m.ctx).Error().
			Uint64("total", usage.Total).
			Uint64("used", usage.Used).
			Float64("percentUsed", usage.PercentUsed).
			Uint64("total", usage.Total).
			Msg("Critical storage level warning")
		if err := m.handleStorageWarningCritical(metricCutoff); err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown storage warning level: %d", warn)
	}

	return nil
}

// GetDiskUsage gets the storage usage of the attached volume, and also reports
// the usage to prometheus.
func (m *MetricShipper) GetDiskUsage() (*types.StoreUsage, error) {
	// get the disk usage
	usage, err := m.lister.GetUsage()
	if err != nil {
		return nil, fmt.Errorf("failed to get the usage: %w", err)
	}

	// report all of the metrics
	metricDiskTotalSizeBytes.WithLabelValues().Set(float64(usage.Total))
	metricCurrentDiskUsageBytes.WithLabelValues().Set(float64(usage.Used))
	metricCurrentDiskUsagePercentage.WithLabelValues().Set(usage.PercentUsed)

	// read file counts
	unsent, err := m.lister.GetFiles()
	if err != nil {
		return nil, fmt.Errorf("failed to get the unsent files: %w", err)
	}
	sent, err := m.lister.GetFiles(UploadedSubDirectory)
	if err != nil {
		return nil, fmt.Errorf("failed to get the uploaded files; %w", err)
	}
	rr, err := m.lister.GetFiles(ReplaySubDirectory)
	if err != nil {
		return nil, fmt.Errorf("failed to get the replay request files: %w", err)
	}

	// set the file metrics
	metricCurrentDiskUnsentFile.WithLabelValues().Set(float64(len(unsent)))
	metricCurrentDiskSentFile.WithLabelValues().Set(float64(len(sent)))
	metricCurrentDiskReplayRequest.WithLabelValues().Set(float64(len(rr)))

	return usage, nil
}

func (m *MetricShipper) handleStorageWarningMedium(before time.Time) error {
	return m.PurgeMetricsBefore(before)
}

func (m *MetricShipper) handleStorageWarningHigh(before time.Time) error {
	return m.PurgeMetricsBefore(before) // TODO -- add more aggressive cleanup
}

func (m *MetricShipper) handleStorageWarningCritical(before time.Time) error {
	return m.PurgeMetricsBefore(before) // TODO -- add more aggressive cleanup
}

// PurgeMetricsBefore deletes all uploaded metric files older than `metricCutoffTime`
func (m *MetricShipper) PurgeMetricsBefore(before time.Time) error {
	oldFiles := make([]string, 0)
	if err := m.lister.Walk(UploadedSubDirectory, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("unknown error walking directory: %w", err)
		}

		// ignore dirs (i.e. not recurrsive)
		if info.IsDir() {
			return nil
		}

		// compare the file
		if info.ModTime().Before(before) {
			oldFiles = append(oldFiles, path)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("failed to walk the filestore: %w", err)
	}

	if len(oldFiles) == 0 {
		return nil
	}

	// delete all files
	for _, file := range oldFiles {
		if err := os.Remove(file); err != nil {
			return fmt.Errorf("failed to delete the file: %w", err)
		}
	}

	log.Ctx(m.ctx).Info().Int("numFiles", len(oldFiles)).Msg("Successfully purged old files")

	return nil
}

func (m *MetricShipper) PurgeUntilSize(size uint64) error {
	return errors.ErrUnsupported
}
