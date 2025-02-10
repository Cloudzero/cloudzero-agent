// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type FileOpt func(f *File)

// Pass in a pre-made pre-signed url.
func FileWithPresignedURL(url string) FileOpt {
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

	file   *os.File
	data   []byte
	sha256 string
	size   int64

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
		stat, err := osFile.Stat()
		if err != nil {
			return nil, fmt.Errorf("failed to get the internal stat of the file: %w", err)
		}
		f.sha256 = fmt.Sprintf("%x", sha256.Sum256(data))
		f.size = stat.Size()
	}

	return f.data, nil
}

// Reset the internal state of the file
// This will NOT reset the `ReferenceID` or the `PresignedURL`
func (f *File) Clear() {
	f.file = nil
	f.data = nil
	f.sha256 = ""
	f.size = 0
}

// Get name of this file on disk
func (f *File) Filename() string {
	return filepath.Base(f.ReferenceID)
}

// Get the root location of this file on disk
func (f *File) Filepath() string {
	return filepath.Dir(f.ReferenceID)
}

// Get the full location of this file on disk
func (f *File) Location() string {
	return filepath.Join(f.Filepath(), f.Filename())
}
