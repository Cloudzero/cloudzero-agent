// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0
package logging

import (
	"os"

	"github.com/sirupsen/logrus"
)

type LogFormat string

const (
	LogFormatJSON         LogFormat = "json"
	LogFormatText         LogFormat = "text"
	LogFormatTextColorful LogFormat = "text:colorful"

	timestampFormat = "15:04:05.000" // Not printing the date to save some space

	DefaultLogLevel = "info"
)

func SetUpLogging(level string, format LogFormat) {
	logrus.SetLevel(LogLevel(level))

	var formatter logrus.Formatter
	switch format {
	case LogFormatText:
		formatter = &PlainTextFormatter{
			DisableQuote:    true,
			PadLevelText:    true,
			FullTimestamp:   true,
			TimestampFormat: timestampFormat,
		}

	case LogFormatTextColorful:
		formatter = &logrus.TextFormatter{
			ForceColors:     format == LogFormatTextColorful,
			DisableColors:   format != LogFormatTextColorful,
			DisableQuote:    true,
			PadLevelText:    true,
			FullTimestamp:   true,
			TimestampFormat: timestampFormat,
		}

	case LogFormatJSON:
		formatter = &logrus.JSONFormatter{}

	default:
		logrus.Warnf("Unknown log format (using JSONFomatter): %s", format)
		formatter = &logrus.JSONFormatter{}
	}

	// wrap in sequence logger, so we always have monotonically increasing sequence number
	logrus.SetFormatter(NewSequenceLogger(formatter))
}

func LogLevel(name string) logrus.Level {
	switch name {
	case "panic":
		return logrus.PanicLevel
	case "fatal":
		return logrus.FatalLevel
	case "error":
		return logrus.ErrorLevel
	case "warn":
		return logrus.WarnLevel
	case "info":
		return logrus.InfoLevel
	case "debug":
		return logrus.DebugLevel
	case "trace":
		return logrus.TraceLevel
	default:
		return logrus.InfoLevel
	}
}

func NewLogger() *logrus.Logger {
	return logrus.StandardLogger()
}

func LogToFile(file string) *logrus.Logger {
	logger := NewLogger()
	logger.Out = os.Stdout
	if f, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o666); err == nil {
		logger.Out = f
	}
	return logger
}
