// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package storage

import (
	"time"

	"github.com/rs/zerolog/log"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/types"
)

type HouseKeeper struct {
	writer   types.DatabaseWriter
	settings *config.Settings
}

func NewHouseKeeper(writer types.DatabaseWriter, settings *config.Settings) *HouseKeeper {
	return &HouseKeeper{writer: writer, settings: settings}
}

func (rw *HouseKeeper) StartHouseKeeper() time.Ticker {
	ticker := time.NewTicker(rw.settings.Database.CleanupInterval)

	for range ticker.C {
		err := rw.writer.PurgeStaleData(rw.settings.Database.RetentionTime)
		if err != nil {
			log.Error().Err(err).Msg("Failed to purge stale data")
		}
	}
	return *ticker
}
