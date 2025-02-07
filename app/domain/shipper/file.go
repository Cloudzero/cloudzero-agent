// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
)

type FileOpt func(f *File)

// Pass in a pre-made pre-signed url.
func FileWithPresignedUrl(url string) FileOpt {
	return func(f *File) {
		f.PresignedURL = url
	}
}

// If set to true, the file will be opened on initialization.
// This opening will set the sha and size fields of the file,
// in addition to ensuring at initialization time that this file exists.
func FileWithoutLazyLoad(lazy bool) FileOpt {
	return func(f *File) {
		f.notLazy = lazy
	}
}

type File struct {
	ReferenceID  string `json:"reference_id"` //nolint:tagliatelle // endstream api accepts cammel case
	PresignedURL string `json:"-"`            //nolint:tagliatelle // ignore this property when marshalling to json
	SHA256       string `json:"sha256"`
	SizeBytes    int64  `json:"size_bytes"`

	file *os.File
	data []byte

	notLazy bool
}

// Creates a new `File` with an optional list of `FileOpt`.
func NewFile(path string, opts ...FileOpt) (*File, error) {
	f := &File{ReferenceID: path}
	if f.ReferenceID == "" {
		return nil, errors.New("an empty path is not valid")
	}

	// apply the options
	for _, item := range opts {
		item(f)
	}

	// set the internal state of the file
	if f.notLazy {
		if _, err := f.ReadFile(); err != nil {
			return nil, err
		}
	}

	return f, nil
}

// Convenience function to create multiple `File` objects from a list of file paths
// The `FileOpt`s passed in `opts` will be applied to ALL files created.
func NewFilesFromPaths(paths []string, opts ...FileOpt) ([]*File, error) {
	files := make([]*File, 0)
	for _, path := range paths {
		f, err := NewFile(path, opts...)
		if err != nil {
			return nil, fmt.Errorf("issue creating file from path '%s': %w", path, err)
		}
		files = append(files, f)
	}
	return files, nil
}

// Opens the file from the filesystem path: `ReferenceID`
// Gives an `os.File` mount for this file
// This will cache the result if called multiple times.
// To clear the cached read value, call `f.Clear()`.
func (f *File) GetFile() (*os.File, error) {
	if f.file == nil {
		// open the file
		osFile, err := os.Open(f.ReferenceID)
		if err != nil {
			return nil, fmt.Errorf("failed to open the file: %w", err)
		}
		f.file = osFile
	}

	return f.file, nil
}

// Read the file as defined from the filesystem path `ReferenceID`
// This will cache the result if called multiple times.
// To clear the cached read value, call `f.Clear()`.
func (f *File) ReadFile() ([]byte, error) {
	if f.data == nil {
		osFile, err := f.GetFile()
		if err != nil {
			return nil, err
		}

		// read file content into buffer
		data, err := io.ReadAll(osFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read file content: %w", err)
		}

		f.data = data

		// set internal state of the file
		f.SHA256 = fmt.Sprintf("%x", sha256.Sum256(data))
		f.SizeBytes = int64(len(data))
	}

	return f.data, nil
}

// Reset the internal state of the file
// This will NOT reset the `ReferenceID` or the `PresignedURL`
func (f *File) Clear() {
	f.SHA256 = ""
	f.SizeBytes = 0
	f.file = nil
	f.data = nil
}

// Mark a file as successfully uploaded
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
