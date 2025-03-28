// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type Logging struct {
	Level string `yaml:"level" default:"info" env:"LOG_LEVEL" env-description:"logging level such as debug, info, error"`
}

func setLoggingOptions(l *Logging) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	// Set the logging level
	switch l.Level {
	case "debug":
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		log.Warn().Str("level", l.Level).Msg("Unknown log level, defaulting to info")
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}
