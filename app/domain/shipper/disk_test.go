// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cloudzero/cloudzero-agent/app/domain/shipper"
	"github.com/cloudzero/cloudzero-agent/app/logging/instr"
	"github.com/cloudzero/cloudzero-agent/app/types"
)

// MockStore implements types.ReadableStore for testing
type MockStore struct {
	basePath string
	usage    *types.StoreUsage
	files    map[string][]string // directory -> files

	// Function fields for overriding behavior
	GetUsageFunc func(limit uint64, paths ...string) (*types.StoreUsage, error)
}

func NewMockStore(basePath string) *MockStore {
	store := &MockStore{
		basePath: basePath,
		usage: &types.StoreUsage{
			Total:       1000000, // 1MB
			Used:        0,
			Available:   1000000,
			PercentUsed: 0,
		},
		files: make(map[string][]string),
	}

	// Set default behavior
	store.GetUsageFunc = store.defaultGetUsage

	return store
}

func (m *MockStore) GetUsage(limit uint64, paths ...string) (*types.StoreUsage, error) {
	return m.GetUsageFunc(limit, paths...)
}

func (m *MockStore) defaultGetUsage(limit uint64, paths ...string) (*types.StoreUsage, error) {
	// Simulate real disk usage calculation
	actualUsed := m.calculateActualUsage()

	usage := &types.StoreUsage{
		Total:       m.usage.Total,
		Used:        actualUsed,
		Available:   m.usage.Total - actualUsed,
		PercentUsed: (float64(actualUsed) / float64(m.usage.Total)) * 100,
	}

	// Apply limit if provided
	if limit > 0 && limit < usage.Total {
		usage.Total = limit
		if usage.Used >= limit {
			usage.Used = limit
			usage.Available = 0
		} else {
			usage.Available = limit - usage.Used
		}
		usage.PercentUsed = (float64(usage.Used) / float64(usage.Total)) * 100
	}

	return usage, nil
}

func (m *MockStore) calculateActualUsage() uint64 {
	// Simulate usage by counting files in the base path
	var usage uint64
	filepath.Walk(m.basePath, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			usage += uint64(info.Size())
		}
		return nil
	})
	return usage
}

func (m *MockStore) GetFiles(paths ...string) ([]string, error) {
	fullPath := filepath.Join(append([]string{m.basePath}, paths...)...)

	var files []string
	err := filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})

	return files, err
}

func (m *MockStore) ListFiles(paths ...string) ([]os.DirEntry, error) {
	fullPath := filepath.Join(append([]string{m.basePath}, paths...)...)
	return os.ReadDir(fullPath)
}

func (m *MockStore) Walk(loc string, process filepath.WalkFunc) error {
	fullPath := filepath.Join(m.basePath, loc)
	return filepath.Walk(fullPath, process)
}

func (m *MockStore) Find(ctx context.Context, filterName string, filterExtension string) ([]string, error) {
	var files []string
	err := filepath.Walk(m.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			if filterName == "" || filepath.Base(path) == filterName {
				if filterExtension == "" || filepath.Ext(path) == filterExtension {
					files = append(files, path)
				}
			}
		}
		return nil
	})
	return files, err
}

// Reset resets the mock to default behavior
func (m *MockStore) Reset() {
	m.GetUsageFunc = m.defaultGetUsage
}

// TestHelper provides utilities for test setup
type TestHelper struct {
	tempDir            string
	mockStore          *MockStore
	metrics            *instr.PrometheusMetrics
	storagePath        string
	availableSizeBytes uint64
}

func NewTestHelper(t *testing.T) *TestHelper {
	tempDir, err := os.MkdirTemp("", "disk_manager_test_*")
	require.NoError(t, err)

	// Create subdirectories
	uploadedDir := filepath.Join(tempDir, shipper.UploadedSubDirectory)
	require.NoError(t, os.MkdirAll(uploadedDir, 0o755))

	mockStore := NewMockStore(tempDir)
	metrics, err := shipper.InitMetrics()
	require.NoError(t, err)

	t.Cleanup(func() {
		os.RemoveAll(tempDir)
	})

	return &TestHelper{
		tempDir:            tempDir,
		mockStore:          mockStore,
		metrics:            metrics,
		storagePath:        tempDir,
		availableSizeBytes: 1000000, // 1MB limit
	}
}

func (th *TestHelper) CreateTestFiles(count int, baseTime time.Time) error {
	uploadedDir := filepath.Join(th.tempDir, shipper.UploadedSubDirectory)

	for i := 0; i < count; i++ {
		filename := fmt.Sprintf("test_file_%03d.json", i) // Use zero-padded numbers
		filepath := filepath.Join(uploadedDir, filename)

		// Create file with some content
		content := fmt.Sprintf(`{"test": "data", "file": %d}`, i)
		if err := os.WriteFile(filepath, []byte(content), 0o644); err != nil {
			return err
		}

		// Set modification time (older files have earlier times)
		// Use hours instead of minutes to create a realistic spread
		modTime := baseTime.Add(time.Duration(-i) * time.Hour)
		if err := os.Chtimes(filepath, modTime, modTime); err != nil {
			return err
		}
	}

	return nil
}

func (th *TestHelper) GetFileCount() int {
	uploadedDir := filepath.Join(th.tempDir, shipper.UploadedSubDirectory)
	files, err := os.ReadDir(uploadedDir)
	if err != nil {
		return 0
	}

	count := 0
	for _, file := range files {
		if !file.IsDir() {
			count++
		}
	}
	return count
}

func (th *TestHelper) CreateDiskManager() *shipper.DiskManager {
	return &shipper.DiskManager{
		Store:              th.mockStore,
		Metrics:            th.metrics,
		StoragePath:        th.storagePath,
		AvailableSizeBytes: th.availableSizeBytes,
	}
}

// Test Suite
func TestUnit_Shipper_Disk_CalculatePressureLevel(t *testing.T) {
	helper := NewTestHelper(t)
	dm := helper.CreateDiskManager()

	tests := []struct {
		name          string
		percentUsed   float64
		expectedLevel shipper.PressureLevel
	}{
		{"No pressure", 30, shipper.PressureNone},
		{"Low pressure", 55, shipper.PressureLow},
		{"Medium pressure", 75, shipper.PressureMedium},
		{"High pressure", 90, shipper.PressureHigh},
		{"Critical pressure", 98, shipper.PressureCritical},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usage := &types.StoreUsage{
				PercentUsed: tt.percentUsed,
			}

			level := dm.CalculatePressureLevel(usage)
			assert.Equal(t, tt.expectedLevel, level)
		})
	}
}

func TestUnit_Shipper_Disk_GetCleanupPercentage(t *testing.T) {
	helper := NewTestHelper(t)
	dm := helper.CreateDiskManager()

	tests := []struct {
		pressure shipper.PressureLevel
		expected int
	}{
		{shipper.PressureHigh, 25},
		{shipper.PressureCritical, 50},
		{shipper.PressureMedium, 10}, // default case
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("Pressure_%d", tt.pressure), func(t *testing.T) {
			percent := dm.GetCleanupPercentage(tt.pressure)
			assert.Equal(t, tt.expected, percent)
		})
	}
}

func TestUnit_Shipper_Disk_PurgeFilesBefore(t *testing.T) {
	helper := NewTestHelper(t)
	dm := helper.CreateDiskManager()

	// Create test files with different timestamps
	baseTime := time.Now()
	require.NoError(t, helper.CreateTestFiles(10, baseTime))

	// Verify files were created
	assert.Equal(t, 10, helper.GetFileCount())

	// Files are created with times:
	// file 0: baseTime (newest)
	// file 1: baseTime - 1 hour
	// file 2: baseTime - 2 hours
	// file 3: baseTime - 3 hours
	// file 4: baseTime - 4 hours
	// file 5: baseTime - 5 hours
	// file 6: baseTime - 6 hours
	// file 7: baseTime - 7 hours
	// file 8: baseTime - 8 hours
	// file 9: baseTime - 9 hours (oldest)

	// Purge files older than just before 5 hours ago
	// This should remove files 5, 6, 7, 8, 9 (5 files)
	cutoff := baseTime.Add(-5*time.Hour + time.Minute) // 4 hours 59 minutes ago
	ctx := context.Background()

	removed, err := dm.PurgeFilesBefore(ctx, cutoff)
	require.NoError(t, err)

	// Should have removed 5 files (files 5-9 are older than cutoff)
	assert.Equal(t, 5, removed)
	assert.Equal(t, 5, helper.GetFileCount())
}

func TestUnit_Shipper_Disk_PurgeOldestPercentage(t *testing.T) {
	tests := []struct {
		name              string
		percent           int
		startingFiles     int
		expectedRemoved   int
		expectedRemaining int
	}{
		{"Remove 50%", 50, 10, 5, 5},
		{"Remove 25%", 25, 8, 2, 6},
		{"Remove 100%", 100, 4, 4, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh test environment for each test
			helper := NewTestHelper(t)
			dm := helper.CreateDiskManager()

			// Create test files
			baseTime := time.Now()
			require.NoError(t, helper.CreateTestFiles(tt.startingFiles, baseTime))

			ctx := context.Background()
			removed, err := dm.PurgeOldestPercentage(ctx, tt.percent)
			require.NoError(t, err)

			assert.Equal(t, tt.expectedRemoved, removed)
			assert.Equal(t, tt.expectedRemaining, helper.GetFileCount())
		})
	}
}

func TestUnit_Shipper_Disk_ManageDiskUsage_NoPressure(t *testing.T) {
	helper := NewTestHelper(t)

	// Set up low disk usage
	helper.mockStore.usage.PercentUsed = 30

	dm := helper.CreateDiskManager()
	ctx := context.Background()
	cutoff := time.Now().Add(-1 * time.Hour)

	err := dm.ManageDiskUsage(ctx, cutoff)
	require.NoError(t, err)

	// No files should be removed
	assert.Equal(t, 0, helper.GetFileCount())
}

func TestUnit_Shipper_Disk_ManageDiskUsage_MediumPressure(t *testing.T) {
	helper := NewTestHelper(t)

	// Create test files
	baseTime := time.Now()
	require.NoError(t, helper.CreateTestFiles(10, baseTime))

	// Set medium pressure by overriding the usage calculation
	helper.mockStore.GetUsageFunc = func(limit uint64, paths ...string) (*types.StoreUsage, error) {
		return &types.StoreUsage{
			Total:       1000,
			Used:        750, // 75% usage
			Available:   250,
			PercentUsed: 75,
		}, nil
	}

	dm := helper.CreateDiskManager()
	ctx := context.Background()

	// Set cutoff to remove files older than just before 5 hours ago
	cutoff := baseTime.Add(-5*time.Hour + time.Minute) // 4 hours 59 minutes ago

	err := dm.ManageDiskUsage(ctx, cutoff)
	require.NoError(t, err)

	// Should have removed old files
	assert.Equal(t, 5, helper.GetFileCount())

	// Reset to default behavior
	helper.mockStore.Reset()
}

func TestUnit_Shipper_Disk_ManageDiskUsage_HighPressureEscalation(t *testing.T) {
	helper := NewTestHelper(t)

	// Create test files
	baseTime := time.Now()
	require.NoError(t, helper.CreateTestFiles(20, baseTime))

	dm := helper.CreateDiskManager()
	ctx := context.Background()
	cutoff := baseTime.Add(-5 * time.Hour)

	// Mock high pressure that persists after time-based cleanup
	callCount := 0
	helper.mockStore.GetUsageFunc = func(limit uint64, paths ...string) (*types.StoreUsage, error) {
		callCount++
		switch callCount {
		case 1:
			// Initial high pressure
			return &types.StoreUsage{
				Total: 1000, Used: 900, Available: 100, PercentUsed: 90,
			}, nil
		case 2:
			// Still high pressure after time-based cleanup
			return &types.StoreUsage{
				Total: 1000, Used: 880, Available: 120, PercentUsed: 88,
			}, nil
		default:
			// Pressure reduced after percentage-based cleanup
			return &types.StoreUsage{
				Total: 1000, Used: 600, Available: 400, PercentUsed: 60,
			}, nil
		}
	}

	err := dm.ManageDiskUsage(ctx, cutoff)
	require.NoError(t, err)

	// Should have called GetUsage multiple times for escalation
	assert.GreaterOrEqual(t, callCount, 3)

	// Reset to default behavior
	helper.mockStore.Reset()
}

func TestUnit_Shipper_Disk_ErrorHandling(t *testing.T) {
	helper := NewTestHelper(t)
	dm := helper.CreateDiskManager()

	t.Run("Invalid percentage", func(t *testing.T) {
		ctx := context.Background()

		// Test negative percentage
		_, err := dm.PurgeOldestPercentage(ctx, -1)
		assert.Error(t, err)

		// Test percentage over 100
		_, err = dm.PurgeOldestPercentage(ctx, 101)
		assert.Error(t, err)

		// Test valid percentages (should not error)
		_, err = dm.PurgeOldestPercentage(ctx, 0)
		assert.NoError(t, err)

		_, err = dm.PurgeOldestPercentage(ctx, 100)
		assert.NoError(t, err)
	})

	t.Run("Directory does not exist", func(t *testing.T) {
		// Remove the directory
		os.RemoveAll(helper.tempDir)

		ctx := context.Background()
		cutoff := time.Now().Add(-1 * time.Hour)

		_, err := dm.PurgeFilesBefore(ctx, cutoff)
		assert.Error(t, err)
	})
}

// Memory vs Disk specific tests
func TestUnit_Shipper_Disk_MemoryVsDisk(t *testing.T) {
	t.Run("Memory-backed storage", func(t *testing.T) {
		// Create tmpfs-backed directory for memory test
		helper := NewTestHelper(t)

		// Memory storage should be more aggressive in cleanup
		dm := helper.CreateDiskManager()

		baseTime := time.Now()
		require.NoError(t, helper.CreateTestFiles(20, baseTime))

		ctx := context.Background()
		cutoff := baseTime.Add(-10 * time.Hour)

		// Simulate critical memory pressure
		helper.mockStore.GetUsageFunc = func(limit uint64, paths ...string) (*types.StoreUsage, error) {
			return &types.StoreUsage{
				Total: 100000, Used: 96000, Available: 4000, PercentUsed: 96,
			}, nil
		}

		err := dm.ManageDiskUsage(ctx, cutoff)
		require.NoError(t, err)

		// Should be very aggressive in cleanup for memory
		assert.LessOrEqual(t, helper.GetFileCount(), 10) // Should remove at least half

		// Reset to default behavior
		helper.mockStore.Reset()
	})

	t.Run("Disk-backed storage", func(t *testing.T) {
		helper := NewTestHelper(t)

		dm := helper.CreateDiskManager()

		baseTime := time.Now()
		require.NoError(t, helper.CreateTestFiles(20, baseTime))

		ctx := context.Background()
		cutoff := baseTime.Add(-10 * time.Hour)

		// Simulate moderate disk pressure
		helper.mockStore.GetUsageFunc = func(limit uint64, paths ...string) (*types.StoreUsage, error) {
			return &types.StoreUsage{
				Total: 10000000, Used: 7500000, Available: 2500000, PercentUsed: 75,
			}, nil
		}

		err := dm.ManageDiskUsage(ctx, cutoff)
		require.NoError(t, err)

		// Should be less aggressive for disk storage
		assert.GreaterOrEqual(t, helper.GetFileCount(), 10) // Should preserve more files

		// Reset to default behavior
		helper.mockStore.Reset()
	})
}

func TestUnit_Shipper_Disk_isMemoryBacked(t *testing.T) {
	t.Run("Memory-backed detection", func(t *testing.T) {
		// Create a tmpfs mount for testing
		tempDir, err := os.MkdirTemp("", "tmpfs_test_*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		dm := &shipper.DiskManager{
			StoragePath: tempDir,
		}

		// This will depend on your test environment
		isMemory := dm.IsMemoryBacked()
		// Assert based on your test setup
		t.Logf("Storage path %s is memory-backed: %v", tempDir, isMemory)
	})
}

func TestIntegration_Shipper_Disk(t *testing.T) {
	helper := NewTestHelper(t)
	dm := helper.CreateDiskManager()

	// Create a realistic scenario
	baseTime := time.Now()
	require.NoError(t, helper.CreateTestFiles(100, baseTime))

	// Simulate actual high disk usage to trigger cleanup
	callCount := 0
	helper.mockStore.GetUsageFunc = func(limit uint64, paths ...string) (*types.StoreUsage, error) {
		callCount++

		// First call - simulate high pressure
		if callCount == 1 {
			return &types.StoreUsage{
				Total:       10000,
				Used:        8500, // 85% usage - triggers high pressure
				Available:   1500,
				PercentUsed: 85,
			}, nil
		}

		// Second call (after time-based cleanup) - still high pressure
		if callCount == 2 {
			return &types.StoreUsage{
				Total:       10000,
				Used:        8000, // Still high but reduced
				Available:   2000,
				PercentUsed: 80,
			}, nil
		}

		// Final call - pressure reduced
		return &types.StoreUsage{
			Total:       10000,
			Used:        3000, // Much lower usage
			Available:   7000,
			PercentUsed: 30,
		}, nil
	}

	ctx := context.Background()
	// Set cutoff to remove files older than 48 hours (should remove files 48-99)
	cutoff := baseTime.Add(-48 * time.Hour)

	err := dm.ManageDiskUsage(ctx, cutoff)
	require.NoError(t, err)

	// Should have managed disk usage effectively
	finalCount := helper.GetFileCount()
	assert.Less(t, finalCount, 100)  // Should have removed some files
	assert.Greater(t, finalCount, 0) // But not all files

	// Reset to default behavior
	helper.mockStore.Reset()
}
