// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

import (
	"errors"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShipper_DiskMonitor_StorageCalculations(t *testing.T) {
	tests := []struct {
		name          string
		mockStat      *syscall.Statfs_t
		expectedTotal uint64
		expectedUsed  uint64
		expectedAvail uint64
		expectedPct   float64
	}{
		{
			name: "Normal case",
			mockStat: &syscall.Statfs_t{
				Blocks: 1000,
				Bfree:  200,
				Bavail: 150,
				Bsize:  1024,
			},
			expectedTotal: 1000 * 1024,
			expectedUsed:  800 * 1024,
			expectedAvail: 150 * 1024,
			expectedPct:   80.0,
		},
		{
			name: "Near empty disk",
			mockStat: &syscall.Statfs_t{
				Blocks: 1000,
				Bfree:  900,
				Bavail: 850,
				Bsize:  1024,
			},
			expectedTotal: 1000 * 1024,
			expectedUsed:  100 * 1024,
			expectedAvail: 850 * 1024,
			expectedPct:   10.0,
		},
		{
			name: "Near full disk",
			mockStat: &syscall.Statfs_t{
				Blocks: 1000,
				Bfree:  50,
				Bavail: 40,
				Bsize:  1024,
			},
			expectedTotal: 1000 * 1024,
			expectedUsed:  950 * 1024,
			expectedAvail: 40 * 1024,
			expectedPct:   95.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitor := &DiskMonitor{
				stat: tt.mockStat,
			}

			stat, err := monitor.GetStatFS()
			require.NotNil(t, stat)
			require.NoError(t, err)

			total, err := monitor.StorageTotal()
			require.NoError(t, err)
			require.Equal(t, tt.expectedTotal, total)

			used, err := monitor.StorageUsed()
			require.NoError(t, err)
			require.Equal(t, tt.expectedUsed, used)

			avail, err := monitor.StorageAvailable()
			require.NoError(t, err)
			require.Equal(t, tt.expectedAvail, avail)

			pct, err := monitor.StoragePercentUsed()
			require.NoError(t, err)
			require.Equal(t, tt.expectedPct, pct)
		})
	}
}

func TestShipper_DiskMonitor_ErrorHandling(t *testing.T) {
	monitor := &DiskMonitor{}

	// override statfs to throw an error
	origStatFs := statfs
	defer func() { statfs = origStatFs }()

	statfs = func(path string, stat *syscall.Statfs_t) (err error) {
		return errors.New("mock error")
	}

	_, err := monitor.GetStatFS()
	require.Error(t, err)

	_, err = monitor.StorageTotal()
	require.Error(t, err)

	_, err = monitor.StorageAvailable()
	require.Error(t, err)

	_, err = monitor.StorageUsed()
	require.Error(t, err)

	_, err = monitor.StoragePercentUsed()
	require.Error(t, err)
}
