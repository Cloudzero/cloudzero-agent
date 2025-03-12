// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/rs/zerolog/log"
)

type AbandonAPIPayloadFile struct {
	ReferenceID string `json:"reference_id"` //nolint:tagliatelle // downstream expects cammel case
	Reason      string `json:"reason"`
}

// sends an abandon request for a list of files with a given reason
func (m *MetricShipper) AbandonFiles(referenceIDs []string, reason string) error {
	return m.metrics.Span("shipper_AbandonFiles", func() error {
		if len(referenceIDs) == 0 {
			return errors.New("cannot send in an empty slice")
		}

		// create the body
		body := make([]*AbandonAPIPayloadFile, len(referenceIDs))
		for i, item := range referenceIDs {
			body[i] = &AbandonAPIPayloadFile{
				ReferenceID: item,
				Reason:      reason,
			}
		}

		// serialize the body
		enc, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to encode the body: %w", err)
		}

		// Create a new HTTP request
		abandonEndpoint, err := m.setting.GetRemoteAPIBase()
		if err != nil {
			return fmt.Errorf("failed to get the abandon endpoint: %w", err)
		}
		abandonEndpoint.Path += abandonAPIPath
		req, err := http.NewRequestWithContext(m.ctx, "POST", abandonEndpoint.String(), bytes.NewBuffer(enc))
		if err != nil {
			return fmt.Errorf("failed to create HTTP request: %w", err)
		}

		// Set necessary headers
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", m.setting.GetAPIKey())

		// Make sure we set the query parameters for count, cloud_account_id, region, cluster_name
		q := req.URL.Query()
		q.Add("count", strconv.Itoa(len(referenceIDs)))
		q.Add("cluster_name", m.setting.ClusterName)
		q.Add("cloud_account_id", m.setting.CloudAccountID)
		q.Add("region", m.setting.Region)
		req.URL.RawQuery = q.Encode()

		log.Info().Msgf("Abandoning %d files from '%s'", len(referenceIDs), req.URL.String())

		// Send the request
		httpSpan := m.metrics.StartSpan("shipper_AbandonFiles_httpRequest")
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

		// log the number of success abandoned files
		metricReplayRequestAbandonFilesTotal.WithLabelValues().Add(float64(len(referenceIDs)))

		// success
		return nil
	})
}
