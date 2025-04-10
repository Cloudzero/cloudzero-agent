// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package inspector

import (
	"context"

	"github.com/rs/zerolog"
)

type ResponseInspectorFunc func(ctx context.Context, resp *responseData, logger zerolog.Logger) (bool, error)

func (i *Inspector) inspect403(_ context.Context, resp *responseData, logger zerolog.Logger) (bool, error) {
	if match, err := resp.JSONMatch(".message == \"User is not authorized to access this resource\""); err != nil {
		return false, err
	} else if match {
		logger.Error().Msg("Invalid CloudZero API key")
		return true, nil
	}

	// Couldn't find a match, dump it all.
	logger.Warn().
		Interface("headers", resp.resp.Header).
		Str("body", string(resp.body())).
		Msg("Unknown HTTP 403 Forbidden error")

	return true, nil
}
