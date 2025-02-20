// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

import (
	"errors"
	"fmt"
	"syscall"
)

type StorageWarning uint

var (
	StorageWarningLow  StorageWarning = 50
	StorageWarningMed  StorageWarning = 75
	StorageWarningHigh StorageWarning = 90
	StorageWarningCrit StorageWarning = 95
)

// Add a variable to hold the Statfs function for mocking errors
var statfs = syscall.Statfs

type DiskMonitor struct {
	Path string

	// internal metadata
	stat *syscall.Statfs_t
}

func NewDiskFromPath(path string) (*DiskMonitor, error) {
	return nil, errors.ErrUnsupported
}

// Read the stats from the disk
func (d *DiskMonitor) GetStatFS() (*syscall.Statfs_t, error) {
	if d.stat == nil {
		var stat syscall.Statfs_t
		if err := statfs(d.Path, &stat); err != nil {
			return nil, fmt.Errorf("failed to read the disk stats: %w", err)
		}
		d.stat = &stat
	}
	return d.stat, nil
}

// Get total available storage in bytes
func (d *DiskMonitor) StorageTotal() (uint64, error) {
	if _, err := d.GetStatFS(); err != nil {
		return 0, err
	}

	total := d.stat.Blocks * uint64(d.stat.Bsize)

	return total, nil
}

// Get available (free) storage remaining in the disk in bytes
func (d *DiskMonitor) StorageAvailable() (uint64, error) {
	if _, err := d.GetStatFS(); err != nil {
		return 0, err
	}

	avail := d.stat.Bavail * uint64(d.stat.Bsize)

	return avail, nil
}

// Get the total storage used in bytes
func (d *DiskMonitor) StorageUsed() (uint64, error) {
	if _, err := d.GetStatFS(); err != nil {
		return 0, err
	}
	total, err := d.StorageTotal()
	if err != nil {
		return 0, err
	}
	used := total - (d.stat.Bfree * uint64(d.stat.Bsize))

	return used, nil
}

// Get the storage used as a percentage
//
// Runs (StorageUsed() / StorageTotal()) * 100
func (d *DiskMonitor) StoragePercentUsed() (float64, error) {
	if _, err := d.GetStatFS(); err != nil {
		return 0, err
	}

	total, err := d.StorageTotal()
	if err != nil {
		return 0, err
	}
	used, err := d.StorageUsed()
	if err != nil {
		return 0, err
	}

	percentUsed := (float64(used) / float64(total)) * 100 //nolint:revive // ?? its just a fraction bro

	return percentUsed, nil
}

// Returns the largest `n` files
func (d *DiskMonitor) LargestN(n uint) ([]string, error) {
	if _, err := d.GetStatFS(); err != nil {
		return nil, err
	}
	return nil, errors.ErrUnsupported
}

// Returns the oldest `n` files
func (d *DiskMonitor) OldestN(n uint) ([]string, error) {
	if _, err := d.GetStatFS(); err != nil {
		return nil, err
	}
	return nil, errors.ErrUnsupported
}

func (m *MetricShipper) HandleDisk() error {
	return errors.ErrUnsupported
}
