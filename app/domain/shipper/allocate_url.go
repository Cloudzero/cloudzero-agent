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
	Files []*PresignedURLAPIPayloadFile `json:"files"`
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
	// create the payload with the files
	bodyFiles := make([]*PresignedURLAPIPayloadFile, len(files))
	for i, file := range files {
		bodyFiles[i] = &PresignedURLAPIPayloadFile{
			ReferenceID: GetRemoteFileID(file),
		}
	}

	// create the http request body
	body := PresignedURLAPIPayload{Files: bodyFiles}

	// marshal to json
	enc, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to encode the body into json: %w", err)
	}

	// Create a new HTTP request
	uploadEndpoint, err := m.setting.GetRemoteAPIBase()
	if err != nil {
		return nil, fmt.Errorf("failed to get remote base: %w", err)
	}
	uploadEndpoint.Path += "/upload"
	req, err := http.NewRequestWithContext(m.ctx, "POST", uploadEndpoint.String(), bytes.NewBuffer(enc))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set necessary headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", m.setting.GetAPIKey())

	// Make sure we set the query parameters for count, expiration, cloud_account_id, region, cluster_name
	q := req.URL.Query()
	q.Add("count", strconv.Itoa(len(files)))
	q.Add("expiration", strconv.Itoa(expirationTime))
	q.Add("cloud_account_id", m.setting.CloudAccountID)
	q.Add("cluster_name", m.setting.ClusterName)
	q.Add("region", m.setting.Region)
	req.URL.RawQuery = q.Encode()

	log.Info().Msgf("Requesting %d presigned URLs", len(files))

	// Send the request
	resp, err := m.HTTPClient.Do(req)
	if err != nil {
		log.Error().Err(err).Msg("HTTP request failed")
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, ErrUnauthorized
	}

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse the response
	var response map[string]string // map of: {ReferenceId: PresignedURL}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// validation
	if len(response) == 0 {
		return nil, ErrNoURLs
	}

	// check for a replay request
	rrh := resp.Header.Get("X-CloudZero-Replay")
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

	return response, nil
}
