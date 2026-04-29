// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package streaming provides a ResourceStore implementation that sends records
// directly to a remote_write endpoint instead of persisting them. Designed for
// the backfill job where the SQLite store is unnecessary overhead.
package streaming

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	config "github.com/cloudzero/cloudzero-agent/app/config/webhook"
	"github.com/cloudzero/cloudzero-agent/app/domain/pusher"
	"github.com/cloudzero/cloudzero-agent/app/types"
)

// Store implements types.ResourceStore by buffering records in memory and
// flushing them to the collector via remote_write when a batch threshold
// is reached. It does not persist data to disk or support queries.
const defaultBatchRecords = 500

type Store struct {
	settings      *config.Settings
	mu            sync.Mutex
	batch         []*types.ResourceTags
	maxBatchCount int
	maxRetries    int
	sendTimeout   time.Duration
}

// New creates a streaming store that sends records directly to the
// collector endpoint configured in settings.RemoteWrite.
func New(settings *config.Settings) *Store {
	return &Store{
		settings:      settings,
		batch:         make([]*types.ResourceTags, 0, defaultBatchRecords),
		maxBatchCount: defaultBatchRecords,
		maxRetries:    settings.RemoteWrite.MaxRetries,
		sendTimeout:   settings.RemoteWrite.SendTimeout,
	}
}

func (s *Store) Create(_ context.Context, record *types.ResourceTags) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.batch = append(s.batch, record)

	if len(s.batch) >= s.maxBatchCount {
		return s.flushLocked()
	}
	return nil
}

func (s *Store) Update(ctx context.Context, record *types.ResourceTags) error {
	return s.Create(ctx, record)
}

// Flush sends any buffered records to the collector.
func (s *Store) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.flushLocked()
}

func (s *Store) flushLocked() error {
	if len(s.batch) == 0 {
		return nil
	}

	ts := pusher.FormatMetrics(s.batch)
	log.Info().
		Int("records", len(s.batch)).
		Int("timeseries", len(ts)).
		Msg("streaming store: flushing batch to collector")

	err := pusher.PushMetrics(
		context.Background(),
		s.settings.RemoteWrite.Host,
		s.settings.GetAPIKey(),
		ts,
		s.maxRetries,
		s.settings.RemoteWrite.SendTimeout,
	)

	// Always clear the batch — if the send fails, the data is lost, which
	// is acceptable for backfill (it will be rediscovered on the next run).
	s.batch = s.batch[:0]

	if err != nil {
		return fmt.Errorf("streaming store flush: %w", err)
	}
	return nil
}

// The remaining ResourceStore methods are no-ops or return not-found. The
// backfill path writes records but never queries them.

func (s *Store) Tx(_ context.Context, block func(context.Context) error) error {
	return block(context.Background())
}

func (s *Store) Get(_ context.Context, _ string) (*types.ResourceTags, error) {
	return nil, types.ErrNotFound
}

func (s *Store) Delete(_ context.Context, _ string) error { return nil }
func (s *Store) Count(_ context.Context) (int, error)     { return 0, nil }
func (s *Store) DeleteAll(_ context.Context) error        { return nil }

func (s *Store) FindFirstBy(_ context.Context, _ ...interface{}) (*types.ResourceTags, error) {
	return nil, types.ErrNotFound
}

func (s *Store) FindAllBy(_ context.Context, _ ...interface{}) ([]*types.ResourceTags, error) {
	return nil, nil
}
