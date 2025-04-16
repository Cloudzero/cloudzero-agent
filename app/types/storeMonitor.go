// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package types

type StoreWarning int

var (
	StoreWarningNone StoreWarning = 49
	StoreWarningLow  StoreWarning = 50
	StoreWarningMed  StoreWarning = 65
	StoreWarningHigh StoreWarning = 80
	StoreWarningCrit StoreWarning = 90
)

// StoreUsage stores information about the current state of a store
type StoreUsage struct {
	Total       uint64  `json:"total"`       // Total storage in bytes
	Available   uint64  `json:"available"`   // Available storage in bytes
	Used        uint64  `json:"used"`        // Computed as Total - Available
	PercentUsed float64 `json:"percentUsed"` // Computed as (Used / Total) * 100
	BlockSize   uint32  `json:"blockSize"`   // Underlying block size
}

// GetStorageWarning uses the `PercentUsed` field to calculate the current warning state of a store
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
