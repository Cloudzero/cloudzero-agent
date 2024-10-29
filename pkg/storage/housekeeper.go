package storage

import (
	"time"

	"github.com/rs/zerolog/log"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
)

type HouseKeeper struct {
	writer   DatabaseWriter
	settings *config.Settings
}

func NewHouseKeeper(writer DatabaseWriter, settings *config.Settings) *HouseKeeper {
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
