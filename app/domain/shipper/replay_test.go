package shipper_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/cloudzero/cloudzero-agent/app/config/gator"
	"github.com/cloudzero/cloudzero-agent/app/domain/shipper"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestShipper_Unit_ReplayRequest_Create(t *testing.T) {
	t.Parallel()
	referenceIDs := types.NewSetFromList([]string{"file1", "file2"})

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
	metricShipper, err := shipper.NewMetricShipper(context.Background(), settings, nil)
	require.NoError(t, err)

	// save the request
	rr := &shipper.ReplayRequest{ReferenceIDs: referenceIDs}
	err = metricShipper.SaveReplayRequest(context.Background(), rr)
	require.NoError(t, err)
	require.NotNil(t, rr)

	// check the file actually exists
	t.Run("TestShipper_ReplayCreate_ReadFromDisk", func(t *testing.T) {
		// read from the saved directory
		data, err := os.ReadFile(rr.Filepath)
		require.NoError(t, err)

		// serialize
		rr2 := shipper.ReplayRequest{}
		err = json.Unmarshal(data, &rr2)
		require.NoError(t, err)

		// validate
		require.Equal(t, rr.ReferenceIDs.Size(), rr2.ReferenceIDs.Size())
	})

	// ensure reading the active requests works
	t.Run("TestShipper_ReplayCreate_ReadActive", func(t *testing.T) {
		// get active requests
		requests, err := metricShipper.GetActiveReplayRequests(context.Background())
		require.NoError(t, err)
		enc, _ := json.Marshal(requests)
		fmt.Println(string(enc))
		require.Equal(t, 1, len(requests))
		require.Equal(t, rr.Filepath, requests[0].Filepath)
	})
}

func TestShipper_Unit_ReplayRequest_Run(t *testing.T) {
	// get a tmp dir
	tmpDir := getTmpDir(t)

	// create some test files
	files := createTestFiles(t, tmpDir, 5)

	// create the replay request reference ids
	refIDs := types.NewSet[string]()
	for _, item := range files {
		refIDs.Add(shipper.GetRemoteFileID(item))
	}

	// Setup http response
	mockURL := "https://example.com/upload"

	// create the mock response body
	mockResponseBody := make(map[string]string)
	for _, item := range files {
		mockResponseBody[shipper.GetRemoteFileID(item)] = fmt.Sprintf("https://s3.amazonaws.com/bucket/%s?signature=abc123", shipper.GetRemoteFileID(item))
	}

	mockRoundTripper := &MockRoundTripper{
		status:           http.StatusOK,
		mockResponseBody: mockResponseBody,
		mockError:        nil,
	}

	// create the settings
	settings := getMockSettings(mockURL, tmpDir)

	// setup the database backend for the test
	mockFiles := &MockAppendableFiles{baseDir: tmpDir}
	mockFiles.On("GetFiles").Return(refIDs.List(), nil)
	mockFiles.On("GetFiles", shipper.UploadedSubDirectory).Return([]string{}, nil)
	mockFiles.On("Walk", mock.Anything, mock.Anything).Return(nil)

	// create the metricShipper with the http override
	metricShipper, err := shipper.NewMetricShipper(context.Background(), settings, mockFiles)
	require.NoError(t, err)
	metricShipper.HTTPClient.Transport = mockRoundTripper

	// save the replay request
	err = metricShipper.SaveReplayRequest(context.Background(), &shipper.ReplayRequest{ReferenceIDs: refIDs})
	require.NoError(t, err)

	// ensure the replay request can be found
	requests, err := metricShipper.GetActiveReplayRequests(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, requests)

	// process the active replay requests
	err = metricShipper.ProcessReplayRequests(context.Background())
	require.NoError(t, err)

	// ensure files got uploaded
	base, err := os.ReadDir(metricShipper.GetBaseDir())
	require.NoError(t, err)
	uploaded, err := os.ReadDir(metricShipper.GetUploadedDir())
	require.NoError(t, err)
	require.Equal(t, 3, len(base)) // .shipperid replay/ uploaded/
	require.Equal(t, 5, len(uploaded))

	// validate replay request was deleted
	replays, err := os.ReadDir(metricShipper.GetReplayRequestDir())
	require.NoError(t, err)
	require.Empty(t, replays)
}

func TestShipper_Unit_ReplayRequest_Abandon(t *testing.T) {
	// get a tmp dir
	tmpDir := getTmpDir(t)

	// create some test files
	files := createTestFiles(t, tmpDir, 5)

	// create the replay request reference ids
	refIDs := types.NewSet[string]()
	for _, item := range files {
		refIDs.Add(shipper.GetRemoteFileID(item))

		// remove the file
		loc, err := item.Location()
		require.NoError(t, err)
		err = os.Remove(loc)
		require.NoError(t, err)
	}

	// add some previously uploaded files
	files2 := createTestFiles(t, tmpDir, 2)
	uploadedRefIDs := types.NewSet[string]()
	for _, item := range files2 {
		uploadedRefIDs.Add(shipper.GetRemoteFileID(item))
		// move files to the uploaded dir
		loc, err := item.Location()
		require.NoError(t, err)
		err = os.Rename(loc, filepath.Join(tmpDir, shipper.UploadedSubDirectory, item.UniqueID()))
		require.NoError(t, err)
	}

	// Setup http response
	mockURL := "https://example.com/upload"

	// create the mock response body
	mockResponseBody := make(map[string]string)
	for _, item := range files {
		mockResponseBody[shipper.GetRemoteFileID(item)] = fmt.Sprintf("https://s3.amazonaws.com/bucket/%s?signature=abc123", shipper.GetRemoteFileID(item))
	}

	mockRoundTripper := &MockRoundTripper{
		status:           http.StatusOK,
		mockResponseBody: mockResponseBody,
		mockError:        nil,
	}

	// create the settings
	settings := getMockSettings(mockURL, tmpDir)

	// setup the database backend for the test
	mockFiles := &MockAppendableFiles{baseDir: tmpDir}
	mockFiles.On("GetFiles").Return([]string{}, nil)
	mockFiles.On("GetFiles", shipper.UploadedSubDirectory).Return(uploadedRefIDs.List(), nil)
	mockFiles.On("Walk", mock.Anything, mock.Anything).Return(nil)

	// create the metricShipper with the http override
	metricShipper, err := shipper.NewMetricShipper(context.Background(), settings, mockFiles)
	require.NoError(t, err)
	metricShipper.HTTPClient.Transport = mockRoundTripper

	// save the replay request
	err = metricShipper.SaveReplayRequest(context.Background(), &shipper.ReplayRequest{ReferenceIDs: refIDs})
	require.NoError(t, err)

	// ensure the replay request can be found
	requests, err := metricShipper.GetActiveReplayRequests(context.Background())
	require.NoError(t, err)
	require.NotEmpty(t, requests)

	// process the active replay requests
	err = metricShipper.ProcessReplayRequests(context.Background())
	require.NoError(t, err)

	// ensure files got uploaded
	base, err := os.ReadDir(metricShipper.GetBaseDir())
	require.NoError(t, err)
	uploaded, err := os.ReadDir(metricShipper.GetUploadedDir())
	require.NoError(t, err)
	require.Equal(t, 3, len(base)) // .shipperid replay/ uploaded/
	require.Equal(t, 2, len(uploaded))

	// validate replay request was deleted
	replays, err := os.ReadDir(metricShipper.GetReplayRequestDir())
	require.NoError(t, err)
	require.Empty(t, replays)
}
