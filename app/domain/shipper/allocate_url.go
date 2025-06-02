// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/cloudzero/cloudzero-agent/app/build"
	"github.com/cloudzero/cloudzero-agent/app/logging/instr"
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

// PresignedURLPayload maps a reference id to a presigned url
type PresignedURLPayload = map[string]string

type AllocatePresignedURLsResponse struct {
	Allocation PresignedURLPayload
	Replay     PresignedURLPayload
}

type replayRequestHeaderValue struct {
	RefID string `json:"ref_id"` //nolint:tagliatelle // upstream uses cammel case
	URL   string `json:"url"`
}

// AllocatePresignedURLs allocates a set of pre-signed urls for the passed file
// objects.
func (m *MetricShipper) AllocatePresignedURLs(files []types.File) (*AllocatePresignedURLsResponse, error) {
	// create the root request object
	response := &AllocatePresignedURLsResponse{
		Allocation: make(PresignedURLPayload),
		Replay:     make(PresignedURLPayload),
	}

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
			return ErrEncodeBody
		}

		// Create a new HTTP request
		uploadEndpoint, err := m.setting.GetRemoteAPIBase()
		if err != nil {
			logger.Err(err).Msg(ErrGetRemoteBase.Error())
			return ErrGetRemoteBase
		}
		uploadEndpoint.Path += uploadAPIPath

		logger.Debug().Int("numFiles", len(files)).Msg("Requesting presigned URLs")

		// Send the request
		resp, err := m.SendHTTPRequest(ctx, "shipper_AllocatePresignedURLs_httpRequest", func() (*http.Request, error) {
			req, ierr := http.NewRequestWithContext(ctx, "POST", uploadEndpoint.String(), bytes.NewBuffer(enc))
			if ierr != nil {
				return nil, errors.Join(ErrHTTPUnknown, fmt.Errorf("failed to create the HTTP request: %w", ierr))
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

			return req, nil
		})
		if err != nil {
			return err
		}

		defer resp.Body.Close()

		// Parse the response
		if err := json.NewDecoder(resp.Body).Decode(&response.Allocation); err != nil {
			return errors.Join(ErrInvalidBody, fmt.Errorf("failed to decode the response: %w", err))
		}

		// check for a replay request
		rrh := resp.Header.Get(ReplayRequestHeader)
		if rrh != "" {
			log.Ctx(m.ctx).Debug().Msg("Recieved replay request")

			var rr []replayRequestHeaderValue
			if err := json.Unmarshal([]byte(rrh), &rr); err == nil {
				for _, item := range rr {
					response.Replay[item.RefID] = item.URL
				}
			} else {
				logger.Err(err).Msg("Failed to parse the replay request into an object")
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
