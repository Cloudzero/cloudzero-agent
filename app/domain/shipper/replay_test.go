package shipper

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"

	"github.com/cloudzero/cloudzero-insights-controller/app/config"
	"github.com/stretchr/testify/require"
)

func TestShipper_ReplayRequestCreate(t *testing.T) {
	t.Parallel()
	referenceIDs := []string{"file1", "file2"}

	settings := &config.Settings{
		Cloudzero: config.Cloudzero{
			SendTimeout:  10,
			SendInterval: 1,
			Host:         "http://example.com",
		},
		Database: config.Database{
			StoragePath: t.TempDir(),
		},
	}
	shipper, err := NewMetricShipper(context.Background(), settings, nil)
	require.NoError(t, err)

	// save the request
	rr, err := shipper.SaveReplayRequest(referenceIDs)
	require.NoError(t, err)
	require.NotNil(t, rr)

	// check the file actually exists
	t.Run("TestShipper_ReplayCreate_ReadFromDisk", func(t *testing.T) {
		// read from the saved directory
		data, err := os.ReadFile(rr.Filepath)
		require.NoError(t, err)

		// serialize
		rr2 := ReplayRequest{}
		err = json.Unmarshal(data, &rr2)
		require.NoError(t, err)

		// validate
		require.Equal(t, len(rr.ReferenceIDs), len(rr2.ReferenceIDs))
	})

	// ensure reading the active requests works
	t.Run("TestShipper_ReplayCreate_ReadActive", func(t *testing.T) {
		// get active requests
		requests, err := shipper.GetActiveReplayRequests()
		require.NoError(t, err)
		enc, _ := json.Marshal(requests)
		fmt.Println(string(enc))
		require.Equal(t, 1, len(requests))
		require.Equal(t, rr.Filepath, requests[0].Filepath)
	})
}

func TestShiper_ReplayRequestRun(t *testing.T) {
	// get a tmp dir
	tmpDir := t.TempDir()
	// create some test files
	files := createTestFiles(t, tmpDir, 5)

	// create the replay request reference ids
	refIDs := make([]string, len(files))
	for i, item := range files {
		refIDs[i] = item.ReferenceID
	}

	// Setup http response
	mockURL := "https://example.com/upload"

	// create the mock response body
	mockResponseBody := make(map[string]string)
	for _, item := range files {
		mockResponseBody[item.ReferenceID] = fmt.Sprintf("https://s3.amazonaws.com/bucket/%s?signature=abc123", item.ReferenceID)
	}

	mockRoundTripper := &MockRoundTripper{
		status:           http.StatusOK,
		mockResponseBody: mockResponseBody,
		mockError:        nil,
	}

	// create the settings
	settings := setupSettings(mockURL)
	settings.Database.StoragePath = tmpDir // use the tmp dir as the root storage dir

	// setup the database backend for the test
	mockFiles := &MockAppendableFiles{}
	mockFiles.On("GetMatching", "", refIDs).Return(refIDs, nil)
	mockFiles.On("GetMatching", settings.Database.StorageUploadSubpath, refIDs).Return([]string{}, nil)

	// create the shipper with the http override
	shipper, err := NewMetricShipper(context.Background(), settings, mockFiles)
	require.NoError(t, err)
	shipper.HTTPClient.Transport = mockRoundTripper

	// save the replay request
	_, err = shipper.SaveReplayRequest(refIDs)
	require.NoError(t, err)

	// ensure the replay request can be found
	requests, err := shipper.GetActiveReplayRequests()
	require.NoError(t, err)
	require.NotEmpty(t, requests)

	// process the active replay requests
	err = shipper.ProcessReplayRequests()
	require.NoError(t, err)

	// ensure files got uploaded
	base, err := os.ReadDir(shipper.GetBaseDir())
	require.NoError(t, err)
	uploaded, err := os.ReadDir(shipper.GetUploadedDir())
	require.NoError(t, err)
	require.Equal(t, 2, len(base))
	require.Equal(t, 5, len(uploaded))

	// validate replay request was deleted
	replays, err := os.ReadDir(shipper.GetReplayRequestDir())
	require.NoError(t, err)
	require.Empty(t, replays)
}
