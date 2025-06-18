// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package inspector provides a way to inspect HTTP responses from the CloudZero
// API to diagnose issues.
package inspector

import (
	"context"
	"net/http"

	"github.com/rs/zerolog"
)

// Inspector inspects HTTP responses.
type Inspector struct {
	inspectors map[int]ResponseInspectorFunc
}

// New returns a new Inspector.
func New() *Inspector {
	i := &Inspector{}

	i.inspectors = map[int]ResponseInspectorFunc{
		http.StatusForbidden: i.inspect403,
	}

	return i
}

// Inspect inspects an HTTP response and logs relevant information.
func (i *Inspector) Inspect(ctx context.Context, resp *http.Response, logger zerolog.Logger) error {
	responseData := &responseData{resp: resp}

	// We don't really need to inspect 2xx and 3xx responses; things seem to be working 🎊
	if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusBadRequest {
		logger.Debug().Int("status", resp.StatusCode).Msg("successful HTTP response")
		return nil
	}

	logger = logger.With().Int("status", resp.StatusCode).Logger()
	logger = addCommonHeaders(logger, resp.Header)

	if inspector, ok := i.inspectors[resp.StatusCode]; ok {
		if handled, err := inspector(ctx, responseData, logger); err != nil {
			return err
		} else if handled {
			return nil
		}
	}

	logger.Warn().Msg("Unknown HTTP error")

	return nil
}
