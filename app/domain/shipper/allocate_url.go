// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/cloudzero/cloudzero-insights-controller/app/types"
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

// format of: `{reference_id: presigned_url}`
type PresignedURLAPIResponse = map[string]string

// Allocates a set of pre-signed urls for the passed file objects
func (m *MetricShipper) AllocatePresignedURLs(files []types.File) (PresignedURLAPIResponse, error) {
	var response PresignedURLAPIResponse

	err := m.metrics.Span("shipper_AllocatePresignedURLs", func() error {
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
			return fmt.Errorf("failed to get the shipper id: %w", err)
		}

		// create the http request body
		body := PresignedURLAPIPayload{
			ShipperID: shipperID,
			Files:     bodyFiles,
		}

		// marshal to json
		enc, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to encode the body into json: %w", err)
		}

		// Create a new HTTP request
		uploadEndpoint, err := m.setting.GetRemoteAPIBase()
		if err != nil {
			return fmt.Errorf("failed to get remote base: %w", err)
		}
		uploadEndpoint.Path += uploadAPIPath
		req, err := http.NewRequestWithContext(m.ctx, "POST", uploadEndpoint.String(), bytes.NewBuffer(enc))
		if err != nil {
			return fmt.Errorf("failed to create HTTP request: %w", err)
		}

		// Set necessary headers
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", m.setting.GetAPIKey())
		req.Header.Set(shipperIDRequestHeader, shipperID)

		// Make sure we set the query parameters for count, expiration, cloud_account_id, region, cluster_name
		q := req.URL.Query()
		q.Add("count", strconv.Itoa(len(files)))
		q.Add("expiration", strconv.Itoa(expirationTime))
		q.Add("cloud_account_id", m.setting.CloudAccountID)
		q.Add("cluster_name", m.setting.ClusterName)
		q.Add("region", m.setting.Region)
		q.Add("shipper_id", shipperID)
		req.URL.RawQuery = q.Encode()

		log.Ctx(m.ctx).Info().Int("numFiles", len(files)).Msg("Requesting presigned URLs")

		// Send the request
		httpSpan := m.metrics.StartSpan("shipper_AllocatePresignedURLs_httpRequest")
		defer httpSpan.End()
		resp, err := m.HTTPClient.Do(req)
		if err != nil {
			log.Error().Err(err).Msg("HTTP request failed")
			return httpSpan.Error(fmt.Errorf("HTTP request failed: %w", err))
		}
		httpSpan.End()
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return ErrUnauthorized
		}

		// Check for HTTP errors
		if resp.StatusCode != http.StatusOK {
			bodyBytes, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(bodyBytes))
		}

		// Parse the response
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}

		// validation
		if len(response) == 0 {
			return ErrNoURLs
		}

		// check for a replay request
		rrh := resp.Header.Get(replayRequestHeader)
		if rrh != "" {
			// parse the replay request
			rr, err := NewReplayRequestFromHeader(rrh)
			if err == nil {
				// save the replay request to disk
				if err = m.SaveReplayRequest(rr); err != nil {
					// do not fail here
					log.Ctx(m.ctx).Err(err).Msg("failed to save the replay request to disk")
				}

				// observe the presence of the replay request
				metricReplayRequestTotal.WithLabelValues().Inc()
				metricReplayRequestCurrent.WithLabelValues().Inc()
			} else {
				// do not fail the operation here
				log.Ctx(m.ctx).Err(err).Msg("failed to parse the replay request header")
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return response, nil
}
