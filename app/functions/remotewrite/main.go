package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-obvious/env"
	"github.com/go-obvious/server"
	"github.com/rs/zerolog/log"

	"github.com/cloudzero/cirrus-remote-write/app/domain"
	"github.com/cloudzero/cirrus-remote-write/app/handlers"
	"github.com/cloudzero/cirrus-remote-write/app/internal/build"
	"github.com/cloudzero/cirrus-remote-write/app/store"
)

func main() {
	// validate the mode was passed
	_ = env.MustGet("SERVER_MODE")

	// dir := env.MustGet("DATABASE_DIR")
	// cntStr := env.MustGet("MAX_RECORDS_PER_FILE")
	// // parse the max records per file into int64
	// max, err := strconv.ParseInt(cntStr, 10, 64)
	// if err != nil {
	// 	log.Fatal().Err(err).Msg("failed to parse MAX_RECORDS_PER_FILE")
	// }

	// appendable, err := store.NewParquetStore(dir, int(max))
	// if err != nil {
	// 	log.Fatal().Err(err).Msg("failed to parse MAX_RECORDS_PER_FILE")
	// }

	go func() {
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
		sig := <-signalChan
		log.Error().Msgf("Received %s signal; shutting down...", sig)

		// FLUSH DATABASE BEFORE SHUTDOWN
		// appendable.Flush()

		os.Exit(0)
	}()

	// Create the data storage layer
	ctx := context.Background()

	domain := domain.NewMetricsDomain(store.NewMemoryStore(), nil)

	// start the service
	server.New(build.Version(), handlers.NewRemoteWrite("/metrics", domain)).Run(ctx)
}
