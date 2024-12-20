// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package housekeeper provides a mechanism for cleaning up stale data in a resource store.
// It periodically checks for and removes records that are older than a specified retention time.
package housekeeper

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/types"
	"github.com/cloudzero/cloudzero-insights-controller/pkg/utils"
)

type HouseKeeper struct {
	store           types.ResourceStore
	running         bool
	originalCtx     context.Context
	ctx             context.Context
	cancel          context.CancelFunc
	mu              sync.Mutex
	cleanupInterval time.Duration
	retentionTime   time.Duration
	clock           types.TimeProvider
	done            chan struct{}
}

func New(
	ctx context.Context,
	store types.ResourceStore,
	clock types.TimeProvider,
	settings *config.Settings,
) types.Runnable {
	newCtx, cancel := context.WithCancel(ctx)
	return &HouseKeeper{
		originalCtx:     ctx,
		ctx:             newCtx,
		cancel:          cancel,
		done:            make(chan struct{}),
		clock:           clock,
		store:           store,
		cleanupInterval: settings.Database.CleanupInterval,
		retentionTime:   settings.Database.RetentionTime,
	}
}

func (h *HouseKeeper) Start() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.running {
		return nil
	}

	ticker := time.NewTicker(h.cleanupInterval)
	go func() {
		defer ticker.Stop()
		defer close(h.done)
		defer func() {
			if r := recover(); r != nil {
				log.Info().
					Interface("panic", r).
					Msg("Recovered from panic in stale data removal")
			}
		}()
		for {
			select {
			case <-h.ctx.Done():
				h.running = false
				return
			case <-ticker.C:
				// use the store to cleanup old data
				currentTime := h.clock.GetCurrentTime()
				retentionTime := currentTime.Add(-1 * h.retentionTime)
				log.Debug().
					Dur("retention_time", h.retentionTime).
					Msg("Starting data purge process for stale records")
				expired, err := h.store.FindAllBy(h.ctx,
					fmt.Sprintf("sent_at < '%[1]s' AND record_created < '%[1]s' AND record_updated < '%[1]s' AND sent_at IS NOT NULL", utils.FormatForStorage(retentionTime)),
				)
				if err != nil {
					log.Error().
						Err(err).
						Msg("Failed to delete old tag data")
					continue // keep trying
				}

				expiredLen := len(expired)

				// avoid transaction if no records to delete
				if expiredLen == 0 {
					continue
				}

				// open a transaction to delete the records (will auto rollback on error)
				if err := h.store.Tx(h.ctx, func(txCtx context.Context) error {
					for _, record := range expired {
						if err := h.store.Delete(txCtx, record.ID); err != nil {
							return err
						}
					}
					log.Debug().
						Int("deleted_count", expiredLen).
						Msg("Deleted old records")
					return nil // commit the transaction
				}); err != nil {
					log.Err(err).Msg("Failed to delete old tag data")
				}
			}
		}
	}()
	h.running = true
	return nil
}

func (h *HouseKeeper) Shutdown() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if !h.running {
		return nil
	}
	h.cancel()
	<-h.done
	h.reset()
	return nil
}

func (h *HouseKeeper) reset() {
	h.running = false
	ctx, cancel := context.WithCancel(h.originalCtx)
	h.ctx = ctx
	h.cancel = cancel
	h.done = make(chan struct{})
}

func (h *HouseKeeper) IsRunning() bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.running
}
