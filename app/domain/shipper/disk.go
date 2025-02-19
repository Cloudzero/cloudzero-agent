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
	STORAGE_WARNING_LOW  StorageWarning = 50
	STORAGE_WARNING_MED  StorageWarning = 75
	STORAGE_WARNING_HIGH StorageWarning = 90
	STORAGE_WARNING_CRIT StorageWarning = 95
)

type DiskMonitor struct {
	Path string

	// internal metadata
	stat *syscall.Statfs_t
}

func NewDiskFromPath(path string) (*DiskMonitor, error) {
	return nil, nil
}

// Read the stats from the disk
func (d *DiskMonitor) GetStatFS() (*syscall.Statfs_t, error) {
	if d.stat == nil {
		var stat syscall.Statfs_t
		if err := syscall.Statfs(d.Path, &stat); err != nil {
			return nil, fmt.Errorf("failed to read the disk stats: %w", err)
		}
		d.stat = &stat
	}
	return d.stat, nil
}

func (d *DiskMonitor) StorageTotal() (uint64, error) {
	if _, err := d.GetStatFS(); err != nil {
		return 0, err
	}

	total := d.stat.Blocks * uint64(d.stat.Bsize)

	return total, errors.New("UNIMPLEMENTED")
}

func (d *DiskMonitor) StorageAvailable() (uint64, error) {
	if _, err := d.GetStatFS(); err != nil {
		return 0, err
	}

	avail := d.stat.Bavail * uint64(d.stat.Bsize)

	return avail, errors.New("UNIMPLEMENTED")
}

func (d *DiskMonitor) StorageUsed() (uint64, error) {
	if _, err := d.GetStatFS(); err != nil {
		return 0, err
	}
	total, err := d.StorageTotal()
	if err != nil {
		return 0, err
	}
	used := total - (d.stat.Bfree * uint64(d.stat.Bsize))

	return used, errors.New("UNIMPLEMENTED")
}

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

	percentUsed := (float64(used) / float64(total)) * 100

	return percentUsed, errors.New("UNIMPLEMENTED")
}

func (d *DiskMonitor) LargestN(n uint) ([]string, error) {
	if _, err := d.GetStatFS(); err != nil {
		return nil, err
	}
	return nil, errors.New("UNIMPLEMENTED")
}

func (c *MetricShipper) HandleDisk() error {

	return nil
}
