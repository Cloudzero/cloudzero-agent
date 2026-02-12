// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package types defines storage monitoring types for disk usage tracking and alerting.
package types

// StoreWarning represents storage usage alert thresholds based on percentage utilization.
// These thresholds are used to determine when storage systems require attention
// to prevent data loss or collection failures in the CloudZero Agent.
type StoreWarning int

// Storage usage warning thresholds as percentage values.
// These constants define the escalation levels for storage monitoring alerts.
var (
	// StoreWarningNone indicates normal storage usage (below 50% utilization).
	StoreWarningNone StoreWarning = 49

	// StoreWarningLow indicates elevated storage usage (50-64% utilization).
	// This level suggests monitoring should be increased but no immediate action required.
	StoreWarningLow StoreWarning = 50

	// StoreWarningMed indicates moderate storage pressure (65-79% utilization).
	// This level may require cleanup of old data or storage expansion planning.
	StoreWarningMed StoreWarning = 65

	// StoreWarningHigh indicates high storage usage (80-89% utilization).
	// This level requires immediate attention to prevent collection disruption.
	StoreWarningHigh StoreWarning = 80

	// StoreWarningCrit indicates critical storage usage (90%+ utilization).
	// This level may cause metric collection failures and requires emergency response.
	StoreWarningCrit StoreWarning = 90
)

// StoreUsage contains comprehensive storage utilization information for monitoring disk usage.
// This structure provides the data needed to calculate warning levels and make
// storage management decisions in the CloudZero Agent.
type StoreUsage struct {
	// Total is the total storage capacity in bytes of the monitored filesystem.
	Total uint64 `json:"total"`

	// Available is the available storage space in bytes for new data.
	Available uint64 `json:"available"`

	// Used is the currently utilized storage in bytes, computed as Total - Available.
	Used uint64 `json:"used"`

	// PercentUsed is the utilization percentage, computed as (Used / Total) * 100.
	// This field is used with StoreWarning thresholds to determine alert levels.
	PercentUsed float64 `json:"percentUsed"`

	// BlockSize is the underlying filesystem block size in bytes for storage efficiency calculations.
	BlockSize uint32 `json:"blockSize"`
}

// GetStorageWarning evaluates the current storage usage against warning thresholds.
// Returns the appropriate StoreWarning level based on percentage utilization,
// enabling the monitoring system to take appropriate actions for storage management.
func (du *StoreUsage) GetStorageWarning() StoreWarning {
	percentUsed := StoreWarning(du.PercentUsed)

	switch {
	case percentUsed >= StoreWarningCrit:
		return StoreWarningCrit
	case percentUsed >= StoreWarningHigh:
		return StoreWarningHigh
	case percentUsed >= StoreWarningMed:
		return StoreWarningMed
	case percentUsed >= StoreWarningLow:
		return StoreWarningLow
	default:
		return StoreWarningNone
	}
}

// StoreMonitor is a generic interface for reporting on the usage of a store
type StoreMonitor interface {
	// GetUsage returns a complete snapshot of the store usage.
	// optional `paths` can be defined which will be used as `filepath.Join(paths...)`
	GetUsage(limit uint64, paths ...string) (*StoreUsage, error)
}
