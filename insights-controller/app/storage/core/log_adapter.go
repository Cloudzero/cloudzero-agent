// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package core provides core functionalities for database repository implementations.
// This package includes base implementations for repositories that can be extended
// to fit specific use cases. It supports transaction management and context-based
// database operations.
//
// ZeroLogAdapter is a custom logger adapter for GORM that uses zerolog for logging.
package core

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"gorm.io/gorm/logger"
)

type ZeroLogAdapter struct{}

func (l ZeroLogAdapter) LogMode(logger.LogLevel) logger.Interface {
	return l
}

func (l ZeroLogAdapter) Error(ctx context.Context, msg string, opts ...interface{}) {
	zerolog.Ctx(ctx).Error().Msg(fmt.Sprintf(msg, opts...))
}

func (l ZeroLogAdapter) Warn(ctx context.Context, msg string, opts ...interface{}) {
	zerolog.Ctx(ctx).Warn().Msg(fmt.Sprintf(msg, opts...))
}

func (l ZeroLogAdapter) Info(ctx context.Context, msg string, opts ...interface{}) {
	zerolog.Ctx(ctx).Info().Msg(fmt.Sprintf(msg, opts...))
}

// Trace logs the execution of a SQL query using zerolog.
// It logs the duration of the query execution and additional information such as the SQL statement and the number of rows affected.
//
// Parameters:
//   - ctx: The context for the logging operation.
//   - begin: The time when the query execution started.
//   - f: A function that returns the SQL statement and the number of rows affected.
//   - err: An error that occurred during the query execution, if any.
func (l ZeroLogAdapter) Trace(ctx context.Context, begin time.Time, f func() (string, int64), err error) {
	zl := zerolog.Ctx(ctx)
	var event *zerolog.Event

	if err != nil {
		event = zl.Debug()
	} else {
		event = zl.Trace()
	}

	var durKey string

	switch zerolog.DurationFieldUnit {
	case time.Nanosecond:
		durKey = "elapsed_ns"
	case time.Microsecond:
		durKey = "elapsed_us"
	case time.Millisecond:
		durKey = "elapsed_ms"
	case time.Second:
		durKey = "elapsed"
	case time.Minute:
		durKey = "elapsed_min"
	case time.Hour:
		durKey = "elapsed_hr"
	default:
		zl.Error().Interface("zerolog.DurationFieldUnit", zerolog.DurationFieldUnit).Msg("gormzerolog encountered a mysterious, unknown value for DurationFieldUnit")
		durKey = "elapsed_"
	}

	event.Dur(durKey, time.Since(begin))

	sql, rows := f()
	if sql != "" {
		event.Str("sql", sql)
	}
	if rows > -1 {
		event.Int64("rows", rows)
	}

	event.Send()
}
