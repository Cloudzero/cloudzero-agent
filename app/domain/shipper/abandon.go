// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/cloudzero/cloudzero-agent/app/build"
	"github.com/cloudzero/cloudzero-agent/app/logging/instr"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/rs/zerolog"
)

type AbandonAPIPayloadFile struct {
	ReferenceID string `json:"reference_id"` //nolint:tagliatelle // downstream expects camel case
	Reason      string `json:"reason"`
}

// AbandonFiles sends an abandon request for a list of files with a given
// reason.
func (m *MetricShipper) AbandonFiles(ctx context.Context, payload []*AbandonAPIPayloadFile) error {
	return m.metrics.SpanCtx(ctx, "shipper_AbandonFiles", func(ctx context.Context, id string) error {
		logger := instr.SpanLogger(ctx, id,
			func(ctx zerolog.Context) zerolog.Context {
				return ctx.Int("numFiles", len(payload))
			},
		)
		logger.Debug().
			Msg("Abandoning files ...")

		// ignore empty requests
		if len(payload) == 0 {
			return nil
		}

		// get the shipper id
		shipperID, err := m.GetShipperID()
		if err != nil {
			return errors.Join(ErrInvalidShipperID, fmt.Errorf("failed to get the shipper id: %w", err))
		}

		// serialize the body
		enc, err := json.Marshal(payload)
		if err != nil {
			return errors.Join(ErrEncodeBody, fmt.Errorf("failed to encode the body: %w", err))
		}

		// Create a new HTTP request
		abandonEndpoint, err := m.setting.GetRemoteAPIBase()
		if err != nil {
			return errors.Join(ErrGetRemoteBase, fmt.Errorf("failed to get the abandon endpoint: %w", err))
		}
		abandonEndpoint.Path += abandonAPIPath

		// create the request
		req, err := retryablehttp.NewRequestWithContext(ctx, "POST", abandonEndpoint.String(), bytes.NewBuffer(enc))
		if err != nil {
			return errors.Join(ErrHTTPUnknown, fmt.Errorf("failed to create the HTTP request: %w", err))
		}

		// Set necessary headers
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", m.setting.GetAPIKey())
		req.Header.Set(ShipperIDRequestHeader, shipperID)
		req.Header.Set(AppVersionRequestHeader, build.GetVersion())

		// Make sure we set the query parameters for count, cloud_account_id, region, cluster_name
		q := req.URL.Query()
		q.Add("count", strconv.Itoa(len(payload)))
		q.Add("cluster_name", m.setting.ClusterName)
		q.Add("cloud_account_id", m.setting.CloudAccountID)
		q.Add("region", m.setting.Region)
		q.Add("shipper_id", shipperID)
		req.URL.RawQuery = q.Encode()

		logger.Debug().Msg("Abandoning files")

		// Send the request
		resp, err := m.HTTPClient.Do(req)
		if err != nil {
			return err
		}

		// inspect
		if err := InspectHTTPResponse(ctx, resp); err != nil {
			return err
		}

		defer resp.Body.Close()

		logger.Debug().Msg("Successfully abandoned files")

		// success
		return nil
	})
}
