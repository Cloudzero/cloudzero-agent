// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/cloudzero/cloudzero-insights-controller/app/types"
	"github.com/rs/zerolog/log"
)

func (m *MetricShipper) HandleDisk(metricCutoff time.Time) error {
	return m.metrics.Span("shipper_HandleDisk", func() error {
		// get the disk usage
		usage, err := m.GetDiskUsage()
		if err != nil {
			return err
		}

		// get the storage warning level
		warn := usage.GetStorageWarning()

		// log the storage warning
		log.Ctx(m.ctx).Info().
			Uint64("used", usage.Used).
			Float64("percentUsed", usage.PercentUsed).
			Uint64("total", usage.Total).
			Any("warningLevel", warn).
			Msg("Storage usage")

		switch warn {
		case types.StoreWarningNone:
			fallthrough
		case types.StoreWarningLow:
			fallthrough
		case types.StoreWarningMed:
			return nil
		case types.StoreWarningHigh:
			if err = m.handleStorageWarningHigh(metricCutoff); err != nil {
				// note the error in prom
				metricDiskCleanupFailureTotal.WithLabelValues(strconv.Itoa(int(warn)), err.Error()).Inc()
				return err
			}

			// note the success in prom
			metricDiskCleanupSuccessTotal.WithLabelValues(strconv.Itoa(int(warn))).Inc()

		case types.StoreWarningCrit:
			if err = m.handleStorageWarningCritical(); err != nil {
				// note the error in prom
				metricDiskCleanupFailureTotal.WithLabelValues(strconv.Itoa(int(warn)), err.Error()).Inc()
				return err
			}

			// note the success in prom
			metricDiskCleanupSuccessTotal.WithLabelValues(strconv.Itoa(int(warn))).Inc()

		default:
			return fmt.Errorf("unknown storage warning level: %d", warn)
		}

		// fetch the usage again
		usage2, err := m.lister.GetUsage()
		if err != nil {
			return fmt.Errorf("failed to get the disk usage: %w", err)
		}

		// log how much storage was purged
		changed := usage.PercentUsed - usage2.PercentUsed
		if changed != 0 {
			metricDiskCleanupPercentage.Observe(usage2.PercentUsed)
		}

		return nil
	})
}

// GetDiskUsage gets the storage usage of the attached volume, and also reports
// the usage to prometheus.
func (m *MetricShipper) GetDiskUsage() (*types.StoreUsage, error) {
	var usage *types.StoreUsage

	err := m.metrics.Span("shipper_GetDiskUsage", func() error {
		log.Ctx(m.ctx).Debug().Msg("Fetching disk info")
		var err error

		// get the disk usage
		usage, err = m.lister.GetUsage()
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
		sent, err := m.lister.GetFiles(UploadedSubDirectory)
		if err != nil {
			return fmt.Errorf("failed to get the uploaded files; %w", err)
		}
		rr, err := m.lister.GetFiles(ReplaySubDirectory)
		if err != nil {
			return fmt.Errorf("failed to get the replay request files: %w", err)
		}

		// set the file metrics
		metricCurrentDiskUnsentFile.WithLabelValues().Set(float64(len(unsent)))
		metricCurrentDiskSentFile.WithLabelValues().Set(float64(len(sent)))
		metricCurrentDiskReplayRequest.WithLabelValues().Set(float64(len(rr)))

		return nil
	})
	if err != nil {
		return nil, err
	}

	return usage, nil
}

func (m *MetricShipper) handleStorageWarningHigh(before time.Time) error {
	log.Ctx(m.ctx).Info().Msg("Handling high storage usage ...")
	return m.PurgeMetricsBefore(before)
}

func (m *MetricShipper) handleStorageWarningCritical() error {
	log.Ctx(m.ctx).Info().Msg("Handling critical storage usage ...")
	return m.PurgeOldestNPercentage(CriticalPurgePercent)
}

// PurgeMetricsBefore deletes all uploaded metric files older than `before`
func (m *MetricShipper) PurgeMetricsBefore(before time.Time) error {
	return m.metrics.Span("shipper_PurgeMetricsBefore", func() error {
		log.Ctx(m.ctx).Info().Msgf("Purging all metrics before: %s", before.String())
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
			log.Ctx(m.ctx).Info().Msg("No files to purge found")
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
	})
}

// PurgeOldestPercentage removes the oldest `percent` of files
func (m *MetricShipper) PurgeOldestNPercentage(percent int) error {
	return m.metrics.Span("shipper_PurgeOldestNPercentage", func() error {
		log.Ctx(m.ctx).Info().Msgf("Purging oldest %d percent of files", percent)

		if percent <= 0 || percent > 100 {
			return fmt.Errorf("invalid percentage: %d (must be between 1-100)", percent)
		}

		entries, err := m.lister.ListFiles(UploadedSubDirectory)
		if err != nil {
			return fmt.Errorf("failed to list files: %w", err)
		}

		type fileData struct {
			path    string
			modTime time.Time
		}
		files := make([]fileData, 0)

		// parse with path and modified time
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			files = append(files, fileData{path: filepath.Join(m.setting.Database.StoragePath, UploadedSubDirectory, entry.Name()), modTime: info.ModTime()})
		}

		if len(files) == 0 {
			log.Ctx(m.ctx).Info().Msg("No files to purge found")
			return nil
		}

		// sort by the mod time
		sort.Slice(files, func(i, j int) bool {
			return files[i].modTime.Before(files[j].modTime)
		})

		// calculate how many files to remove
		n := (len(files) * percent) / 100
		if n == 0 && percent > 0 && len(files) > 0 {
			n = 1 // remove one file if percentage is positive
		}

		// create the list of paths to remove
		toRemove := make([]string, n)
		for i := range n {
			toRemove[i] = files[i].path
		}

		// remove all these files
		for _, item := range toRemove {
			if err := os.Remove(item); err != nil {
				return fmt.Errorf("failed to remove file '%s': %w", item, err)
			}
		}

		log.Ctx(m.ctx).Info().
			Int("numFiles", n).
			Int("totalFiles", len(files)).
			Int("percentage", percent).
			Msg("Successfully purged oldest percentage of files")

		return nil
	})
}
