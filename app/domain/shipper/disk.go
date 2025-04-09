// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/cloudzero/cloudzero-agent/app/instr"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/rs/zerolog"
)

func (m *MetricShipper) HandleDisk(ctx context.Context, metricCutoff time.Time) error {
	return m.metrics.SpanCtx(ctx, "shipper_HandleDisk", func(ctx context.Context, id string) error {
		logger := instr.SpanLogger(ctx, id,
			func(ctx zerolog.Context) zerolog.Context {
				return ctx.Time("metricCutoff", metricCutoff)
			},
		)
		logger.Debug().Msg("Handling the disk usage ...")

		// get the size limit from the config
		sizeLimitBytes, err := m.setting.GetAvailableSizeBytes()
		if err != nil {
			return fmt.Errorf("failed to get the size_limit from the config: %w", err)
		}

		// get the disk usage
		usage, err := m.GetDiskUsage(ctx, sizeLimitBytes)
		if err != nil {
			return err
		}

		// get the storage warning level
		logger.Debug().Msg("Getting the storage warning")
		warn := usage.GetStorageWarning()

		// log the storage warning
		logger.Debug().
			Uint64("used", usage.Used).
			Float64("percentUsed", usage.PercentUsed).
			Uint64("total", usage.Total).
			Any("warningLevel", warn).
			Msg("Current storage usage")

		switch warn {
		case types.StoreWarningNone:
			fallthrough
		case types.StoreWarningLow:
			fallthrough
		case types.StoreWarningMed:
			// purge old metrics if not in lazy mode
			if !m.setting.Database.PurgeRules.Lazy {
				if err = m.PurgeMetricsBefore(ctx, metricCutoff); err != nil {
					metricDiskCleanupFailureTotal.WithLabelValues(strconv.Itoa(int(warn)), GetErrStatusCode(err)).Inc()
					return ErrStorageCleanup
				}
			}
		case types.StoreWarningHigh:
			if err = m.handleStorageWarningHigh(ctx, metricCutoff); err != nil {
				// note the error in prom
				metricDiskCleanupFailureTotal.WithLabelValues(strconv.Itoa(int(warn)), GetErrStatusCode(err)).Inc()
				return ErrStorageCleanup
			}

			// note the success in prom
			metricDiskCleanupSuccessTotal.WithLabelValues(strconv.Itoa(int(warn))).Inc()

		case types.StoreWarningCrit:
			if err = m.handleStorageWarningCritical(ctx); err != nil {
				// note the error in prom
				metricDiskCleanupFailureTotal.WithLabelValues(strconv.Itoa(int(warn)), GetErrStatusCode(err)).Inc()
				return ErrStorageCleanup
			}

			// note the success in prom
			metricDiskCleanupSuccessTotal.WithLabelValues(strconv.Itoa(int(warn))).Inc()

		default:
			return fmt.Errorf("unknown storage warning level: %d", warn)
		}

		// fetch the usage again
		usage2, err := m.GetDiskUsage(ctx, sizeLimitBytes)
		if err != nil {
			return ErrGetDiskUsage
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
func (m *MetricShipper) GetDiskUsage(ctx context.Context, limit uint64) (*types.StoreUsage, error) {
	var usage *types.StoreUsage

	err := m.metrics.SpanCtx(ctx, "shipper_GetDiskUsage", func(ctx context.Context, id string) error {
		logger := instr.SpanLogger(ctx, id)
		logger.Debug().Msg("Fetching disk info")
		var err error

		// get the disk usage
		usage, err = m.store.GetUsage(limit)
		if err != nil {
			return ErrGetDiskUsage
		}

		// report all of the metrics
		metricDiskTotalSizeBytes.WithLabelValues().Set(float64(usage.Total))
		metricCurrentDiskUsageBytes.WithLabelValues().Set(float64(usage.Used))
		metricCurrentDiskUsagePercentage.WithLabelValues().Set(usage.PercentUsed)

		// read file counts
		unsent, err := m.store.GetFiles()
		if err != nil {
			return errors.Join(ErrFilesList, fmt.Errorf("failed to get the unsent files: %w", err))
		}
		sent, err := m.store.GetFiles(UploadedSubDirectory)
		if err != nil {
			return errors.Join(ErrFilesList, fmt.Errorf("failed to get the uploaded files: %w", err))
		}
		rr, err := m.store.GetFiles(ReplaySubDirectory)
		if err != nil {
			return errors.Join(ErrFilesList, fmt.Errorf("failed to get the replay request files: %w", err))
		}

		// set the file metrics
		metricCurrentDiskUnsentFile.WithLabelValues().Set(float64(len(unsent)))
		metricCurrentDiskSentFile.WithLabelValues().Set(float64(len(sent)))
		metricCurrentDiskReplayRequest.WithLabelValues().Set(float64(len(rr)))

		logger.Debug().Msg("Successfully fetched disk usage")

		return nil
	})
	if err != nil {
		return nil, err
	}

	return usage, nil
}

func (m *MetricShipper) handleStorageWarningHigh(ctx context.Context, before time.Time) error {
	return m.PurgeMetricsBefore(ctx, before)
}

func (m *MetricShipper) handleStorageWarningCritical(ctx context.Context) error {
	return m.PurgeOldestNPercentage(ctx, m.setting.Database.PurgeRules.Percent)
}

// PurgeMetricsBefore deletes all uploaded metric files older than `before`
func (m *MetricShipper) PurgeMetricsBefore(ctx context.Context, before time.Time) error {
	return m.metrics.SpanCtx(ctx, "shipper_PurgeMetricsBefore", func(ctx context.Context, id string) error {
		logger := instr.SpanLogger(ctx, id)
		logger.Debug().Msg("Purging old metrics")

		oldFiles := make([]string, 0)
		if err := m.store.Walk(UploadedSubDirectory, func(path string, info fs.FileInfo, err error) error {
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
			return errors.Join(ErrFilesList, fmt.Errorf("failed to walk the uploaded filestore: %w", err))
		}

		if len(oldFiles) == 0 {
			logger.Debug().Msg("No files to purge found")
			return nil
		}

		// delete all files
		for _, file := range oldFiles {
			if err := os.Remove(file); err != nil {
				return errors.Join(ErrFileRemove, fmt.Errorf("failed to delete a file during disk cleanup: file=%s, err=%w", file, err))
			}
		}

		logger.Debug().Int("numFiles", len(oldFiles)).Msg("Successfully purged old files")

		return nil
	})
}

// PurgeOldestNPercentage removes the oldest `percent` of files
func (m *MetricShipper) PurgeOldestNPercentage(ctx context.Context, percent int) error {
	return m.metrics.SpanCtx(ctx, "shipper_PurgeOldestNPercentage", func(ctx context.Context, id string) error {
		logger := instr.SpanLogger(ctx, id,
			func(ctx zerolog.Context) zerolog.Context {
				return ctx.Int("percentage", percent)
			},
		)
		logger.Debug().Msg("Purging oldest percentage of files")

		if percent <= 0 || percent > 100 {
			return fmt.Errorf("invalid percentage: %d (must be between 1-100)", percent)
		}

		entries, err := m.store.ListFiles(UploadedSubDirectory)
		if err != nil {
			return errors.Join(ErrFilesList, fmt.Errorf("failed to list the uploaded files: %w", err))
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
			logger.Debug().Msg("No files to purge found")
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
				return errors.Join(ErrFileRemove, fmt.Errorf("failed to remove the file during a file purge: file=%s, err=%w", item, err))
			}
		}

		logger.Debug().
			Int("numFiles", n).
			Int("totalFiles", len(files)).
			Msg("Successfully purged oldest percentage of files")

		return nil
	})
}
