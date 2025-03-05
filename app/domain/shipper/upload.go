package shipper

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudzero/cloudzero-insights-controller/app/types"
	"github.com/rs/zerolog/log"
)

// Upload uploads the specified file to S3 using the provided presigned URL.
func (m *MetricShipper) Upload(file types.File, presignedUrl string) error {
	log.Ctx(m.ctx).Debug().Str("fileId", GetRemoteFileID(file)).Msg("Uploading file")

	// Create a unique context with a timeout for the upload
	ctx, cancel := context.WithTimeout(m.ctx, m.setting.Cloudzero.SendTimeout)
	defer cancel()

	data, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read the file: %w", err)
	}

	// Create a new HTTP PUT request with the file as the body
	req, err := http.NewRequestWithContext(ctx, "PUT", presignedUrl, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create upload HTTP request: %w", err)
	}

	// Send the request
	resp, err := m.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("file upload HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check for successful upload
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected upload status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}

func (m *MetricShipper) MarkFileUploaded(file types.File) error {
	log.Ctx(m.ctx).Debug().Str("fileId", GetRemoteFileID(file)).Msg("Marking file as uploaded")

	// create the uploaded dir if needed
	uploadDir := m.GetUploadedDir()
	if err := os.MkdirAll(uploadDir, filePermissions); err != nil {
		return fmt.Errorf("failed to create the upload directory: %w", err)
	}

	// if the filepath already contains the uploaded location,
	// then ignore this entry
	location, err := file.Location()
	if err != nil {
		return fmt.Errorf("failed to get the file location: %w", err)
	}
	if strings.Contains(location, UploadedSubDirectory) {
		return nil
	}

	// rename the file to the uploaded directory
	new := filepath.Join(uploadDir, filepath.Base(location))
	if err := file.Rename(new); err != nil {
		return fmt.Errorf("failed to move the file to the uploaded directory: %s", err)
	}

	return nil
}
