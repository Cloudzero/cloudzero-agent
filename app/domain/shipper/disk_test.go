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

	"github.com/cloudzero/cloudzero-insights-controller/app/domain/shipper"
	"github.com/cloudzero/cloudzero-insights-controller/app/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestShipper_Disk_StorageWarnings(t *testing.T) {
	tests := []struct {
		name          string
		percentUsed   float64
		expectedError string
	}{
		{"NoWarning", 49.9, ""},
		{"LowWarning", 50.0, ""},
		{"MediumWarning", 65.0, ""},
		{"HighWarning", 80.0, ""},
		{"CriticalWarning", 90.0, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			mockLister := &MockAppendableFiles{}
			mockLister.On("GetUsage").Return(&types.StoreUsage{PercentUsed: tt.percentUsed}, nil)
			mockLister.On("GetFiles").Return([]string{}, nil)
			mockLister.On("GetMatching", mock.Anything, mock.Anything).Return([]string{}, nil)
			mockLister.On("GetOlderThan", mock.Anything, mock.Anything).Return([]string{}, nil)

			settings := getMockSettings("")
			settings.Database.StoragePath = tmpDir
			metricShipper, err := shipper.NewMetricShipper(context.Background(), settings, mockLister)
			require.NoError(t, err)

			err = metricShipper.HandleDisk()
			if tt.expectedError != "" {
				require.ErrorContains(t, err, tt.expectedError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestShipper_Disk_DeletesOldFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// create old file
	oldFile := filepath.Join(tmpDir, "old.txt")
	require.NoError(t, os.WriteFile(oldFile, []byte("data"), 0o644), "failed to create old file")
	oldTime := time.Now().AddDate(0, 0, -91)
	require.NoError(t, os.Chtimes(oldFile, oldTime, oldTime), "failed to change file time")

	// create new file
	newFile := filepath.Join(tmpDir, "new.txt")
	require.NoError(t, os.WriteFile(newFile, []byte("data"), 0o644), "failed to create new file")

	// setup the mock lister
	mockLister := &MockAppendableFiles{}
	mockLister.On("GetOlderThan", mock.Anything, mock.Anything).Return([]string{filepath.Join(tmpDir, "old.txt")}, nil)

	settings := getMockSettings("")
	settings.Database.StoragePath = tmpDir
	metricShipper, err := shipper.NewMetricShipper(context.Background(), settings, mockLister)
	require.NoError(t, err, "failed to create metric shipper")

	require.NoError(t, metricShipper.PurgeOldMetrics())

	// verify old file deleted
	_, err = os.Stat(oldFile)
	require.True(t, os.IsNotExist(err), "old file should be deleted")

	// verify new file remains
	_, err = os.Stat(newFile)
	require.NoError(t, err, "new file should remain")
}

func TestShipper_Disk_SetsMetrics(t *testing.T) {
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
	mockLister.On("GetFiles").Return([]string{filepath.Join(tmpDir, "f1"), filepath.Join(tmpDir, "f2")}, nil)
	mockLister.On("GetMatching", "uploaded", mock.Anything).Return([]string{filepath.Join(tmpDir, "f1")}, nil)
	mockLister.On("GetMatching", "replay", mock.Anything).Return([]string{filepath.Join(tmpDir, "f1"), filepath.Join(tmpDir, "f2")}, nil)

	// setup the shipper
	settings := getMockSettings("")
	settings.Database.StoragePath = tmpDir
	metricShipper, err := shipper.NewMetricShipper(context.Background(), settings, mockLister)
	require.NoError(t, err, "failed to create metric shipper")

	// get disk usage
	_, err = metricShipper.GetDiskUsage()
	require.NoError(t, err, "failed to get disk usage")

	// fetch metrics from the mock handler
	resp, err := http.Get(srv.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	// ensure the metrics are set
	require.Contains(t, string(body), "shipper_disk_total_size_bytes")
	require.Contains(t, string(body), "shipper_current_disk_usage_bytes")
	require.Contains(t, string(body), "shipper_current_disk_usage_percentage")
	require.Contains(t, string(body), "shipper_current_disk_unsent_file")
	require.Contains(t, string(body), "shipper_current_disk_sent_file")
	require.Contains(t, string(body), "shipper_current_disk_replay_request")
}

func TestShipper_Disk_ErrorHandling(t *testing.T) {
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
				m.On("GetFiles").Return([]string{}, errors.New("file error"))
			},
			"failed to get the unsent files",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLister := &MockAppendableFiles{}
			tt.mockSetup(mockLister)

			// setup the shipper
			settings := getMockSettings("")
			settings.Database.StoragePath = tmpDir
			metricShipper, err := shipper.NewMetricShipper(context.Background(), settings, mockLister)
			require.NoError(t, err, "failed to create metric shipper")

			_, err = metricShipper.GetDiskUsage()
			require.ErrorContains(t, err, tt.expectedError)
		})
	}
}
