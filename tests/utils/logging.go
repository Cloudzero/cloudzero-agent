// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package utils

// This file provides utilities meant for use in tests that want to verify log output

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

type LogCapture struct {
	Lines []string
	lock  sync.Mutex
}

// NewLogCapture sets up plain text logging to be captured for testing purposes
func NewLogCapture(logger *logrus.Logger) *LogCapture {
	// override any logging already setup to use plain text formatter
	formatter := &logrus.TextFormatter{
		DisableColors: true,
		DisableQuote:  false,
	}
	logrus.SetFormatter(formatter)
	return NewLogCaptureWithCurrentFormatter(logger)
}

// NewLogCaptureWithCurrentFormatter sets up capture with current formatter
func NewLogCaptureWithCurrentFormatter(logger *logrus.Logger) *LogCapture {
	capture := &LogCapture{}
	logger.SetOutput(capture)
	return capture
}

// Clear clears captured lines
func (l *LogCapture) Clear() {
	l.lock.Lock()
	defer l.lock.Unlock()

	l.Lines = nil
}

// Write echos line to console and captures same line
func (l *LogCapture) Write(p []byte) (n int, err error) {
	l.lock.Lock()
	defer l.lock.Unlock()

	msg := string(p)
	fmt.Print(msg)
	l.Lines = append(l.Lines, msg)
	return len(p), err
}

// Len returns number of log lines captured
func (l *LogCapture) Len() int {
	l.lock.Lock()
	defer l.lock.Unlock()

	return len(l.Lines)
}

// Extract looks for a key=value pair for line at given index, returning the value or empty if not found.
// This is possible because we log using plain text logging.
func (l *LogCapture) Extract(index int, key string) string {
	l.lock.Lock()
	defer l.lock.Unlock()

	line := l.Lines[index]
	// look for key=value and key="some stuff, in quotes"
	regexPattern := key + `=("[^"]*"|[\w\d\p{P}]+)`

	re := regexp.MustCompile(regexPattern)
	matches := re.FindStringSubmatch(line)

	if len(matches) > 1 {
		value := matches[1]
		// strip leading/trailing quotes if there
		if strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
			value = value[1 : len(value)-1]
		}
		return value
	}
	return ""
}
