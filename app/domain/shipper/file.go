// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
)

type FileOpt func(f *File)

func WithPath(path string) FileOpt {
	return func(f *File) {
		f.ReferenceID = path
	}
}

func WithPresignedUrl(url string) FileOpt {
	return func(f *File) {
		f.PresignedURL = url
	}
}

func WithFileReader(fileReader FileReader) FileOpt {
	return func(f *File) {
		f.fileReader = fileReader
	}
}

type File struct {
	ReferenceID  string `json:"reference_id"` //nolint:tagliatelle // endstream api accepts cammel case
	PresignedURL string `json:"-"`            //nolint:tagliatelle // ignore this property when marshalling to json
	SHA256       string `json:"sha256"`
	SizeBytes    int64  `json:"size_bytes"`

	fileReader FileReader
	file       *os.File
	data       []byte
}

func NewFile(opts ...FileOpt) (*File, error) {
	f := &File{}

	// apply the options
	for _, item := range opts {
		item(f)
	}

	// set a default file reader
	if f.fileReader == nil {
		f.fileReader = &OSFileReader{}
	}

	// internal operations
	data, err := f.ReadFile()
	if err != nil {
		return nil, err
	}
	f.SHA256 = fmt.Sprintf("%x", sha256.Sum256(data))
	f.SizeBytes = int64(len(data))

	return f, nil
}

func NewFilesFromPaths(paths []string, opts ...FileOpt) ([]*File, error) {
	files := make([]*File, 0)
	for _, path := range paths {
		allOpts := make([]FileOpt, 0)
		allOpts = append(allOpts, WithPath(path))
		allOpts = append(allOpts, opts...)
		f, err := NewFile(allOpts...)
		if err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, nil
}

func (f *File) GetFile() (*os.File, error) {
	if f.file == nil {
		// open the file
		osFile, err := f.fileReader.Open(f.ReferenceID)
		if err != nil {
			return nil, fmt.Errorf("failed to open the file: %w", err)
		}
		f.file = osFile
	}

	return f.file, nil
}

func (f *File) ReadFile() ([]byte, error) {
	if f.data == nil {
		osFile, err := f.GetFile()
		if err != nil {
			return nil, err
		}

		// read file content into buffer
		data, err := f.fileReader.Read(osFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read file content: %w", err)
		}

		f.data = data
	}

	return f.data, nil
}

func (f *File) MarkUploaded() error {
	if err := os.Rename(f.ReferenceID, fmt.Sprintf("%s.uploaded", f.ReferenceID)); err != nil {
		return fmt.Errorf("failed to rename the file: %s", err)
	}

	return nil
}

// Upload uploads the specified file to S3 using the provided presigned URL.
func (m *MetricShipper) Upload(file *File) error {
	data, err := file.ReadFile()
	if err != nil {
		return fmt.Errorf("failed to get the file data: %w", err)
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
