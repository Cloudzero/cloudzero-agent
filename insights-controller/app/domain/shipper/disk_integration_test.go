package shipper_test

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudzero/cloudzero-insights-controller/app/domain/shipper"
	"github.com/cloudzero/cloudzero-insights-controller/app/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// DiskIntegrationFSTester implements types.AppendableFilesMonitor with fake disk usage reporting
type DiskIntegrationFSTester struct {
	rootDir       string
	uploadedDir   string
	replayDir     string
	simulatedSize uint64
	simulatedUsed uint64
	fileSize      int64
}

func NewFilesystemTester(t *testing.T, dir string) *DiskIntegrationFSTester {
	// Create subdirectories if needed
	uploadedDir := filepath.Join(dir, shipper.UploadedSubDirectory)
	replayDir := filepath.Join(dir, shipper.ReplaySubDirectory)

	if err := os.MkdirAll(uploadedDir, 0o755); err != nil {
		t.Fatalf("Failed to create uploaded dir: %v", err)
	}
	if err := os.MkdirAll(replayDir, 0o755); err != nil {
		t.Fatalf("Failed to create replay dir: %v", err)
	}

	return &DiskIntegrationFSTester{
		rootDir:       dir,
		uploadedDir:   uploadedDir,
		replayDir:     replayDir,
		simulatedSize: 10 * 1024 * 1024 * 1024, // 10GB total
		simulatedUsed: 0,
		fileSize:      256 * 1024 * 1024, // 256MB per file (matches code assumptions)
	}
}

func (ft *DiskIntegrationFSTester) Cleanup() {
	os.RemoveAll(ft.rootDir)
}

// CreateTestFiles creates empty files with specified timestamps
func (ft *DiskIntegrationFSTester) CreateTestFiles(dir string, count int, timeOffset time.Duration) error {
	for i := 0; i < count; i++ {
		filePath := filepath.Join(dir, fmt.Sprintf("file_%d.dat", i))
		file, err := os.Create(filePath)
		if err != nil {
			return err
		}
		file.Close()

		// Set modtime to be in the past
		modTime := time.Now().Add(-timeOffset).Add(time.Duration(i) * time.Minute)
		if err := os.Chtimes(filePath, modTime, modTime); err != nil {
			return err
		}
	}
	return nil
}

// CreateTestFilesWithSpecificTimes creates files with specific age distributions
func (ft *DiskIntegrationFSTester) CreateTestFilesWithAgeDistribution(dir string, recentCount, oldCount int, oldAgeThreshold time.Duration) error {
	// Create recent files
	for i := 0; i < recentCount; i++ {
		filePath := filepath.Join(dir, fmt.Sprintf("recent_file_%d.dat", i))
		file, err := os.Create(filePath)
		if err != nil {
			return err
		}
		file.Close()

		// Set modtime to be recent (less than the threshold)
		modTime := time.Now().Add(-oldAgeThreshold / 2).Add(time.Duration(i) * time.Minute)
		if err := os.Chtimes(filePath, modTime, modTime); err != nil {
			return err
		}
	}

	// Create old files
	for i := 0; i < oldCount; i++ {
		filePath := filepath.Join(dir, fmt.Sprintf("old_file_%d.dat", i))
		file, err := os.Create(filePath)
		if err != nil {
			return err
		}
		file.Close()

		// Set modtime to be older than the threshold
		modTime := time.Now().Add(-oldAgeThreshold).Add(-time.Duration(i+1) * time.Hour)
		if err := os.Chtimes(filePath, modTime, modTime); err != nil {
			return err
		}
	}
	return nil
}

// SimulateDiskUsage sets the simulated disk usage percentage
func (ft *DiskIntegrationFSTester) SimulateDiskUsage(percentUsed float64) {
	ft.simulatedUsed = uint64(float64(ft.simulatedSize) * percentUsed / 100)
}

// FileLister implementation that uses real filesystem but simulated usage
type TestFileLister struct {
	tester *DiskIntegrationFSTester
}

func NewTestFileLister(tester *DiskIntegrationFSTester) *TestFileLister {
	return &TestFileLister{tester: tester}
}

func (tl *TestFileLister) GetUsage(paths ...string) (*types.StoreUsage, error) {
	used := tl.tester.simulatedUsed
	total := tl.tester.simulatedSize
	available := total - used
	percentUsed := float64(used) / float64(total) * 100

	return &types.StoreUsage{
		Total:          total,
		Available:      available,
		Used:           used,
		PercentUsed:    percentUsed,
		BlockSize:      4096,
		Reserved:       0,
		InodeTotal:     1000000,
		InodeUsed:      1000,
		InodeAvailable: 999000,
	}, nil
}

func (tl *TestFileLister) GetFiles(subDirs ...string) ([]string, error) {
	subDir := tl.tester.rootDir
	if len(subDirs) > 0 {
		subDir = filepath.Join(tl.tester.rootDir, subDirs[0])
	}

	return filepath.Glob(filepath.Join(subDir, "*"))
}

func (tl *TestFileLister) ListFiles(subDirs ...string) ([]fs.DirEntry, error) {
	subDir := tl.tester.rootDir
	if len(subDirs) > 0 {
		subDir = filepath.Join(tl.tester.rootDir, subDirs[0])
	}

	return os.ReadDir(subDir)
}

func (tl *TestFileLister) Walk(subDir string, walkFn filepath.WalkFunc) error {
	fullPath := filepath.Join(tl.tester.rootDir, subDir)
	return filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		fullRelPath := path
		if subDir != "" {
			fullRelPath = path
		}

		return walkFn(fullRelPath, info, err)
	})
}

func TestShipper_Integration_Disk_PurgeMetricsBefore(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := getTmpDir(t)

	fsTester := NewFilesystemTester(t, tmpDir)
	defer fsTester.Cleanup()

	cutoffDate := time.Now().AddDate(0, 0, -90)

	// Create 30 recent files and 20 old files
	err := fsTester.CreateTestFilesWithAgeDistribution(
		fsTester.uploadedDir,
		30,              // recent files
		20,              // old files
		90*24*time.Hour, // 90 days threshold
	)
	assert.NoError(t, err)

	// Create the metric shipper with the mock lister
	fileLister := NewTestFileLister(fsTester)
	settings := getMockSettingsIntegration(t, fsTester.rootDir, "no-api-key")
	metricShipper, err := shipper.NewMetricShipper(context.Background(), settings, fileLister)
	require.NoError(t, err)

	err = metricShipper.PurgeMetricsBefore(context.Background(), cutoffDate)
	assert.NoError(t, err)

	// Verify only old files were deleted
	files, err := os.ReadDir(fsTester.uploadedDir)
	assert.NoError(t, err)
	assert.Len(t, files, 30) // Only the 30 recent files should remain

	// Verify that all remaining files are newer than the cutoff
	for _, file := range files {
		info, err := file.Info()
		assert.NoError(t, err)
		assert.True(t, info.ModTime().After(cutoffDate),
			"Found file with modification time before cutoff: %s", file.Name())
	}
}

func TestShipper_Integration_Disk_PurgeOldestPercentage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := getTmpDir(t)

	fsTester := NewFilesystemTester(t, tmpDir)
	defer fsTester.Cleanup()

	err := fsTester.CreateTestFiles(fsTester.uploadedDir, 100, 24*time.Hour)
	assert.NoError(t, err)

	// Create the metric shipper with the mock lister
	fileLister := NewTestFileLister(fsTester)
	settings := getMockSettingsIntegration(t, fsTester.rootDir, "no-api-key")
	metricShipper, err := shipper.NewMetricShipper(context.Background(), settings, fileLister)
	require.NoError(t, err)

	err = metricShipper.PurgeOldestNPercentage(context.Background(), shipper.CriticalPurgePercent)
	assert.NoError(t, err)

	// Verify shipper.CriticalPurgePercent files were deleted (shipper.CriticalPurgePercent% of 100)
	files, err := os.ReadDir(fsTester.uploadedDir)
	assert.NoError(t, err)
	assert.Len(t, files, 100-shipper.CriticalPurgePercent)

	// The oldest files should be removed, so file_0 through file_${shipper.CriticalPurgePercent-1} should be gone
	for i := 0; i < shipper.CriticalPurgePercent; i++ {
		oldestFilePath := filepath.Join(fsTester.uploadedDir, fmt.Sprintf("file_%d.dat", i))
		_, err := os.Stat(oldestFilePath)
		assert.True(t, os.IsNotExist(err), "File should have been deleted: %s", oldestFilePath)
	}

	// And file_${shipper.CriticalPurgePercent} through file_99 should still exist
	for i := shipper.CriticalPurgePercent; i < 100; i++ {
		newerFilePath := filepath.Join(fsTester.uploadedDir, fmt.Sprintf("file_%d.dat", i))
		_, err := os.Stat(newerFilePath)
		assert.NoError(t, err, "File should still exist: %s", newerFilePath)
	}
}

func TestShipper_Integration_Disk_FSManagement(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	tmpDir := getTmpDir(t)

	fsTester := NewFilesystemTester(t, tmpDir)
	defer fsTester.Cleanup()

	cutoffDate := time.Now().Add(-90 * 24 * time.Hour)

	t.Run("No Warning Level", func(t *testing.T) {
		// Create fresh files for this test
		err := fsTester.CreateTestFilesWithAgeDistribution(
			fsTester.uploadedDir,
			30,              // recent files
			20,              // old files
			90*24*time.Hour, // 90 days threshold
		)
		assert.NoError(t, err)

		// Create the metric shipper with the mock lister
		fileLister := NewTestFileLister(fsTester)
		settings := getMockSettingsIntegration(t, fsTester.rootDir, "no-api-key")
		metricShipper, err := shipper.NewMetricShipper(context.Background(), settings, fileLister)
		require.NoError(t, err)

		fsTester.SimulateDiskUsage(50) // 50% usage
		err = metricShipper.HandleDisk(context.Background(), cutoffDate)
		assert.NoError(t, err)

		// Verify no files were deleted
		files, err := os.ReadDir(fsTester.uploadedDir)
		assert.NoError(t, err)
		assert.Len(t, files, 50) // All 50 files should remain
	})

	t.Run("High Warning Level", func(t *testing.T) {
		// Clean up from previous test
		os.RemoveAll(fsTester.uploadedDir)
		os.MkdirAll(fsTester.uploadedDir, 0o755)

		// Create fresh files for this test
		err := fsTester.CreateTestFilesWithAgeDistribution(
			fsTester.uploadedDir,
			30,              // recent files
			20,              // old files
			90*24*time.Hour, // 90 days threshold
		)
		assert.NoError(t, err)

		// Create the metric shipper with the mock lister
		fileLister := NewTestFileLister(fsTester)
		settings := getMockSettingsIntegration(t, fsTester.rootDir, "no-api-key")
		metricShipper, err := shipper.NewMetricShipper(context.Background(), settings, fileLister)
		require.NoError(t, err)

		fsTester.SimulateDiskUsage(85) // 85% usage - should trigger high warning
		err = metricShipper.HandleDisk(context.Background(), cutoffDate)
		assert.NoError(t, err)

		// Verify only the old files were deleted
		files, err := os.ReadDir(fsTester.uploadedDir)
		assert.NoError(t, err)
		assert.Len(t, files, 30) // Only the 30 recent files should remain

		// All remaining files should be newer than the cutoff
		for _, file := range files {
			info, err := file.Info()
			assert.NoError(t, err)
			assert.True(t, info.ModTime().After(cutoffDate),
				"Found file with modification time before cutoff: %s", file.Name())
		}
	})

	// Test case: Critical warning level
	t.Run("Critical Warning Level", func(t *testing.T) {
		// Clean up from previous test
		os.RemoveAll(fsTester.uploadedDir)
		os.MkdirAll(fsTester.uploadedDir, 0o755)

		// Create fresh files for this test
		err := fsTester.CreateTestFiles(fsTester.uploadedDir, 50, 24*time.Hour)
		assert.NoError(t, err)

		// Create the metric shipper with the mock lister
		fileLister := NewTestFileLister(fsTester)
		settings := getMockSettingsIntegration(t, fsTester.rootDir, "no-api-key")
		metricShipper, err := shipper.NewMetricShipper(context.Background(), settings, fileLister)
		require.NoError(t, err)

		fsTester.SimulateDiskUsage(95) // 95% usage - should trigger critical warning
		err = metricShipper.HandleDisk(context.Background(), cutoffDate)
		assert.NoError(t, err)

		// Verify 20% of files were deleted
		files, err := os.ReadDir(fsTester.uploadedDir)
		assert.NoError(t, err)
		assert.Len(t, files, 40) // 50 - (50 * 20%) = 40
	})
}
