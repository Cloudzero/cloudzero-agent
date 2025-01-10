// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package logging

import (
	"sync/atomic"

	"github.com/sirupsen/logrus"
)

const (
	LogSequence = "log_sequence"
)

// SequenceLogger adds a monotonically increasing number to each log entry, to allow
// ordering when time to the millisecond is identical (a limitation of AWS CloudWatch's
// @timestamp field). In CloudWatch query, can add a secondary sort key like so:
//
//	| sort @timestamp desc, log_sequence desc
type SequenceLogger struct {
	wrapped logrus.Formatter
	num     uint64
}

func NewSequenceLogger(wrap logrus.Formatter) *SequenceLogger {
	return &SequenceLogger{wrapped: wrap}
}

func (f *SequenceLogger) Format(entry *logrus.Entry) ([]byte, error) {
	seqNum := atomic.AddUint64(&f.num, 1)
	entry.Data[LogSequence] = seqNum
	return f.wrapped.Format(entry)
}
