// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
)

type File struct {
	ReferenceID  string `json:"reference_id"` //nolint:tagliatelle // endstream api accepts cammel case
	PresignedURL string `json:"-"`            //nolint:tagliatelle // ignore this property when marshalling to json
}

func NewFileFromPath(path string) *File {
	return &File{ReferenceID: path}
}

func NewFilesFromPaths(paths []string) []*File {
	files := make([]*File, 0)
	for _, path := range paths {
		files = append(files, NewFileFromPath(path))
	}
	return files
}

// UploadFile uploads the specified file to S3 using the provided presigned URL.
func (m *MetricShipper) UploadFile(file *File) error {
	// Open the file to upload
	osFile, err := os.Open(file.ReferenceID)
	if err != nil {
		return fmt.Errorf("failed to open file for upload: %w", err)
	}
	defer osFile.Close()

	// read file content into buffer
	data, err := io.ReadAll(osFile)
	if err != nil {
		return fmt.Errorf("failed to read file content: %w", err)
	}

	// Create a unique context with a timeout for the upload
	ctx, cancel := context.WithTimeout(m.ctx, m.setting.Cloudzero.SendTimeout)
	defer cancel()

	// Create a new HTTP PUT request with the file as the body
	req, err := http.NewRequestWithContext(ctx, "PUT", file.PresignedURL, bytes.NewReader(data))
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
