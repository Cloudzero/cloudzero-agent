package shipper_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudzero/cloudzero-agent-validator/app/domain/shipper"
	"github.com/cloudzero/cloudzero-agent-validator/app/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestShipper_Unit_Disk_StorageWarnings(t *testing.T) {
	tests := []struct {
		name          string
		percentUsed   float64
		expectedError string
	}{
		{"No Warning", 49.9, ""},
		{"Low Warning", 50.0, ""},
		{"Medium Warning", 65.0, ""},
		{"High Warning", 80.0, ""},
		{"Critical Warning", 90.0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := getTmpDir(t)
			mockLister := &MockAppendableFiles{baseDir: tmpDir}
			mockLister.On("GetUsage").Return(&types.StoreUsage{PercentUsed: tt.percentUsed}, nil)
			mockLister.On("GetFiles", []string(nil)).Return([]string{}, nil)
			mockLister.On("GetFiles", mock.Anything).Return([]string{}, nil)
			mockLister.On("ListFiles", []string(nil)).Return([]os.DirEntry{}, nil)
			mockLister.On("ListFiles", mock.Anything).Return([]os.DirEntry{}, nil)
			mockLister.On("Walk", mock.Anything, mock.Anything).Return(nil)

			settings := getMockSettings("", tmpDir)
			metricShipper, err := shipper.NewMetricShipper(context.Background(), settings, mockLister)
			require.NoError(t, err)

			err = metricShipper.HandleDisk(context.Background(), time.Now())
			if tt.expectedError != "" {
				require.ErrorContains(t, err, tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestShipper_Unit_Disk_DeletesOldFiles(t *testing.T) {
	tmpDir := getTmpDir(t)

	// create old file
	oldFile := filepath.Join(tmpDir, "uploaded", "old.txt")
	require.NoError(t, os.WriteFile(oldFile, []byte("data"), 0o644), "failed to create old file")
	oldTime := time.Now().AddDate(0, 0, -2)
	require.NoError(t, os.Chtimes(oldFile, oldTime, oldTime), "failed to change file time")

	// create new file
	newFile := filepath.Join(tmpDir, "uploaded", "new.txt")
	require.NoError(t, os.WriteFile(newFile, []byte("data"), 0o644), "failed to create new file")

	// setup the mock lister
	mockLister := &MockAppendableFiles{baseDir: tmpDir}
	mockLister.On("Walk", mock.Anything, mock.Anything).Return(nil)

	settings := getMockSettings("", tmpDir)
	metricShipper, err := shipper.NewMetricShipper(context.Background(), settings, mockLister)
	require.NoError(t, err, "failed to create metric shipper")

	require.NoError(t, metricShipper.PurgeMetricsBefore(context.Background(), time.Now().AddDate(0, 0, -1)))

	// verify old file deleted
	_, err = os.Stat(oldFile)
	require.True(t, os.IsNotExist(err), "old file should be deleted")

	// verify new file remains
	_, err = os.Stat(newFile)
	require.NoError(t, err, "new file should remain")
}

func TestShipper_Unit_Disk_SetsMetrics(t *testing.T) {
	// create a metric srv
	pm, err := shipper.InitMetrics()
	require.NoError(t, err)
	srv := httptest.NewServer(pm.Handler())
	defer srv.Close()

	tmpDir := t.TempDir()

	// create mock listers
	mockLister := &MockAppendableFiles{}
	mockLister.On("GetUsage").Return(&types.StoreUsage{
		Total: 1000, Used: 500, PercentUsed: 50.0,
	}, nil)
	mockLister.On("GetFiles", []string(nil)).Return([]string{filepath.Join(tmpDir, "f1"), filepath.Join(tmpDir, "f2")}, nil)
	mockLister.On("GetFiles", []string{shipper.UploadedSubDirectory}).Return([]string{filepath.Join(tmpDir, "f1")}, nil)
	mockLister.On("GetFiles", []string{shipper.ReplaySubDirectory}).Return([]string{filepath.Join(tmpDir, "f1"), filepath.Join(tmpDir, "f2")}, nil)

	// setup the shipper
	settings := getMockSettings("", tmpDir)
	metricShipper, err := shipper.NewMetricShipper(context.Background(), settings, mockLister)
	require.NoError(t, err, "failed to create metric shipper")

	// get disk usage
	_, err = metricShipper.GetDiskUsage(context.Background())
	require.NoError(t, err, "failed to get disk usage")

	// fetch metrics from the mock handler
	resp, err := http.Get(srv.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// ensure the metrics are set
	require.Contains(t, string(body), "shipper_disk_total_size_bytes")
	require.Contains(t, string(body), "shipper_disk_current_usage_bytes")
	require.Contains(t, string(body), "shipper_disk_current_usage_percentage")
	require.Contains(t, string(body), "shipper_disk_current_unsent_file")
	require.Contains(t, string(body), "shipper_disk_current_sent_file")
	require.Contains(t, string(body), "shipper_disk_replay_request_current")
}

func TestShipper_Unit_Disk_ErrorHandling(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name          string
		mockSetup     func(*MockAppendableFiles)
		expectedError string
	}{
		{
			"GetUsageError",
			func(m *MockAppendableFiles) {
				m.On("GetUsage").Return((*types.StoreUsage)(nil), errors.New("disk error"))
			},
			"failed to get the usage",
		},
		{
			"GetFilesError",
			func(m *MockAppendableFiles) {
				m.On("GetUsage").Return(&types.StoreUsage{}, nil)
				m.On("GetFiles", []string(nil)).Return([]string{}, errors.New("file error"))
			},
			"failed to get the unsent files",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLister := &MockAppendableFiles{}
			tt.mockSetup(mockLister)

			// setup the shipper
			settings := getMockSettings("", tmpDir)
			metricShipper, err := shipper.NewMetricShipper(context.Background(), settings, mockLister)
			require.NoError(t, err, "failed to create metric shipper")

			_, err = metricShipper.GetDiskUsage(context.Background())
			require.ErrorContains(t, err, tt.expectedError)
		})
	}
}
