// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package shipper

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/cloudzero/cloudzero-agent-validator/app/inspector"
	"github.com/cloudzero/cloudzero-agent-validator/app/instr"
	"github.com/rs/zerolog"
)

func (m *MetricShipper) SendHTTPRequest(
	ctx context.Context,
	name string,
	req *http.Request,
) (*http.Response, error) {
	var resp *http.Response
	err := m.metrics.SpanCtx(ctx, name, func(ctx context.Context, id string) error {
		var err error
		logger := instr.SpanLogger(ctx, id, func(ctx zerolog.Context) zerolog.Context {
			return ctx.Str("httpRequestName", name)
		})

		// send the http request
		logger.Debug().Msg("Sending HTTP request ...")
		resp, err = m.HTTPClient.Do(req)
		if err != nil {
			return err
		}

		// inspect the request
		czInspector := inspector.New()
		if err := czInspector.Inspect(ctx, resp, logger); err != nil {
			return fmt.Errorf("failed to inspect the HTTP response: %w", err)
		}

		logger.Debug().Msg("Successfully sent HTTP request")
		return nil
	})
	if err != nil {
		return nil, errors.Join(ErrHTTPRequestFailed, fmt.Errorf("HTTP request failed: %w", err))
	}

	return resp, nil
}
