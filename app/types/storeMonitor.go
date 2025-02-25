// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package types

type StoreWarning uint

var (
	StorageWarningNone StoreWarning = 49
	StorageWarningLow  StoreWarning = 50
	StorageWarningMed  StoreWarning = 65
	StorageWarningHigh StoreWarning = 80
	StorageWarningCrit StoreWarning = 90
)

// StoreUsage stores information about the current state of a store
type StoreUsage struct {
	Total          uint64  `json:"total"`          // Total storage in bytes
	Available      uint64  `json:"available"`      // Available storage in bytes
	Used           uint64  `json:"used"`           // Computed as Total - Available
	PercentUsed    float64 `json:"percentUsed"`    // Computed as (Used / Total) * 100
	BlockSize      uint32  `json:"blockSize"`      // Underlying block size
	Reserved       uint64  `json:"reserved"`       // Reserved space for system use in bytes
	InodeTotal     uint64  `json:"inodeTotal"`     // Total number of inodes
	InodeUsed      uint64  `json:"inodeUsed"`      // Inodes currently in use
	InodeAvailable uint64  `json:"inodeAvailable"` // Available inodes
}

// GetStorageWarning uses the `PercentUsed` field to calculate the current warning state of a store
func (du *StoreUsage) GetStorageWarning() StoreWarning {
	percentUsed := StoreWarning(du.PercentUsed)

	switch {
	case percentUsed >= StorageWarningCrit:
		return StorageWarningCrit
	case percentUsed >= StorageWarningHigh:
		return StorageWarningHigh
	case percentUsed >= StorageWarningMed:
		return StorageWarningMed
	case percentUsed >= StorageWarningLow:
		return StorageWarningLow
	default:
		return StorageWarningNone
	}
}

// StoreMonitor is a generic interface for reporting on the usage of a store
type StoreMonitor interface {
	// GetUsage returns a complete snapshot of the store usage
	GetUsage() (*StoreUsage, error)
}
