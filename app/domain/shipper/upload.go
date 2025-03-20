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
	"path/filepath"
	"strings"

	"github.com/cloudzero/cloudzero-insights-controller/app/instr"
	"github.com/cloudzero/cloudzero-insights-controller/app/types"
	"github.com/rs/zerolog"
)

// UploadFile uploads the specified file to S3 using the provided presigned URL.
func (m *MetricShipper) UploadFile(ctx context.Context, file types.File, presignedURL string) error {
	return m.metrics.SpanCtx(ctx, "shipper_UploadFile", func(ctx context.Context, id string) error {
		logger := instr.SpanLogger(ctx, id, func(ctx zerolog.Context) zerolog.Context {
			return ctx.Str("fileId", GetRemoteFileID(file))
		})
		logger.Debug().Msg("Uploading file")

		// Create a unique context with a timeout for the upload
		ctx, cancel := context.WithTimeout(ctx, m.setting.Cloudzero.SendTimeout)
		defer cancel()

		data, err := io.ReadAll(file)
		if err != nil {
			return fmt.Errorf("failed to read the file: %w", err)
		}

		// Create a new HTTP PUT request with the file as the body
		req, err := http.NewRequestWithContext(ctx, "PUT", presignedURL, bytes.NewBuffer(data))
		if err != nil {
			return fmt.Errorf("failed to create upload HTTP request: %w", err)
		}

		// Send the request
		httpSpan := m.metrics.StartSpan(ctx, "shipper_UploadFile_httpRequest")
		httpSpanLogger := httpSpan.Logger()
		httpSpanLogger.Debug().Msg("Sending the http request ...")
		defer httpSpan.End()
		resp, err := m.HTTPClient.Do(req)
		if err != nil {
			httpSpanLogger.Err(err).Msg("HTTP request failed")
			return fmt.Errorf("file upload HTTP request failed: %w", err)
		}
		httpSpanLogger.Debug().Msg("Successfully sent http request")
		httpSpan.End()
		defer resp.Body.Close()

		// Check for successful upload
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
			bodyBytes, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("unexpected upload status code %d: %s", resp.StatusCode, string(bodyBytes))
		}

		return nil
	})
}

func (m *MetricShipper) MarkFileUploaded(ctx context.Context, file types.File) error {
	return m.metrics.SpanCtx(ctx, "shipper_MarkFileUploaded", func(ctx context.Context, id string) error {
		logger := instr.SpanLogger(ctx, id, func(ctx zerolog.Context) zerolog.Context {
			return ctx.Str("fileId", GetRemoteFileID(file))
		})
		logger.Debug().Msg("Marking file as uploaded")

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

		logger.Debug().Msg("Successfully marked file as uploaded")

		return nil
	})
}
