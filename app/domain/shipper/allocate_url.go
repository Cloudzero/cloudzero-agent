// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/cloudzero/cloudzero-agent/app/build"
	"github.com/cloudzero/cloudzero-agent/app/instr"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type PresignedURLAPIPayload struct {
	ShipperID string                        `json:"shipperId"`
	Files     []*PresignedURLAPIPayloadFile `json:"files"`
}

type PresignedURLAPIPayloadFile struct {
	ReferenceID string `json:"reference_id"`      //nolint:tagliatelle // downstream expects cammel case
	SHA256      string `json:"sha_256,omitempty"` //nolint:tagliatelle // downstream expects cammel case
	Size        int64  `json:"size,omitempty"`
}

// PresignedURLAPIResponse is the format of the response from the remote API.
// The format of the response is: `{reference_id: presigned_url}`.
type PresignedURLAPIResponse = map[string]string

// AllocatePresignedURLs allocates a set of pre-signed urls for the passed file
// objects.
func (m *MetricShipper) AllocatePresignedURLs(files []types.File) (PresignedURLAPIResponse, error) {
	var response PresignedURLAPIResponse
	err := m.metrics.SpanCtx(m.ctx, "shipper_AllocatePresignedURLs", func(ctx context.Context, id string) error {
		logger := instr.SpanLogger(ctx, id, func(ctx zerolog.Context) zerolog.Context {
			return ctx.Int("numFiles", len(files))
		})
		logger.Debug().Msg("Allocating pre-signed URLs ...")

		// create the payload with the files
		bodyFiles := make([]*PresignedURLAPIPayloadFile, len(files))
		for i, file := range files {
			bodyFiles[i] = &PresignedURLAPIPayloadFile{
				ReferenceID: GetRemoteFileID(file),
			}
		}

		// get the shipper id
		shipperID, err := m.GetShipperID()
		if err != nil {
			return ErrInvalidShipperID
		}

		// create the http request body
		body := PresignedURLAPIPayload{
			ShipperID: shipperID,
			Files:     bodyFiles,
		}

		// marshal to json
		enc, err := json.Marshal(body)
		if err != nil {
			logger.Err(err).Msg(ErrEncodeBody.Error())
			return ErrEncodeBody
		}

		// Create a new HTTP request
		uploadEndpoint, err := m.setting.GetRemoteAPIBase()
		if err != nil {
			logger.Err(err).Msg(ErrGetRemoteBase.Error())
			return ErrGetRemoteBase
		}
		uploadEndpoint.Path += uploadAPIPath
		req, err := http.NewRequestWithContext(m.ctx, "POST", uploadEndpoint.String(), bytes.NewBuffer(enc))
		if err != nil {
			return errors.Join(ErrHTTPUnknown, fmt.Errorf("failed to create the HTTP request: %w", err))
		}

		// Set necessary headers
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", m.setting.GetAPIKey())
		req.Header.Set(ShipperIDRequestHeader, shipperID)
		req.Header.Set(AppVersionRequestHeader, build.GetVersion())

		// Make sure we set the query parameters for count, expiration, cloud_account_id, region, cluster_name
		q := req.URL.Query()
		q.Add("count", strconv.Itoa(len(files)))
		q.Add("expiration", strconv.Itoa(expirationTime))
		q.Add("cloud_account_id", m.setting.CloudAccountID)
		q.Add("cluster_name", m.setting.ClusterName)
		q.Add("region", m.setting.Region)
		q.Add("shipper_id", shipperID)
		req.URL.RawQuery = q.Encode()

		logger.Debug().Int("numFiles", len(files)).Msg("Requesting presigned URLs")

		// Send the request
		var resp *http.Response
		err = m.metrics.SpanCtx(ctx, "shipper_AllocatePresignedURLs_httpRequest", func(ctx context.Context, id string) error {
			spanLogger := instr.SpanLogger(ctx, id)
			spanLogger.Debug().Msg("Sending the http request ...")
			resp, err = m.HTTPClient.Do(req)
			if err != nil {
				return err
			}
			spanLogger.Debug().Msg("Successfully sent http request")
			return nil
		})
		if err != nil {
			return errors.Join(ErrHTTPRequestFailed, fmt.Errorf("HTTP request failed: %w", err))
		}

		defer resp.Body.Close()

		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return ErrUnauthorized
		}

		// Check for HTTP errors
		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			return errors.Join(ErrHTTPUnknown, fmt.Errorf("unexpected status code: statusCode=%d, body=%s", resp.StatusCode, string(bodyBytes)))
		}

		// Parse the response
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return errors.Join(ErrInvalidBody, fmt.Errorf("failed to decode the response: %w", err))
		}

		// validation
		if len(response) == 0 {
			logger.Warn().Msg(ErrNoURLs.Error())
			return ErrNoURLs
		}

		// check for a replay request
		rrh := resp.Header.Get(ReplayRequestHeader)
		if rrh != "" {
			log.Ctx(m.ctx).Debug().Msg("Saving replay request to disk ...")

			// parse the replay request
			rr, err := NewReplayRequestFromHeader(rrh)
			if err == nil {
				// save the replay request to disk
				if err = m.SaveReplayRequest(ctx, rr); err != nil {
					// do not fail here
					metricReplayRequestSaveErrorTotal.WithLabelValues(GetErrStatusCode(err)).Inc()
					logger.Err(err).Msg("failed to save the replay request to disk")
				}

				// observe the presence of the replay request
				metricReplayRequestTotal.WithLabelValues().Inc()
				metricReplayRequestCurrent.WithLabelValues().Inc()
				log.Ctx(m.ctx).Debug().Msg("Successfully saved replay request")
			} else {
				// do not fail the operation here
				logger.Err(err).Msg("failed to parse the replay request header")
			}
		}

		logger.Debug().Msg("Successfully allocated presigned urls")

		return nil
	})
	if err != nil {
		return nil, err
	}

	return response, nil
}
