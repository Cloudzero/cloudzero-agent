// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/cloudzero/cloudzero-insights-controller/app/types"
	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"
)

const (
	// FileCreated event type
	FileCreated = "file_created"
	// FileChanged event type
	FileChanged = "file_changed"
	// FileDeleted event type
	FileDeleted = "file_deleted"
	// FileRenamed event type
	FileRenamed = "file_rename"
)

type FileMonitor struct {
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.Mutex

	filePath string
	watcher  *fsnotify.Watcher
	running  bool
	bus      types.Bus
}

// This structure is responsible for monitoring changes in the secret file,
// and will notify components in the application that depend on the secret
// filePath can be a directory or a single file
func NewFileMonitor(ctx context.Context, bus types.Bus, filePath string) (*FileMonitor, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if err := watcher.Add(filePath); err != nil {
		return nil, fmt.Errorf("failed to watch file: %w", err)
	}

	ctx, cancel := context.WithCancel(ctx)
	return &FileMonitor{
		ctx:      ctx,
		cancel:   cancel,
		filePath: filePath,
		watcher:  watcher,
		bus:      bus,
	}, nil
}

func (m *FileMonitor) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.running {
		return
	}

	// Start listening for events.
	go func() {
		for {
			select {
			case <-m.ctx.Done():
				m.mu.Lock()
				defer m.mu.Unlock()
				m.running = false
				m.ctx = nil
				m.cancel = nil
				return
			case event, ok := <-m.watcher.Events:
				if !ok {
					return
				}

				switch {
				// For example "mv /tmp/file /tmp/rename" will emit:
				//
				//   Event{Op: Rename, Name: "/tmp/file"}
				//   Event{Op: Create, Name: "/tmp/rename", RenamedFrom: "/tmp/file"}
				case event.Has(fsnotify.Create):
					m.bus.Publish(
						types.Event{
							Type: FileCreated,
							Value: types.FileCreated{
								Name: event.Name,
							},
						},
					)
				case event.Has(fsnotify.Rename):
					m.bus.Publish(
						types.Event{
							Type: FileRenamed,
							Value: types.FileRenamed{
								Name: event.Name,
							},
						},
					)

				case event.Has(fsnotify.Write):
					m.bus.Publish(
						types.Event{
							Type: FileChanged,
							Value: types.FileChanged{
								Name: event.Name,
							},
						},
					)

				case event.Has(fsnotify.Remove):
					m.bus.Publish(
						types.Event{
							Type: FileDeleted,
							Value: types.FileDeleted{
								Name: event.Name,
							},
						},
					)

				}

			case _, ok := <-m.watcher.Errors:
				if !ok {
					log.Ctx(m.ctx).Error().Str("path", m.filePath).Msg("error watching file - stopping watcher")
					m.cancel()
				}
			}
		}
	}()
	m.running = true
}

func (m *FileMonitor) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.watcher != nil {
		m.watcher.Close()
	}
	if m.ctx != nil {
		m.cancel()
	}
}
