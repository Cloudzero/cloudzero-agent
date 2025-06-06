// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

import (
	"bufio"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/cloudzero/cloudzero-agent/app/logging/instr"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/rs/zerolog/log"
)

// Linux filesystem type constants
const (
	TmpfsMagic = 0x01021994
	Ext4Magic  = 0xEF53
	XfsMagic   = 0x58465342
)

// DiskManager handles disk usage monitoring and cleanup
type DiskManager struct {
	Store              types.ReadableStore
	Metrics            *instr.PrometheusMetrics
	StoragePath        string
	AvailableSizeBytes uint64
}

// PressureThresholds defines cleanup trigger points
type PressureThresholds struct {
	Critical float64
	High     float64
	Medium   float64
	Low      float64
}

// PressureLevel defines how aggressive cleanup should be
type PressureLevel int

const (
	PressureNone PressureLevel = iota
	PressureLow
	PressureMedium
	PressureHigh
	PressureCritical
)

// CleanupResult tracks what was cleaned up
type CleanupResult struct {
	FilesRemoved   int
	BytesFreed     uint64
	PressureBefore PressureLevel
	PressureAfter  PressureLevel
}

// HandleDisk is the main entry point for disk management
func (m *MetricShipper) HandleDisk(ctx context.Context, metricCutoff time.Time) error {
	size, _ := m.setting.GetAvailableSizeBytes()
	dm := &DiskManager{
		Store:              m.store,
		Metrics:            m.metrics,
		StoragePath:        m.setting.Database.StoragePath,
		AvailableSizeBytes: size,
	}

	return dm.ManageDiskUsage(ctx, metricCutoff)
}

// ManageDiskUsage handles the complete disk management cycle
func (dm *DiskManager) ManageDiskUsage(ctx context.Context, metricCutoff time.Time) error {
	return dm.Metrics.SpanCtx(ctx, "shipper_disk_manager_ManageDiskUsage", func(ctx context.Context, id string) error {
		logger := instr.SpanLogger(ctx, id)

		// Get current usage and pressure level
		usage, err := dm.getCurrentUsage(ctx)
		if err != nil {
			return fmt.Errorf("failed to get disk usage: %w", err)
		}

		pressure := dm.CalculatePressureLevel(usage)
		logger.Debug().
			Float64("percentUsed", usage.PercentUsed).
			Any("pressureLevel", pressure).
			Msg("Current disk pressure")

		// No cleanup needed
		if pressure == PressureNone || pressure == PressureLow {
			return nil
		}

		// Execute cleanup strategy based on pressure
		result, err := dm.executeCleanupStrategy(ctx, pressure, metricCutoff)
		if err != nil {
			return fmt.Errorf("cleanup failed: %w", err)
		}

		logger.Debug().
			Int("filesRemoved", result.FilesRemoved).
			Uint64("bytesFreed", result.BytesFreed).
			Any("pressureBefore", result.PressureBefore).
			Any("pressureAfter", result.PressureAfter).
			Msg("Cleanup completed")

		return nil
	})
}

// executeCleanupStrategy runs cleanup with escalating aggression
func (dm *DiskManager) executeCleanupStrategy(ctx context.Context, initialPressure PressureLevel, cutoff time.Time) (*CleanupResult, error) {
	result := &CleanupResult{PressureBefore: initialPressure}

	// Strategy 1: Remove files older than cutoff
	if initialPressure >= PressureMedium {
		removed, err := dm.PurgeFilesBefore(ctx, cutoff)
		if err != nil {
			return nil, fmt.Errorf("time-based purge failed: %w", err)
		}
		result.FilesRemoved += removed
	}

	// Check if we still have pressure after time-based cleanup
	usage, err := dm.getCurrentUsage(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get usage after cleanup: %w", err)
	}

	currentPressure := dm.CalculatePressureLevel(usage)

	// Strategy 2: If still under pressure, remove oldest percentage
	if currentPressure >= PressureHigh {
		percent := dm.GetCleanupPercentage(currentPressure)
		removed, err := dm.PurgeOldestPercentage(ctx, percent)
		if err != nil {
			return nil, fmt.Errorf("percentage-based purge failed: %w", err)
		}
		result.FilesRemoved += removed

		// Final pressure check
		usage, err = dm.getCurrentUsage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get final usage: %w", err)
		}
		currentPressure = dm.CalculatePressureLevel(usage)
	}

	result.PressureAfter = currentPressure
	return result, nil
}

// CalculatePressureLevel determines cleanup aggression needed
func (dm *DiskManager) CalculatePressureLevel(usage *types.StoreUsage) PressureLevel {
	thresholds := dm.getThresholds()

	switch {
	case usage.PercentUsed >= thresholds.Critical:
		return PressureCritical
	case usage.PercentUsed >= thresholds.High:
		return PressureHigh
	case usage.PercentUsed >= thresholds.Medium:
		return PressureMedium
	case usage.PercentUsed >= thresholds.Low:
		return PressureLow
	default:
		return PressureNone
	}
}

// GetCleanupPercentage returns how much to clean based on pressure
func (dm *DiskManager) GetCleanupPercentage(pressure PressureLevel) int {
	isMemory := dm.IsMemoryBacked()

	switch pressure {
	case PressureCritical:
		if isMemory {
			return 70 //nolint:revive // keep magic number
		}
		return 50 //nolint:revive // keep magic number
	case PressureHigh:
		if isMemory {
			return 50 //nolint:revive // keep magic number
		}
		return 25 //nolint:revive // keep magic number
	default:
		if isMemory {
			return 30 //nolint:revive // keep magic number
		}
		return 10
	}
}

// PurgeFilesBefore removes files older than the cutoff time
func (dm *DiskManager) PurgeFilesBefore(ctx context.Context, before time.Time) (int, error) {
	var res int
	err := dm.Metrics.SpanCtx(ctx, "shipper_disk_manager_purgeFilesBefore", func(ctx context.Context, id string) error {
		logger := instr.SpanLogger(ctx, id)

		var filesToRemove []string

		// Walk the uploaded directory to find old files
		uploadDir := filepath.Join(dm.StoragePath, UploadedSubDirectory)
		err := filepath.WalkDir(uploadDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}

			info, err := d.Info()
			if err != nil {
				return err
			}

			if info.ModTime().Before(before) {
				filesToRemove = append(filesToRemove, path)
			}

			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to walk directory: %w", err)
		}

		// Remove files
		removed := 0
		for _, file := range filesToRemove {
			if err := os.Remove(file); err != nil {
				logger.Warn().Err(err).Str("file", file).Msg("Failed to remove file")
				continue
			}
			removed++
		}

		logger.Debug().Int("removed", removed).Msg("Time-based cleanup completed")
		res = removed
		return nil
	})

	return res, err
}

// PurgeOldestPercentage removes the oldest percentage of files
func (dm *DiskManager) PurgeOldestPercentage(ctx context.Context, percent int) (int, error) {
	if percent < 0 || percent > 100 {
		return 0, fmt.Errorf("invalid percentage: %d (must be 0-100)", percent)
	}

	var res int
	err := dm.Metrics.SpanCtx(ctx, "shipper_disk_manager_purgeOldestPercentage", func(ctx context.Context, id string) error {
		logger := instr.SpanLogger(ctx, id)

		// Collect file info in a memory-efficient way
		type fileInfo struct {
			path    string
			modTime time.Time
		}

		var files []fileInfo

		uploadDir := filepath.Join(dm.StoragePath, UploadedSubDirectory)
		err := filepath.WalkDir(uploadDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}

			info, err := d.Info()
			if err != nil {
				return err
			}

			files = append(files, fileInfo{
				path:    path,
				modTime: info.ModTime(),
			})

			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to walk directory: %w", err)
		}

		if len(files) == 0 {
			return nil
		}

		// Sort by modification time (oldest first)
		sort.Slice(files, func(i, j int) bool {
			return files[i].modTime.Before(files[j].modTime)
		})

		// Calculate how many files to remove
		numToRemove := (len(files) * percent) / 100
		if numToRemove == 0 && percent > 0 {
			numToRemove = 1 // Always remove at least one file if percentage > 0
		}

		// Remove the oldest files
		removed := 0
		for i := 0; i < numToRemove && i < len(files); i++ {
			if err := os.Remove(files[i].path); err != nil {
				logger.Warn().Err(err).Str("file", files[i].path).Msg("Failed to remove file")
				continue
			}
			removed++
		}

		logger.Debug().
			Int("removed", removed).
			Int("total", len(files)).
			Int("percent", percent).
			Msg("Percentage-based cleanup completed")

		res = removed
		return nil
	})

	return res, err
}

// getCurrentUsage gets current disk usage with metrics reporting
func (dm *DiskManager) getCurrentUsage(_ context.Context) (*types.StoreUsage, error) {
	usage, err := dm.Store.GetUsage(dm.AvailableSizeBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to get disk usage: %w", err)
	}

	// Report metrics
	metricDiskTotalSizeBytes.WithLabelValues().Set(float64(usage.Total))
	metricCurrentDiskUsageBytes.WithLabelValues().Set(float64(usage.Used))
	metricCurrentDiskUsagePercentage.WithLabelValues().Set(usage.PercentUsed)

	return usage, nil
}

// getThresholds returns appropriate thresholds based on storage backend
func (dm *DiskManager) getThresholds() PressureThresholds {
	if dm.IsMemoryBacked() {
		// Memory-backed storage: much more aggressive
		return PressureThresholds{
			Critical: 80, //nolint:revive // keep magic number
			High:     60, //nolint:revive // keep magic number
			Medium:   40, //nolint:revive // keep magic number
			Low:      20, //nolint:revive // keep magic number
		}
	}

	// Disk-backed storage: more conservative
	return PressureThresholds{
		Critical: 95, //nolint:revive // keep magic number
		High:     85, //nolint:revive // keep magic number
		Medium:   70, //nolint:revive // keep magic number
		Low:      50, //nolint:revive // keep magic number
	}
}

// IsMemoryBacked detects if the storage path is backed by memory (tmpfs)
func (dm *DiskManager) IsMemoryBacked() bool {
	// Primary method: Check filesystem type via statfs
	if isMemory, err := dm.checkFilesystemType(); err == nil {
		return isMemory
	}

	// Fallback method: Parse /proc/mounts
	if isMemory, err := dm.checkProcMounts(); err == nil {
		return isMemory
	}

	log.Warn().Str("path", dm.StoragePath).Msg("Failed to detect storage backend, assuming disk-backed")
	return false
}

// checkFilesystemType uses statfs syscall to determine filesystem type
func (dm *DiskManager) checkFilesystemType() (bool, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(dm.StoragePath, &stat); err != nil {
		return false, fmt.Errorf("statfs failed: %w", err)
	}

	// Check if filesystem type is tmpfs
	fsType := stat.Type
	return fsType == TmpfsMagic, nil
}

func (dm *DiskManager) checkProcMounts() (bool, error) {
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return false, fmt.Errorf("failed to open /proc/mounts: %w", err)
	}
	defer file.Close()

	// Get absolute path for comparison
	absPath, err := filepath.Abs(dm.StoragePath)
	if err != nil {
		return false, fmt.Errorf("failed to get absolute path: %w", err)
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)

		if len(fields) < 3 {
			continue
		}

		mountPoint := fields[1]
		fsType := fields[2]

		// Check if our path is under this mount point and it's tmpfs
		if strings.HasPrefix(absPath, mountPoint) && fsType == "tmpfs" {
			return true, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("error reading /proc/mounts: %w", err)
	}

	return false, nil
}
