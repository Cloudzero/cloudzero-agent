package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-obvious/server"
	"github.com/rs/zerolog/log"

	"github.com/cloudzero/cirrus-remote-write/app/config"
	"github.com/cloudzero/cirrus-remote-write/app/domain"
	"github.com/cloudzero/cirrus-remote-write/app/handlers"
	"github.com/cloudzero/cirrus-remote-write/app/internal/build"
	"github.com/cloudzero/cirrus-remote-write/app/store"
	"github.com/cloudzero/cirrus-remote-write/app/types"
)

func main() {
	var configFile string
	flag.StringVar(&configFile, "config", configFile, "Path to the configuration file")
	flag.Parse()

	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		log.Fatal().Err(err).Msg("configuration file does not exist")
	}

	settings, err := config.NewSettings(configFile)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load settings")
	}

	appendable, err := store.NewParquetStore(settings.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize database")
	}

	go HandleShutdownEvents(appendable)

	domain := domain.NewMetricCollector(settings, appendable)
	defer domain.Close()

	log.Info().Msg("Starting service")
	server.New(build.Version(), handlers.NewRemoteWriteAPI("/metrics", domain)).Run(context.Background())
	log.Info().Msg("Service stopping")
}

func HandleShutdownEvents(appendable types.Appendable) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	log.Info().Msg("Service stopping")
	appendable.Flush()
	os.Exit(0)
}
