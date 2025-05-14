// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package status contains generated code for reading and writing the ClusterStatus protobuf.
package status

import (
	"sync"
)

var _ interface {
	Accessor
} = (*builder)(nil) // integrity check

// Accessor allows for low-level access to the Report
type Accessor interface {
	// AddCheck adds a check to the report.
	AddCheck(...*StatusCheck)
	// WriteToReport writes to the report.
	WriteToReport(func(*ClusterStatus))
	// ReadFromReport reads from the report.
	ReadFromReport(func(*ClusterStatus))
}

type builder struct {
	report  *ClusterStatus
	lock    *sync.RWMutex
	onWrite []func(*ClusterStatus)
}

func NewAccessor(s *ClusterStatus, onWrite ...func(*ClusterStatus)) Accessor {
	return &builder{
		report:  s,
		lock:    &sync.RWMutex{},
		onWrite: onWrite,
	}
}

func (b builder) onWriteEvent() {
	for _, fn := range b.onWrite {
		fn(b.report)
	}
}

func (b builder) WriteToReport(fn func(*ClusterStatus)) {
	b.lock.Lock()
	defer b.lock.Unlock()

	fn(b.report)
	b.onWriteEvent()
}

func (b builder) ReadFromReport(fn func(*ClusterStatus)) {
	b.lock.RLock()
	defer b.lock.RUnlock()

	fn(b.report)
}

func (b builder) AddCheck(c ...*StatusCheck) {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.report.Checks = append(b.report.Checks, c...)
	b.onWriteEvent()
}
