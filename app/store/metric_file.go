// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudzero/cloudzero-agent-validator/app/types"
)

type MetricFile struct {
	*os.File // wrapper around an os.File

	location string
	reader   io.ReadCloser
}

// ensure MetricFile implements File
var _ types.File = (*MetricFile)(nil)

func NewMetricFile(path string) (*MetricFile, error) {
	var file *os.File
	if _, err := os.Stat(path); err == nil {
		// read the file
		f, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("failed to open the file: %w", err)
		}
		file = f
	} else {
		// create a new file
		f, err := os.Create(path)
		if err != nil {
			return nil, fmt.Errorf("failed to create the file: %w", err)
		}
		file = f
	}

	metricFile := &MetricFile{
		File:     file,
		location: path,
	}

	return metricFile, nil
}

func (f *MetricFile) UniqueID() string {
	base := filepath.Base(f.location)

	// remove all extensions
	if idx := strings.Index(base, "."); idx != -1 {
		return base[:idx]
	}

	// base case
	return base
}

func (f *MetricFile) Location() (string, error) {
	return f.location, nil
}

func (f *MetricFile) Rename(new string) error {
	return os.Rename(f.location, new)
}

// TODO -- this is not correct because of how data is streamed into parquet format
func (f *MetricFile) Size() (int64, error) {
	s, err := os.Stat(f.location)
	if err != nil {
		return 0, fmt.Errorf("failed to find the file: %w", err)
	}
	return s.Size(), nil
}

func (f *MetricFile) Read(p []byte) (int, error) {
	if f.reader == nil {
		_, err := f.File.Seek(0, io.SeekStart)
		if err != nil {
			return 0, fmt.Errorf("failed to seek to beginning of file: %w", err)
		}
		f.reader = NewParquetStreamer(f.File)
	}
	return f.reader.Read(p)
}

func (f *MetricFile) Close() error {
	if f.reader != nil {
		f.reader.Close()
		f.reader = nil
	}
	return f.File.Close()
}
