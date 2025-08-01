// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudzero/cloudzero-agent/app/logging/instr"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/rs/zerolog"
)

// UploadFileRequest wraps a file and it's allocated presigned URL
type UploadFileRequest struct {
	File         types.File
	PresignedURL string
}

// UploadFile uploads the specified file to S3 using the provided presigned URL.
func (m *MetricShipper) UploadFile(ctx context.Context, req *UploadFileRequest) error {
	return m.metrics.SpanCtx(ctx, "shipper_UploadFile", func(ctx context.Context, id string) error {
		logger := instr.SpanLogger(ctx, id, func(ctx zerolog.Context) zerolog.Context {
			return ctx.Str("fileId", GetRemoteFileID(req.File))
		})
		logger.Debug().Msg("Uploading file")

		// Create a unique context with a timeout for the upload
		ctx, cancel := context.WithTimeout(ctx, m.setting.Cloudzero.SendTimeout)
		defer cancel()

		{
			data, err := io.ReadAll(req.File)
			if err != nil {
				return errors.Join(ErrFileRead, fmt.Errorf("failed to read the file: %w", err))
			}

			// create the request
			req, ierr := retryablehttp.NewRequestWithContext(ctx, "PUT", req.PresignedURL, bytes.NewBuffer(data))
			if ierr != nil {
				return errors.Join(ErrHTTPUnknown, fmt.Errorf("failed to create upload HTTP request: %w", ierr))
			}

			resp, err := m.HTTPClient.Do(req)
			if err != nil {
				return err
			}

			// check for invalid urls
			if resp != nil && resp.StatusCode == http.StatusForbidden {
				// check the message
				if raw, err := io.ReadAll(resp.Body); err != nil {
					// search in the xml response
					if strings.Contains(string(raw), "Request has expired") {
						resp.Body.Close()
						return ErrExpiredURL
					}
				}
			}

			// inspect
			if err := InspectHTTPResponse(ctx, resp); err != nil {
				return err
			}

			// force nil of the memory
			data = nil
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
			return errors.Join(ErrCreateDirectory, fmt.Errorf("failed to create the upload directory: %w", err))
		}

		// if the filepath already contains the uploaded location,
		// then ignore this entry
		location := file.Location()
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
