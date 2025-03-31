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

	"github.com/cloudzero/cloudzero-agent-validator/app/build"
	"github.com/cloudzero/cloudzero-agent-validator/app/instr"
	"github.com/rs/zerolog"
)

type AbandonAPIPayloadFile struct {
	ReferenceID string `json:"reference_id"` //nolint:tagliatelle // downstream expects cammel case
	Reason      string `json:"reason"`
}

// sends an abandon request for a list of files with a given reason
func (m *MetricShipper) AbandonFiles(ctx context.Context, referenceIDs []string, reason string) error {
	return m.metrics.SpanCtx(ctx, "shipper_AbandonFiles", func(ctx context.Context, id string) error {
		logger := instr.SpanLogger(ctx, id,
			func(ctx zerolog.Context) zerolog.Context {
				return ctx.Int("numFiles", len(referenceIDs))
			},
		)
		logger.Debug().Msg("Abandoning files ...")

		if len(referenceIDs) == 0 {
			return errors.New("cannot send in an empty slice")
		}

		// get the shipper id
		shipperID, err := m.GetShipperID()
		if err != nil {
			return fmt.Errorf("failed to get the shipper id: %w", err)
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
		req.Header.Set(ShipperIDRequestHeader, shipperID)
		req.Header.Set(AppVersionRequestHeader, build.GetVersion())

		// Make sure we set the query parameters for count, cloud_account_id, region, cluster_name
		q := req.URL.Query()
		q.Add("count", strconv.Itoa(len(referenceIDs)))
		q.Add("cluster_name", m.setting.ClusterName)
		q.Add("cloud_account_id", m.setting.CloudAccountID)
		q.Add("region", m.setting.Region)
		q.Add("shipper_id", shipperID)
		req.URL.RawQuery = q.Encode()

		logger.Debug().Str("url", req.URL.String()).Msg("Abandoning files")

		// Send the request
		httpSpan := m.metrics.StartSpan(ctx, "shipper_AbandonFiles_httpRequest")
		httpSpanLogger := httpSpan.Logger()
		httpSpanLogger.Debug().Msg("Sending the http request ...")
		defer httpSpan.End()
		resp, err := m.HTTPClient.Do(req)
		if err != nil {
			httpSpanLogger.Err(err).Msg("HTTP request failed")
			return httpSpan.Error(fmt.Errorf("HTTP request failed: %w", err))
		}
		httpSpanLogger.Debug().Msg("Successfully sent http request")
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

		logger.Debug().Msg("Successfully abandoned files")

		// success
		return nil
	})
}
